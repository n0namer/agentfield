"""Live localhost release-gate proof across the Python and Go SDKs.

The test submits exactly one harmless async execution with the Python SDK to a
registry-confirmed Go SDK node. It then verifies durable identifier stability,
foreign-owner read denial, fresh-client readback, and terminal mutation denial.
It intentionally fails with a precise assertion when the required controlled
localhost fixture is not provisioned; it never substitutes mocks for live proof.
"""

from __future__ import annotations

import json
import os
import time
import uuid
from typing import Any

import pytest
import requests

from agentfield.async_config import AsyncConfig
from agentfield.client import AgentFieldClient


CONTROL_PLANE = os.getenv(
    "AGENTFIELD_RELEASEGATE_URL",
    os.getenv("AGENTFIELD_CONTROL_PLANE_URL", "http://127.0.0.1:8080"),
).rstrip("/")
GO_NODE = os.getenv("AGENTFIELD_RELEASEGATE_GO_NODE", "harness-duo-go")
GO_REASONER = os.getenv("AGENTFIELD_RELEASEGATE_GO_REASONER", "compare")
OWNER_API_KEY = os.getenv("AGENTFIELD_API_KEY", "")
FOREIGN_API_KEY = os.getenv("AGENTFIELD_RELEASEGATE_FOREIGN_API_KEY", "")
REQUEST_TIMEOUT = float(os.getenv("AGENTFIELD_RELEASEGATE_REQUEST_TIMEOUT", "10"))
POLL_TIMEOUT = float(os.getenv("AGENTFIELD_RELEASEGATE_POLL_TIMEOUT", "30"))


def _headers(api_key: str, **extra: str) -> dict[str, str]:
    headers = {"accept": "application/json", **extra}
    if api_key:
        headers["X-API-Key"] = api_key
    return headers


def _request_json(
    method: str,
    path: str,
    *,
    api_key: str = OWNER_API_KEY,
    body: dict[str, Any] | None = None,
) -> tuple[int, Any]:
    response = requests.request(
        method,
        f"{CONTROL_PLANE}{path}",
        headers=_headers(api_key, **({"content-type": "application/json"} if body is not None else {})),
        data=None if body is None else json.dumps(body, separators=(",", ":")),
        timeout=REQUEST_TIMEOUT,
    )
    try:
        payload: Any = response.json() if response.content else {}
    except ValueError:
        payload = response.text
    return response.status_code, payload


def _find_node(value: Any, node_id: str) -> dict[str, Any] | None:
    if isinstance(value, dict):
        if any(
            str(value.get(name, "")) == node_id
            for name in ("node_id", "id", "agent_id", "name")
        ):
            return value
        for item in value.values():
            found = _find_node(item, node_id)
            if found is not None:
                return found
    elif isinstance(value, list):
        for item in value:
            found = _find_node(item, node_id)
            if found is not None:
                return found
    return None


def _registry_go_node() -> tuple[str, dict[str, Any]]:
    failures: list[str] = []
    for path in ("/api/v1/nodes", "/api/v1/agents", "/api/v1/registry", "/api/v1/discover"):
        status, payload = _request_json("GET", path)
        node = _find_node(payload, GO_NODE) if 200 <= status < 300 else None
        if node is not None:
            marker_fields = {
                str(key).lower(): str(item).lower()
                for key, item in node.items()
                if key.lower() in {"sdk", "language", "runtime", "framework", "implementation"}
            }
            assert any("go" in item for item in marker_fields.values()), (
                f"registered target {GO_NODE!r} lacks explicit Go SDK/runtime metadata: {node!r}"
            )
            return path, node
        failures.append(f"{path}={status}")
    raise AssertionError(
        f"registered Go SDK node {GO_NODE!r} not found; registry probes: {', '.join(failures)}"
    )


def _execution_identity(payload: Any) -> tuple[str, str, str]:
    assert isinstance(payload, dict), f"execution readback is not an object: {payload!r}"
    execution_id = str(payload.get("execution_id") or payload.get("id") or "")
    run_id = str(payload.get("run_id") or "")
    status = str(payload.get("status") or "").lower()
    assert execution_id, f"execution readback lacks execution_id: {payload!r}"
    assert run_id, f"execution readback lacks run_id: {payload!r}"
    assert status, f"execution readback lacks status: {payload!r}"
    return execution_id, run_id, status


def _terminal_mutation_paths(openapi: Any, token: str) -> list[str]:
    if not isinstance(openapi, dict) or not isinstance(openapi.get("paths"), dict):
        return []
    paths: list[str] = []
    for path, methods in openapi["paths"].items():
        normalized = path.lower()
        if token not in normalized or "execution" not in normalized:
            continue
        if "{execution_id}" not in path and "{id}" not in path:
            continue
        if isinstance(methods, dict) and "post" in {str(method).lower() for method in methods}:
            paths.append(path)
    return paths


def _assert_terminal_mutation_denied(
    token: str,
    openapi: Any,
    execution_id: str,
    canonical: dict[str, Any],
) -> None:
    paths = _terminal_mutation_paths(openapi, token)
    assert paths, f"OpenAPI exposes no POST execution {token} route"
    for template in paths:
        path = template.replace("{execution_id}", execution_id).replace("{id}", execution_id)
        status, payload = _request_json("POST", path, body={"reason": "release-gate terminal denial probe"})
        assert 400 <= status < 500, (
            f"terminal execution {token} was not strictly denied at {path}: HTTP {status}: {payload!r}"
        )
        read_status, readback = _request_json("GET", f"/api/v1/executions/{execution_id}")
        assert read_status == 200, f"readback after denied {token} failed: {read_status}: {readback!r}"
        assert readback == canonical, f"denied {token} mutated persisted terminal execution"


