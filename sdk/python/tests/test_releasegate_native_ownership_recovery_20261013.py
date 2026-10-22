"""Native ReleaseGate ownership and restart evidence gate.

This test intentionally uses an external native service process and real HTTP.
It never substitutes an in-process fake. Configure the executable with
AGENTFIELD_NATIVE_E2E_START_ARGV (a JSON string array). The executable must
honour AGENTFIELD_NATIVE_E2E_BASE_URL and AGENTFIELD_NATIVE_E2E_STATE_DIR.
"""

from __future__ import annotations

import json
import os
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import pytest


TERMINAL_STATUSES = {"succeeded", "failed", "timed_out", "cancelled"}


@dataclass(frozen=True)
class NativeConfig:
    start_argv: tuple[str, ...]
    base_url: str
    state_dir: Path
    startup_timeout_seconds: float
    result_timeout_seconds: float


def _required_environment(name: str) -> str:
    value = os.environ.get(name, "").strip()
    assert value, (
        f"{name} is required for native E2E evidence; "
        "an absent native runtime is BLOCKED, not skipped or mocked"
    )
    return value


def _load_config(tmp_path: Path) -> NativeConfig:
    raw_argv = _required_environment("AGENTFIELD_NATIVE_E2E_START_ARGV")
    try:
        decoded_argv = json.loads(raw_argv)
    except json.JSONDecodeError as error:
        raise AssertionError("AGENTFIELD_NATIVE_E2E_START_ARGV must be JSON") from error
    assert isinstance(decoded_argv, list) and decoded_argv, (
        "AGENTFIELD_NATIVE_E2E_START_ARGV must be a non-empty JSON string array"
    )
    assert all(isinstance(item, str) and item for item in decoded_argv), (
        "AGENTFIELD_NATIVE_E2E_START_ARGV entries must be non-empty strings"
    )

    base_url = os.environ.get(
        "AGENTFIELD_NATIVE_E2E_BASE_URL", "http://127.0.0.1:18765"
    ).rstrip("/")
    parsed_base_url = urllib.parse.urlparse(base_url)
    assert parsed_base_url.scheme == "http", "native E2E service must use local HTTP"
    assert parsed_base_url.hostname in {"127.0.0.1", "localhost", "::1"}, (
        "native E2E service must bind to loopback; outbound calls are forbidden"
    )
    assert parsed_base_url.port is not None, "native E2E base URL requires a port"
    state_dir = Path(
        os.environ.get("AGENTFIELD_NATIVE_E2E_STATE_DIR", str(tmp_path / "native-state"))
    ).resolve()
    state_dir.mkdir(parents=True, exist_ok=True)
    return NativeConfig(
        start_argv=tuple(decoded_argv),
        base_url=base_url,
        state_dir=state_dir,
        startup_timeout_seconds=float(
            os.environ.get("AGENTFIELD_NATIVE_E2E_STARTUP_TIMEOUT_SECONDS", "20")
        ),
        result_timeout_seconds=float(
            os.environ.get("AGENTFIELD_NATIVE_E2E_RESULT_TIMEOUT_SECONDS", "30")
        ),
    )


def _request_json(
    method: str,
    url: str,
    payload: dict[str, Any] | None = None,
    timeout_seconds: float = 5,
) -> tuple[int, dict[str, Any]]:
    body = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
            response_body = response.read().decode("utf-8")
            return response.status, json.loads(response_body) if response_body else {}
    except urllib.error.HTTPError as error:
        response_body = error.read().decode("utf-8")
        parsed = json.loads(response_body) if response_body else {}
        return error.code, parsed


def _wait_until_ready(config: NativeConfig, process: subprocess.Popen[bytes]) -> None:
    deadline = time.monotonic() + config.startup_timeout_seconds
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        exit_code = process.poll()
        assert exit_code is None, f"native service exited during startup with {exit_code}"
        try:
            status, payload = _request_json("GET", f"{config.base_url}/health")
            if status == 200 and payload.get("status") == "ok":
                return
        except (OSError, ValueError) as error:
            last_error = error
        time.sleep(0.1)
    raise AssertionError(f"native service did not become ready: {last_error!r}")


def _start_service(config: NativeConfig) -> subprocess.Popen[bytes]:
    environment = os.environ.copy()
    environment["AGENTFIELD_NATIVE_E2E_BASE_URL"] = config.base_url
    environment["AGENTFIELD_NATIVE_E2E_STATE_DIR"] = str(config.state_dir)
    process = subprocess.Popen(
        config.start_argv,
        env=environment,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        shell=False,
    )
    try:
        _wait_until_ready(config, process)
    except BaseException:
        _stop_service(process)
        raise
    return process


