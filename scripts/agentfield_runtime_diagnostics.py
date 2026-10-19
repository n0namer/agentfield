#!/usr/bin/env python3
"""Read-only AgentField anomaly collection using existing native APIs.

Only classification metadata is persisted, never execution payloads or log lines.
The output is local JSONL; it is NOT an external notification integration.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener

MAX_BODY_BYTES = 2_000_000
MAX_PAGES = 3
PAGE_SIZE = 100
NODE_TAIL = 200
SAFE_ID = re.compile(r"^[\w.:-]{1,180}$", re.ASCII)


class DiagnosticsError(Exception):
    """A safe, non-secret diagnostic error."""


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise DiagnosticsError("Unexpected API redirect")


def checked_url(url: str) -> str:
    parsed = urlsplit(url)
    if parsed.scheme != "http" or parsed.hostname not in {"127.0.0.1", "localhost", "::1"}:
        raise DiagnosticsError("Only local HTTP AgentField APIs are allowed")
    if parsed.username or parsed.password or parsed.path not in {"", "/"} or parsed.query or parsed.fragment:
        raise DiagnosticsError("Expected a local API origin without credentials or path")
    return url.rstrip("/")


def safe_id(value: object) -> str | None:
    return value if isinstance(value, str) and SAFE_ID.fullmatch(value) else None


class NativeClient:
    def __init__(self, origin: str, api_key: str = "", timeout: float = 5):
        self.origin = checked_url(origin)
        self.timeout = timeout
        self.headers = {"Accept": "application/json"}
        if api_key:
            self.headers["X-API-Key"] = api_key
        self.opener = build_opener(NoRedirect())

    def _get(self, path: str) -> bytes:
        if not path.startswith("/") or path.startswith("//"):
            raise DiagnosticsError("Invalid API path")
        request = Request(self.origin + path, headers=self.headers, method="GET")
        try:
            with self.opener.open(request, timeout=self.timeout) as response:
                data = response.read(MAX_BODY_BYTES + 1)
        except (HTTPError, URLError, TimeoutError, OSError) as exc:
            # Never emit a raw exception: its URL/headers may contain credentials.
            code = exc.code if isinstance(exc, HTTPError) else "network"
            raise DiagnosticsError(f"AgentField API unavailable ({code})") from None
        if len(data) > MAX_BODY_BYTES:
            raise DiagnosticsError("AgentField API response exceeds size limit")
        return data

    def json(self, path: str) -> dict:
        try:
            value = json.loads(self._get(path))
        except (ValueError, UnicodeDecodeError):
            raise DiagnosticsError("Invalid AgentField JSON") from None
        if not isinstance(value, dict):
            raise DiagnosticsError("Unexpected AgentField response")
        return value

    def ndjson(self, path: str) -> list[dict]:
        rows = []
        for line in self._get(path).splitlines():
            try:
                row = json.loads(line)
            except (ValueError, UnicodeDecodeError):
                continue
            if isinstance(row, dict):
                rows.append(row)
        return rows


def parse_started(raw: object) -> datetime | None:
    if not isinstance(raw, str):
        return None
    try:
        parsed = datetime.fromisoformat(raw.replace("Z", "+00:00"))
    except ValueError:
        return None
    return parsed.astimezone(timezone.utc) if parsed.tzinfo else None


def classify_node(line: str, level: str) -> str | None:
    text = line.casefold()
    if "max reconnection attempts" in text or "retries exhausted" in text:
        return "node_retries_exhausted"
    if "winerror 1225" in text or "connection refused" in text or "failed to connect" in text:
        return "node_connection_failed"
    if re.search(r"\b429\b|rate.?limit", text):
        return "node_rate_limited"
    if "timed out" in text or "timeout" in text:
        return "node_timeout"
    if level.casefold() in {"error", "fatal", "critical"}:
        return "node_runtime_error"
    return None


def make_alert(kind: str, entity: str, *, run_id: str = "", execution_id: str = "",
               node_id: str = "", seq: int | None = None, event_time: str = "") -> dict:
    # Agent-local seq resets on process restart. The log timestamp differentiates generations.
    key = f"{kind}:{entity}" + (f":{seq}:{event_time}" if seq is not None and event_time
                                else f":{seq}" if seq is not None else "")
    severity = "warning" if kind in {"node_rate_limited", "node_timeout", "stale_execution"} else "error"
    # Metadata only: raw log strings, inputs/outputs and Codex sessions are excluded.
    return {"key": key, "kind": kind, "severity": severity,
            "run_id": run_id, "execution_id": execution_id, "node_id": node_id,
            "log_seq": seq, "event_time": event_time or None,
            "ui_path": f"/ui/runs/{quote(run_id, safe='')}" if run_id else "/ui/agents"}


def collect(client, now: datetime, stale_seconds: int = 1800) -> list[dict]:
    alerts = []
    page = 1
    while page <= MAX_PAGES:
        result = client.json(f"/api/ui/v2/workflow-runs?page={page}&page_size={PAGE_SIZE}")
        runs = result.get("runs")
        if not isinstance(runs, list):
            raise DiagnosticsError("Missing run list")
        for run in runs:
            if not isinstance(run, dict):
                continue
            run_id = safe_id(run.get("run_id"))
            execution_id = safe_id(run.get("root_execution_id"))
            if not run_id or not execution_id:
                continue
            status = str(run.get("root_execution_status") or run.get("status") or "").lower()
            if status in {"failed", "timeout", "timed_out"}:
                alerts.append(make_alert("execution_timeout" if status.startswith("time") else "execution_failed",
                                         execution_id, run_id=run_id, execution_id=execution_id))
            if status == "running":
                # latest_activity is derived from LatestStarted, NOT a heartbeat.
                # Be fail-closed about "stalled" until an explicit progress signal is present.
                last_progress = parse_started(run.get("last_progress_at"))
                if last_progress and (now - last_progress).total_seconds() > stale_seconds:
                    alerts.append(make_alert("stale_execution", execution_id, run_id=run_id, execution_id=execution_id))
            if status == "succeeded":
                try:
                    execution = client.json(f"/api/v1/executions/{quote(execution_id, safe='')}")
                except DiagnosticsError:
                    # Do not report application failure for a missing/unavailable result.
                    continue
                application = execution.get("result")
                if isinstance(application, dict) and application.get("success") is False:
                    alerts.append(make_alert("application_failed", execution_id,
                                             run_id=run_id, execution_id=execution_id))
        if not result.get("has_more") or not runs:
            break
        page += 1

    nodes = client.json("/api/ui/v1/nodes/summary").get("nodes", [])
    if not isinstance(nodes, list):
        raise DiagnosticsError("Invalid nodes list")
    for node in nodes:
        if not isinstance(node, dict):
            continue
        node_id = safe_id(node.get("id"))
        if not node_id:
            continue
        try:
            lines = client.ndjson(f"/api/ui/v1/nodes/{quote(node_id, safe='')}/logs?tail_lines={NODE_TAIL}")
        except DiagnosticsError:
            alerts.append(make_alert("node_logs_unavailable", node_id, node_id=node_id))
            continue
        for row in lines:
            raw_line = row.get("line")
            if not isinstance(raw_line, str):
                continue
            kind = classify_node(raw_line, str(row.get("level") or ""))
            if kind is None:
                continue
            seq = row.get("seq")
            when = parse_started(row.get("ts"))
            if not isinstance(seq, int) or isinstance(seq, bool) or seq < 0 or when is None:
                # Cannot replay-fence without a real timestamp and sequence; never hash private lines.
                continue
            alerts.append(make_alert(kind, node_id, node_id=node_id, seq=seq,
                                     event_time=when.isoformat(timespec="microseconds")))
    return alerts


def publish_once(client, *, state_path: Path, alerts_path: Path, now: datetime,
                 stale_seconds: int = 1800, injected_alerts: list[dict] | None = None) -> tuple[int, int]:
    alerts = collect(client, now, stale_seconds) if injected_alerts is None else injected_alerts
    state_path.parent.mkdir(parents=True, exist_ok=True)
    lock_path = state_path.with_suffix(".lock")
    with lock_path.open("a+b") as lock_file:
        if not lock_file.seek(0, 2):
            lock_file.write(b"x")
            lock_file.flush()
        lock_file.seek(0)
        # One task per host: protect state and JSONL against concurrent invocations.
        try:
            if os.name == "nt":
                import msvcrt
                msvcrt.locking(lock_file.fileno(), msvcrt.LK_NBLCK, 1)
            else:
                import fcntl
                fcntl.flock(lock_file.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
        except OSError:
            raise DiagnosticsError("Another diagnostic scan holds the state lock") from None
        try:
            if state_path.exists():
                try:
                    state = json.loads(state_path.read_text(encoding="utf-8"))
                    prior = state["seen"]
                    if not isinstance(prior, list) or not all(isinstance(k, str) for k in prior):
                        raise ValueError("bad seen")
                    seen = set(prior)
                except (OSError, ValueError, KeyError, TypeError):
                    raise DiagnosticsError("Invalid state; preserve file before replay") from None
            else:
                seen = set()
            old_seen = set(seen)
            legacy_seen_at: dict[str, datetime] = {}
            # The alert journal is authoritative after a crash between append and cursor save.
            if alerts_path.exists():
                try:
                    if alerts_path.stat().st_size > 16_000_000:
                        raise ValueError("oversized journal")
                    with alerts_path.open(encoding="utf-8") as fh:
                        for line in fh:
                            if not line.strip():
                                continue
                            item = json.loads(line)
                            if not isinstance(item, dict) or not isinstance(item.get("key"), str):
                                raise ValueError("invalid journal item")
                            seen.add(item["key"])
                            detected = parse_started(item.get("observed_at"))
                            if detected:
                                legacy_seen_at[item["key"]] = detected
                except (ValueError, OSError, UnicodeError):
                    raise DiagnosticsError("Invalid alert journal; preserve file before replay") from None
            fresh = []
            for alert in alerts:
                if alert["key"] in seen:
                    continue
                # Migrate legacy kind:node:seq keys without re-alerting on historical
                # lines; later timestamps indicate a new node generation.
                if alert["node_id"] and alert["log_seq"] is not None and alert["event_time"]:
                    old_key = f'{alert["kind"]}:{alert["node_id"]}:{alert["log_seq"]}'
                    old_detected = legacy_seen_at.get(old_key)
                    source_time = parse_started(alert["event_time"])
                    if old_detected and source_time and source_time <= old_detected:
                        seen.add(alert["key"])
                        continue
                fresh.append(alert)
                seen.add(alert["key"])
            if fresh:
                alerts_path.parent.mkdir(parents=True, exist_ok=True)
                # A fully valid final JSONL record may lack LF after a crash.
                # Recover the separator before appending a new record.
                needs_separator = False
                if alerts_path.exists() and alerts_path.stat().st_size:
                    with alerts_path.open("rb") as existing:
                        existing.seek(-1, os.SEEK_END)
                        needs_separator = existing.read(1) != b"\n"
                with alerts_path.open("a", encoding="utf-8", newline="\n") as fh:
                    if needs_separator:
                        fh.write("\n")
                    for alert in fresh:
                        fh.write(json.dumps({"observed_at": now.isoformat(), **alert},
                                            ensure_ascii=False) + "\n")
                    fh.flush()
                    os.fsync(fh.fileno())
            if seen != old_seen:
                temp = state_path.with_suffix(state_path.suffix + ".tmp")
                temp.write_text(json.dumps({"seen": sorted(seen)}, ensure_ascii=False),
                                encoding="utf-8")
                os.replace(temp, state_path)
            return len(alerts), len(fresh)
        finally:
            if os.name == "nt":
                msvcrt.locking(lock_file.fileno(), msvcrt.LK_UNLCK, 1)
            else:
                fcntl.flock(lock_file.fileno(), fcntl.LOCK_UN)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Local read-only AgentField diagnostic collector")
    parser.add_argument("--url", default="http://127.0.0.1:18080")
    parser.add_argument("--state-dir", type=Path, default=Path.home() / ".agentfield" / "diagnostics")
    parser.add_argument("--stale-seconds", type=int, default=1800)
    parser.add_argument("--follow", action="store_true", help="Repeat collection every interval seconds")
    parser.add_argument("--interval", type=int, default=60)
    args = parser.parse_args(argv)
    if args.interval < 30 or args.stale_seconds < 60:
        parser.error("interval >= 30 and stale-seconds >= 60 are required")
    try:
        client = NativeClient(args.url, os.environ.get("AGENTFIELD_API_KEY", ""))
    except DiagnosticsError as err:
        print(f"diagnostics: {err}", file=sys.stderr)
        return 2
    while True:
        try:
            found, fresh = publish_once(client, state_path=args.state_dir / "state.json",
                                        alerts_path=args.state_dir / "alerts.jsonl",
                                        now=datetime.now(timezone.utc),
                                        stale_seconds=args.stale_seconds)
            print(json.dumps({"status": "ok", "findings": found, "new_alerts": fresh,
                              "alerts_file": str(args.state_dir / "alerts.jsonl")}, ensure_ascii=False), flush=True)
        except DiagnosticsError as err:
            failure_now = datetime.now(timezone.utc)
            # An unavailable control plane must itself leave a durable operator alert.
            # Hourly incident keys bound local journal growth during prolonged outages.
            try:
                failure_alert = make_alert("diagnostics_poll_failed",
                                           "agentfield:" + failure_now.strftime("%Y%m%d%H"))
                publish_once(None, state_path=args.state_dir / "state.json",
                             alerts_path=args.state_dir / "alerts.jsonl",
                             now=failure_now, injected_alerts=[failure_alert])
            except DiagnosticsError:
                # Preserve corrupt state/journal untouched and rely on non-zero exit.
                pass
            print(json.dumps({"status": "error", "category": str(err)}, ensure_ascii=False),
                  file=sys.stderr, flush=True)
            if not args.follow:
                return 2
        if not args.follow:
            return 0
        time.sleep(args.interval)


if __name__ == "__main__":
    raise SystemExit(main())