@pytest.mark.live
@pytest.mark.timeout(90)
def test_releasegate_live_cross_sdk_ids_isolation_restart_and_terminal_denials() -> None:
    """Prove one harmless Python-to-Go async execution survives strict boundaries."""
    assert OWNER_API_KEY, "AGENTFIELD_API_KEY is required for owner-bound live evidence"
    assert FOREIGN_API_KEY, (
        "AGENTFIELD_RELEASEGATE_FOREIGN_API_KEY is required to prove isolation between two valid owners"
    )
    assert FOREIGN_API_KEY != OWNER_API_KEY, "foreign owner API key must differ from owner API key"

    health_status, health = _request_json("GET", "/health")
    assert health_status == 200, f"control plane health failed: HTTP {health_status}: {health!r}"

    registry_path, go_node = _registry_go_node()
    assert registry_path.startswith("/api/v1/")
    assert go_node

    run_id = f"releasegate-run-{uuid.uuid4()}"
    owner_id = f"releasegate-owner-{uuid.uuid4()}"
    request_id = f"releasegate-request-{uuid.uuid4()}"
    target = f"{GO_NODE}.{GO_REASONER}"
    input_data = {
        "task": "Return a deterministic acknowledgement only; do not modify files or call external services.",
        "request_id": request_id,
        "left": "release-gate",
        "right": "release-gate",
    }
    async_config = AsyncConfig.from_environment()
    async_config.initial_poll_interval = 0.25
    async_config.max_poll_interval = 1.0
    async_config.polling_timeout = REQUEST_TIMEOUT
    async_config.max_execution_timeout = POLL_TIMEOUT
    client = AgentFieldClient(
        base_url=CONTROL_PLANE,
        api_key=OWNER_API_KEY,
        async_config=async_config,
    )
    execution_headers = client._prepare_execution_headers(
        {
            "X-Run-ID": run_id,
            "X-Actor-ID": owner_id,
            "X-Caller-Agent-ID": owner_id,
            "Idempotency-Key": request_id,
        }
    )
    submission = client._submit_execution_sync(target, input_data, execution_headers)
    assert submission.execution_id
    assert submission.run_id == run_id, (
        f"Python SDK submission changed requested run_id: {submission.run_id!r} != {run_id!r}"
    )

    terminal = client._await_execution_sync(submission, execution_headers)
    execution_id, persisted_run_id, terminal_status = _execution_identity(terminal)
    assert execution_id == submission.execution_id, "control-plane persistence changed execution_id"
    assert persisted_run_id == submission.run_id, "control-plane persistence changed run_id"
    assert terminal_status in {"succeeded", "completed", "success"}, (
        f"Go SDK target did not complete successfully: {terminal!r}"
    )
    assert request_id in json.dumps(terminal, sort_keys=True), (
        f"persisted result lost the harmless request correlation: {terminal!r}"
    )

    read_status, canonical = _request_json("GET", f"/api/v1/executions/{execution_id}")
    assert read_status == 200, f"canonical readback failed: HTTP {read_status}: {canonical!r}"
    assert canonical == terminal, "first persisted readback differs from Python SDK terminal payload"

    repeat_status, repeated = _request_json("GET", f"/api/v1/executions/{execution_id}")
    assert repeat_status == 200
    assert repeated == canonical, "repeat readback was not byte-structure stable"

    restarted_client = AgentFieldClient(base_url=CONTROL_PLANE, api_key=OWNER_API_KEY)
    restart_headers = restarted_client._prepare_execution_headers(
        {"X-Actor-ID": owner_id, "X-Caller-Agent-ID": owner_id}
    )
    restart_response = requests.get(
        f"{restarted_client.api_base}/executions/{execution_id}",
        headers=restart_headers,
        timeout=REQUEST_TIMEOUT,
    )
    assert restart_response.status_code == 200, (
        f"fresh Python SDK client could not read persisted execution: {restart_response.status_code}: "
        f"{restart_response.text}"
    )
    assert restart_response.json() == canonical, "fresh-client readback changed persisted execution"

    foreign_status, foreign_payload = _request_json(
        "GET",
        f"/api/v1/executions/{execution_id}",
        api_key=FOREIGN_API_KEY,
    )
    assert foreign_status in {401, 403, 404}, (
        f"foreign owner read was not denied: HTTP {foreign_status}: {foreign_payload!r}"
    )
    assert request_id not in json.dumps(foreign_payload, sort_keys=True), (
        "foreign-owner denial leaked the execution request correlation"
    )

    openapi_status, openapi = _request_json("GET", "/openapi.json")
    assert openapi_status == 200, f"OpenAPI lookup failed: HTTP {openapi_status}: {openapi!r}"
    _assert_terminal_mutation_denied("cancel", openapi, execution_id, canonical)
    _assert_terminal_mutation_denied("timeout", openapi, execution_id, canonical)

    final_status, final_readback = _request_json("GET", f"/api/v1/executions/{execution_id}")
    assert final_status == 200
    assert final_readback == canonical
    final_execution_id, final_run_id, _ = _execution_identity(final_readback)
    assert final_execution_id == submission.execution_id
    assert final_run_id == submission.run_id

    time.sleep(0.05)
