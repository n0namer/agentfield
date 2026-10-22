"""Independent, non-destructive QA of the blocked cross-SDK release gate.

These tests challenge the producer artifacts without contacting a live service.
They verify immutable source evidence and exercise the producer's helper paths
with controlled doubles, including negative and crash boundaries.
"""

from __future__ import annotations

import hashlib
import importlib.util
from pathlib import Path
from types import ModuleType
from typing import Any

import pytest
import requests


ROOT = Path(__file__).resolve().parents[3]
PRODUCER_TEST = ROOT / "sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py"
PRODUCER_RECORD = ROOT / "docs/acceptance/releasegate-live-cross-sdk-20261013.md"
RECORDED_PRODUCER_SHA256 = "764cbaf681037405ffd15ddf82a6693ac0820e5b65a0e1ba1eb065478f5bed49"


def _text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


@pytest.fixture(scope="module")
def producer_module() -> ModuleType:
    spec = importlib.util.spec_from_file_location("releasegate_live_cross_sdk_producer", PRODUCER_TEST)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_recorded_producer_source_sha_matches_current_artifact() -> None:
    digest = hashlib.sha256(PRODUCER_TEST.read_bytes()).hexdigest()
    assert digest == RECORDED_PRODUCER_SHA256
    assert RECORDED_PRODUCER_SHA256 in _text(PRODUCER_RECORD)


def test_producer_record_is_fail_closed_and_contains_no_live_success_receipt() -> None:
    record = _text(PRODUCER_RECORD)
    assert "BLOCKED / UNACCEPTED" in record
    assert "PYTHON_VENV_UNAVAILABLE" in record
    assert "This attempt did not execute that request" in record
    assert "no claim is made that a live request" in record.lower()


def test_producer_record_lacks_required_process_and_coordination_attestation() -> None:
    record = _text(PRODUCER_RECORD)
    assert "Git HEAD" not in record
    assert "publish_worker_result" not in record
    assert "PENDING_SYNC" not in record
    assert "CONFLICT" not in record
    assert "exact original CrewSync status readback" not in record


def test_foreign_owner_probe_cannot_prove_the_second_credential_is_valid() -> None:
    source = _text(PRODUCER_TEST)
    assert "assert FOREIGN_API_KEY" in source
    assert "assert FOREIGN_API_KEY != OWNER_API_KEY" in source
    assert "foreign_status in {401, 403, 404}" in source
    assert "/api/v1/auth" not in source
    assert "/api/v1/me" not in source


def test_registry_probe_accepts_nested_echo_as_go_node_provenance(
    producer_module: ModuleType, monkeypatch: pytest.MonkeyPatch
) -> None:
    payload = {
        "fixture_echo": {
            "name": producer_module.GO_NODE,
            "runtime": "go-test-double",
        }
    }
    monkeypatch.setattr(producer_module, "_request_json", lambda *_args, **_kwargs: (200, payload))

    path, node = producer_module._registry_go_node()

    assert path == "/api/v1/nodes"
    assert node == payload["fixture_echo"]


def test_terminal_mutation_probe_accepts_route_or_auth_404_as_terminal_denial(
    producer_module: ModuleType, monkeypatch: pytest.MonkeyPatch
) -> None:
    canonical = {"execution_id": "exec-1", "run_id": "run-1", "status": "completed"}
    calls: list[tuple[str, str]] = []

    def fake_request(method: str, path: str, **_kwargs: Any) -> tuple[int, Any]:
        calls.append((method, path))
        if method == "POST":
            return 404, {"detail": "route or principal not found"}
        return 200, canonical

    monkeypatch.setattr(producer_module, "_request_json", fake_request)
    openapi = {"paths": {"/api/v1/executions/{execution_id}/cancel": {"post": {}}}}

    producer_module._assert_terminal_mutation_denied("cancel", openapi, "exec-1", canonical)

    assert calls == [
        ("POST", "/api/v1/executions/exec-1/cancel"),
        ("GET", "/api/v1/executions/exec-1"),
    ]


def test_terminal_mutation_probe_detects_accidental_state_change(
    producer_module: ModuleType, monkeypatch: pytest.MonkeyPatch
) -> None:
    canonical = {"execution_id": "exec-1", "run_id": "run-1", "status": "completed"}
    mutated = {**canonical, "status": "cancelled"}

    def fake_request(method: str, _path: str, **_kwargs: Any) -> tuple[int, Any]:
        return (409, {"detail": "terminal"}) if method == "POST" else (200, mutated)

    monkeypatch.setattr(producer_module, "_request_json", fake_request)
    openapi = {"paths": {"/api/v1/executions/{execution_id}/cancel": {"post": {}}}}

    with pytest.raises(AssertionError, match="mutated persisted terminal execution"):
        producer_module._assert_terminal_mutation_denied("cancel", openapi, "exec-1", canonical)


def test_terminal_mutation_transport_crash_fails_closed_without_hidden_retry(
    producer_module: ModuleType, monkeypatch: pytest.MonkeyPatch
) -> None:
    canonical = {"execution_id": "exec-1", "run_id": "run-1", "status": "completed"}
    calls = 0

    def crashing_request(_method: str, _path: str, **_kwargs: Any) -> tuple[int, Any]:
        nonlocal calls
        calls += 1
        raise requests.ConnectionError("controlled transport crash")

    monkeypatch.setattr(producer_module, "_request_json", crashing_request)
    openapi = {"paths": {"/api/v1/executions/{execution_id}/timeout": {"post": {}}}}

    with pytest.raises(requests.ConnectionError, match="controlled transport crash"):
        producer_module._assert_terminal_mutation_denied("timeout", openapi, "exec-1", canonical)

    assert calls == 1


def test_request_correlation_can_be_satisfied_by_echoed_input_alone() -> None:
    source = _text(PRODUCER_TEST)
    assert '"request_id": request_id' in source
    assert "request_id in json.dumps(terminal, sort_keys=True)" in source
    assert "terminal.get(" not in source