def _stop_service(process: subprocess.Popen[bytes]) -> None:
    if process.poll() is not None:
        return
    process.terminate()
    try:
        process.wait(timeout=10)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=10)


def _submit(config: NativeConfig, request_id: str, behavior: str) -> dict[str, Any]:
    status, payload = _request_json(
        "POST",
        f"{config.base_url}/v1/releasegate/tasks",
        {
            "request_id": request_id,
            "kind": "native_releasegate_probe",
            "behavior": behavior,
            "payload": {"marker": request_id},
        },
    )
    assert status in {200, 201, 202}, (status, payload)
    assert payload.get("request_id") == request_id, payload
    assert isinstance(payload.get("task_id"), str) and payload["task_id"], payload
    return payload


def _read_task(config: NativeConfig, task_id: str) -> dict[str, Any]:
    status, payload = _request_json(
        "GET", f"{config.base_url}/v1/releasegate/tasks/{task_id}"
    )
    assert status == 200, (status, payload)
    assert payload.get("task_id") == task_id, payload
    return payload


def _wait_terminal(config: NativeConfig, task_id: str) -> dict[str, Any]:
    deadline = time.monotonic() + config.result_timeout_seconds
    last_payload: dict[str, Any] | None = None
    while time.monotonic() < deadline:
        last_payload = _read_task(config, task_id)
        if last_payload.get("status") in TERMINAL_STATUSES:
            return last_payload
        time.sleep(0.1)
    raise AssertionError(f"task did not reach terminal state: {last_payload!r}")


def _assert_single_owner(first: dict[str, Any], duplicate: dict[str, Any]) -> None:
    assert duplicate["task_id"] == first["task_id"], (first, duplicate)
    first_result_id = first.get("result_id")
    duplicate_result_id = duplicate.get("result_id")
    if first_result_id is not None or duplicate_result_id is not None:
        assert first_result_id == duplicate_result_id, (first, duplicate)


@pytest.fixture
def native_config(tmp_path: Path) -> NativeConfig:
    return _load_config(tmp_path)


@pytest.fixture
def native_service(native_config: NativeConfig):
    process = _start_service(native_config)
    try:
        yield native_config, process
    finally:
        _stop_service(process)


def test_native_success_preserves_one_idempotency_owner(native_service) -> None:
    config, _process = native_service
    request_id = f"success-{uuid.uuid4()}"

    first = _submit(config, request_id, "success")
    duplicate = _submit(config, request_id, "success")
    _assert_single_owner(first, duplicate)

    terminal = _wait_terminal(config, first["task_id"])
    assert terminal.get("status") == "succeeded", terminal
    assert terminal.get("result", {}).get("marker") == request_id, terminal

    duplicate_after_completion = _submit(config, request_id, "success")
    _assert_single_owner(terminal, duplicate_after_completion)


def test_native_failure_cannot_be_reduced_to_success(native_service) -> None:
    config, _process = native_service
    request_id = f"failure-{uuid.uuid4()}"

    first = _submit(config, request_id, "failure")
    terminal = _wait_terminal(config, first["task_id"])
    assert terminal.get("status") == "failed", terminal
    assert terminal.get("error"), terminal
    assert terminal.get("result") in (None, {}), terminal

    duplicate = _submit(config, request_id, "failure")
    _assert_single_owner(terminal, duplicate)
    duplicate_terminal = _wait_terminal(config, duplicate["task_id"])
    assert duplicate_terminal.get("status") == "failed", duplicate_terminal


def test_native_restart_recovers_persisted_owner_and_readback(native_config) -> None:
    first_process = _start_service(native_config)
    request_id = f"restart-{uuid.uuid4()}"
    try:
        first = _submit(native_config, request_id, "wait_for_restart")
        before_restart = _read_task(native_config, first["task_id"])
        assert before_restart.get("status") not in TERMINAL_STATUSES, before_restart
    finally:
        _stop_service(first_process)

    second_process = _start_service(native_config)
    try:
        duplicate = _submit(native_config, request_id, "wait_for_restart")
        _assert_single_owner(first, duplicate)
        terminal = _wait_terminal(native_config, first["task_id"])
        assert terminal.get("status") == "succeeded", terminal
        assert terminal.get("recovered_after_restart") is True, terminal
        assert terminal.get("result", {}).get("marker") == request_id, terminal

        persisted_readback = _read_task(native_config, first["task_id"])
        assert persisted_readback == terminal
        duplicate_after_recovery = _submit(
            native_config, request_id, "wait_for_restart"
        )
        _assert_single_owner(terminal, duplicate_after_recovery)
    finally:
        _stop_service(second_process)
