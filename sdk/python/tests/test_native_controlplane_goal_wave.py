"""Live native control-plane goal lifecycle proof.

This test intentionally uses only loopback HTTP and the already-running
AgentField control plane/demo node. It is not a provider mock and must fail
loudly when the native runtime is unavailable.
"""

from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.request
from typing import Any

import pytest


CONTROL_PLANE = os.getenv("AGENTFIELD_CONTROL_PLANE_URL", "http://127.0.0.1:8080").rstrip("/")
NODE_ID = os.getenv("AGENTFIELD_NATIVE_NODE", "harness-duo-go")
NODE_FUNCTION = os.getenv("AGENTFIELD_NATIVE_FUNCTION", "compare")
POLL_SECONDS = float(os.getenv("AGENTFIELD_NATIVE_POLL_SECONDS", "30"))


def _request_json(
    method: str,
    path: str,
    *,
    body: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
    timeout: float = 10.0,
) -> tuple[int, Any, dict[str, str]]:
    payload = None if body is None else json.dumps(body).encode("utf-8")
    request_headers = {"accept": "application/json"}
    if payload is not None:
        request_headers["content-type"] = "application/json"
    if headers:
        request_headers.update(headers)
    request = urllib.request.Request(
        f"{CONTROL_PLANE}{path}",
        data=payload,
        headers=request_headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            raw = response.read().decode("utf-8")
            return response.status, (json.loads(raw) if raw else {}), dict(response.headers)
    except urllib.error.HTTPError as error:
        raw = error.read().decode("utf-8")
        try:
            payload_value: Any = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            payload_value = raw
        return error.code, payload_value, dict(error.headers)


def _contains(value: Any, needle: str) -> bool:
    if isinstance(value, str):
        return needle in value
    if isinstance(value, dict):
        return any(_contains(key, needle) or _contains(item, needle) for key, item in value.items())
    if isinstance(value, list):
        return any(_contains(item, needle) for item in value)
    return False


def _find_first(value: Any, names: set[str]) -> Any:
    if isinstance(value, dict):
        for key, item in value.items():
            if key.lower() in names and item not in (None, ""):
                return item
        for item in value.values():
            found = _find_first(item, names)
            if found not in (None, ""):
                return found
    elif isinstance(value, list):
        for item in value:
            found = _find_first(item, names)
            if found not in (None, ""):
                return found
    return None


def _registry_snapshot() -> tuple[str, Any]:
    candidates = (
        "/api/v1/nodes",
        "/api/v1/agents",
        "/api/v1/registry",
        "/api/v1/discover",
    )
    failures: list[str] = []
    for path in candidates:
        status, payload, _ = _request_json("GET", path)
        if 200 <= status < 300 and _contains(payload, NODE_ID):
            return path, payload
        failures.append(f"{path}={status}")
    raise AssertionError(
        f"registered node {NODE_ID!r} was not found; registry probes: {', '.join(failures)}"
    )


def _execution_id(payload: Any) -> str:
    value = _find_first(payload, {"execution_id", "executionid", "id"})
    assert value is not None, f"async submission had no execution identifier: {payload!r}"
    return str(value)


def _status(payload: Any) -> str:
    value = _find_first(payload, {"status", "state", "phase"})
    return str(value or "").lower()


def _is_terminal(status: str) -> bool:
    return any(token in status for token in ("complete", "success", "failed", "error", "cancel", "timeout"))


def _execution_readback(execution_id: str) -> Any:
    status, payload, _ = _request_json("GET", f"/api/v1/executions/{execution_id}")
    assert 200 <= status < 300, f"execution readback failed: HTTP {status}: {payload!r}"
    return payload


def test_native_controlplane_goal_wave_full_cycle_and_fail_closed() -> None:
    """Submit one real goal, read it back, retry idempotently, and probe failures."""
    health_status, health, _ = _request_json("GET", "/health")
    assert health_status == 200, f"control plane health failed: {health_status}: {health!r}"

    registry_path, registry = _registry_snapshot()
    assert _contains(registry, NODE_ID)

    request_id = f"native-goal-wave-{int(time.time())}"
    goal = {
        "input": {
            "task": "Return a short structured acknowledgement; do not modify files or call external services.",
            "request_id": request_id,
        }
    }
    idempotency_key = f"{request_id}-once"
    execute_path = f"/api/v1/execute/async/{NODE_ID}.{NODE_FUNCTION}"
    submit_status, submitted, _ = _request_json(
        "POST",
        execute_path,
        body=goal,
        headers={"Idempotency-Key": idempotency_key},
    )
    assert submit_status in (200, 201, 202), f"native submit failed: {submit_status}: {submitted!r}"
    execution_id = _execution_id(submitted)

    deadline = time.monotonic() + POLL_SECONDS
    latest = submitted
    while time.monotonic() < deadline:
        latest = _execution_readback(execution_id)
        if _is_terminal(_status(latest)):
            break
        time.sleep(0.25)
    final_status = _status(latest)
    assert any(token in final_status for token in ("complete", "success")), (
        f"native goal did not complete successfully: {latest!r}"
    )

    persisted = _execution_readback(execution_id)
    assert persisted == latest, "canonical execution readback was not stable"
    assert _contains(persisted, request_id)
    assert _find_first(persisted, {"run_id", "runid"}) is not None or _find_first(
        persisted, {"thread_id", "threadid"}
    ) is not None, f"execution lacked run/thread correlation: {persisted!r}"

    retry_status, retried, _ = _request_json(
        "POST",
        execute_path,
        body=goal,
        headers={"Idempotency-Key": idempotency_key},
    )
    assert retry_status in (200, 201, 202), f"idempotent retry failed: {retry_status}: {retried!r}"
    assert _execution_id(retried) == execution_id, "retry created a duplicate execution"

    invalid_status, invalid_payload, _ = _request_json(
        "POST",
        "/api/v1/execute/async/__native_goal_wave_missing__.noop",
        body={"input": {"request_id": f"{request_id}-invalid"}},
    )
    assert invalid_status >= 400, f"unknown node was accepted: {invalid_status}: {invalid_payload!r}"
    assert not _contains(invalid_payload, f"{request_id}-invalid"), "failed request leaked an effect"

    openapi_status, openapi, _ = _request_json("GET", "/openapi.json")
    if openapi_status == 200 and isinstance(openapi, dict):
        cancel_paths = [path for path in openapi.get("paths", {}) if "cancel" in path.lower()]
        if cancel_paths:
            cancel_status, cancel_payload, _ = _request_json("POST", cancel_paths[0].replace("{execution_id}", execution_id))
            assert cancel_status < 500, f"cancel route failed closed: {cancel_status}: {cancel_payload!r}"

    assert registry_path.startswith("/api/v1/")
