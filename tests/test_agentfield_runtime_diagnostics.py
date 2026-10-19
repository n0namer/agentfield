"""Read-only regression tests: no AgentField, Codex or SWE services are started."""
from datetime import datetime, timedelta, timezone
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "agentfield_runtime_diagnostics.py"
spec = importlib.util.spec_from_file_location("agentfield_runtime_diagnostics", SCRIPT)
diag = importlib.util.module_from_spec(spec)
spec.loader.exec_module(diag)
NOW = datetime(2026, 10, 19, 18, 0, tzinfo=timezone.utc)


class FakeClient:
    def __init__(self, runs=None, executions=None, nodes=None, logs=None):
        self.runs = runs if runs is not None else []
        self.executions = executions or {}
        self.nodes = nodes or []
        self.logs = logs or {}
        self.paths = []

    def json(self, path):
        self.paths.append(path)
        if path.startswith("/api/ui/v2/workflow-runs"):
            return {"runs": self.runs, "has_more": False}
        if path.startswith("/api/v1/executions/"):
            key = path.rsplit("/", 1)[-1]
            return self.executions.get(key, {})
        if path.endswith("/nodes/summary"):
            return {"nodes": [{"id": node} for node in self.nodes]}
        raise AssertionError(path)

    def ndjson(self, path):
        self.paths.append(path)
        node = path.split("/")[5]
        return self.logs.get(node, [])


def run(status="succeeded", eid="exec_test", started=None, runid="run_test"):
    return {"run_id": runid, "root_execution_id": eid,
            "root_execution_status": status,
            "started_at": started or (NOW - timedelta(hours=2)).isoformat()}


class DiagnosticsTests(unittest.TestCase):
    def test_false_green_from_actual_transport_and_application_status(self):
        client = FakeClient(runs=[run()], executions={"exec_test": {"status": "succeeded",
                                                                   "result": {"success": False, "secret": "PRIVATE"}}})
        alerts = diag.collect(client, NOW)
        self.assertEqual([a["kind"] for a in alerts], ["application_failed"])
        self.assertEqual(alerts[0]["ui_path"], "/ui/runs/run_test")
        self.assertNotIn("PRIVATE", json.dumps(alerts))

    def test_missing_application_result_is_unknown_not_failure(self):
        client = FakeClient(runs=[run()], executions={"exec_test": {"status": "succeeded"}})
        self.assertEqual(diag.collect(client, NOW), [])

    def test_failed_timeout_and_running_stale(self):
        client = FakeClient(runs=[run("failed", "bad"), run("timeout", "timed"),
                                  run("running", "stale"),
                                  run("running", "fresh", (NOW - timedelta(seconds=10)).isoformat())])
        kinds = {a["kind"] for a in diag.collect(client, NOW, stale_seconds=1800)}
        self.assertEqual(kinds, {"execution_failed", "execution_timeout", "stale_execution"})

    def test_node_errors_with_exact_dedupe_keys(self):
        client = FakeClient(nodes=["swe-planner"], logs={"swe-planner": [
            {"seq": 9, "level": "info", "line": "Failed to connect to memory events: WinError 1225 PRIVATE_SECRET"},
            {"seq": 10, "level": "info", "line": "Max reconnection attempts reached (5)"},
            {"seq": 11, "level": "warning", "line": "Provider returned 429 rate limit"},
            {"seq": 12, "level": "info", "line": "Connected to AgentField server"},
        ]})
        alarms = diag.collect(client, NOW)
        self.assertEqual([a["kind"] for a in alarms],
                         ["node_connection_failed", "node_retries_exhausted", "node_rate_limited"])
        self.assertTrue(all(a["node_id"] == "swe-planner" for a in alarms))
        self.assertNotIn("PRIVATE_SECRET", json.dumps(alarms))

    def test_durable_replay_once_and_restart_no_duplicate(self):
        client = FakeClient(runs=[run("failed")])
        with tempfile.TemporaryDirectory() as tmp:
            state = Path(tmp) / "state.json"
            alerts = Path(tmp) / "alerts.jsonl"
            self.assertEqual(diag.publish_once(client, state_path=state, alerts_path=alerts, now=NOW), (1, 1))
            self.assertEqual(diag.publish_once(FakeClient(runs=[run("failed")]),
                                               state_path=state, alerts_path=alerts, now=NOW), (1, 0))
            self.assertEqual(len(alerts.read_text(encoding="utf-8").splitlines()), 1)

    def test_corrupt_state_fail_closed_no_duplicates(self):
        with tempfile.TemporaryDirectory() as tmp:
            state = Path(tmp) / "state.json"
            alerts = Path(tmp) / "alerts.jsonl"
            state.write_text("not JSON", encoding="utf-8")
            with self.assertRaises(diag.DiagnosticsError):
                diag.publish_once(FakeClient(runs=[run("failed")]), state_path=state,
                                  alerts_path=alerts, now=NOW)
            self.assertFalse(alerts.exists())
            self.assertEqual(state.read_text(encoding="utf-8"), "not JSON")

    def test_local_origin_guard(self):
        for url in ["https://127.0.0.1:18080", "http://example.com:18080",
                    "http://127.0.0.1:18080/path", "http://user:password@localhost:18080"]:
            with self.subTest(url=url), self.assertRaises(diag.DiagnosticsError):
                diag.checked_url(url)
        self.assertEqual(diag.checked_url("http://127.0.0.1:18080"), "http://127.0.0.1:18080")

    def test_untrusted_ids_skipped(self):
        c = FakeClient(runs=[run(eid="a/b"), run(runid="../unsafe")],
                       nodes=["x/y"], logs={"x/y": [{"seq": 1, "level": "error", "line": "oops"}]})
        self.assertEqual(diag.collect(c, NOW), [])

    def test_node_log_error_no_secret_or_raw_exception(self):
        class Unavailable(FakeClient):
            def ndjson(self, path):
                raise diag.DiagnosticsError("remote failed")
        c = Unavailable(nodes=["swe-fast"])
        result = diag.collect(c, NOW)
        self.assertEqual(result[0]["kind"], "node_logs_unavailable")
        self.assertNotIn("remote failed", json.dumps(result))

    def test_invalid_timestamp_does_not_mark_stale(self):
        c = FakeClient(runs=[run("running", started="not-a-date")])
        self.assertEqual(diag.collect(c, NOW), [])

    def test_long_running_with_recent_activity_is_not_stale(self):
        active = run("running", started=(NOW - timedelta(hours=4)).isoformat())
        active["latest_activity"] = (NOW - timedelta(seconds=40)).isoformat()
        c = FakeClient(runs=[active])
        self.assertEqual(diag.collect(c, NOW, stale_seconds=1800), [])


    def test_crash_between_journal_append_and_cursor_save_recovered(self):
        c = FakeClient(runs=[run("failed")])
        with tempfile.TemporaryDirectory() as tmp:
            state = Path(tmp) / "state.json"
            journal = Path(tmp) / "alerts.jsonl"
            alert = diag.make_alert("execution_failed", "exec_test",
                                    run_id="run_test", execution_id="exec_test")
            journal.write_text(json.dumps({"observed_at": NOW.isoformat(), **alert}) + "\n",
                               encoding="utf-8")
            self.assertEqual(diag.publish_once(c, state_path=state, alerts_path=journal, now=NOW),
                             (1, 0))
            self.assertEqual(len(journal.read_text(encoding="utf-8").splitlines()), 1)
            self.assertIn("execution_failed:exec_test", state.read_text(encoding="utf-8"))

    def test_duplicate_rows_within_same_batch_emit_once(self):
        c = FakeClient(runs=[run("failed"), run("failed")])
        with tempfile.TemporaryDirectory() as tmp:
            state = Path(tmp) / "state.json"
            journal = Path(tmp) / "alerts.jsonl"
            self.assertEqual(diag.publish_once(c, state_path=state, alerts_path=journal, now=NOW),
                             (2, 1))
            self.assertEqual(len(journal.read_text(encoding="utf-8").splitlines()), 1)

    def test_missing_node_sequence_is_not_hashed_from_private_content(self):
        c = FakeClient(nodes=["swe-planner"], logs={"swe-planner": [
            {"level": "error", "line": "private_token_goes_here", "ts": "2026-10-19"},
            {"seq": 0, "level": "error", "line": "unexpected error"}]})
        alerts = diag.collect(c, NOW)
        self.assertEqual(len(alerts), 1)
        self.assertEqual(alerts[0]["log_seq"], 0)
        self.assertNotIn("private_token_goes_here", json.dumps(alerts))

    def test_invalid_journal_fails_closed(self):
        c = FakeClient(runs=[run("failed")])
        with tempfile.TemporaryDirectory() as tmp:
            state = Path(tmp) / "state.json"
            journal = Path(tmp) / "alerts.jsonl"
            journal.write_text("{partial", encoding="utf-8")
            with self.assertRaises(diag.DiagnosticsError):
                diag.publish_once(c, state_path=state, alerts_path=journal, now=NOW)
            self.assertEqual(journal.read_text(encoding="utf-8"), "{partial")


if __name__ == "__main__":
    unittest.main()
