from __future__ import annotations

import asyncio
from pathlib import Path
from types import SimpleNamespace
from typing import Any

import pytest

from agentfield.harness import _runner as runner_module
from agentfield.harness._result import FailureType
from agentfield.harness._runner import HarnessRunner


RUNNER_SOURCE_SHA256 = (
    "06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8"
)
C4_RESULT_DTO_SIGNATURE = "agentfield.harness.result-metrics.v1"


def _metrics(**overrides: Any) -> SimpleNamespace:
    values = {
        "total_cost_usd": None,
        "num_turns": 0,
        "session_id": "",
        "input_tokens": 0,
        "output_tokens": 0,
        "cache_read_tokens": 0,
        "cache_creation_tokens": 0,
        "model": None,
    }
    values.update(overrides)
    return SimpleNamespace(**values)


def _raw(
    *,
    result: str = "",
    is_error: bool = False,
    error_message: str | None = None,
    failure_type: FailureType = FailureType.NONE,
    metrics: SimpleNamespace | None = None,
) -> SimpleNamespace:
    return SimpleNamespace(
        result=result,
        is_error=is_error,
        error_message=error_message,
        failure_type=failure_type,
        metrics=metrics or _metrics(),
        messages=[],
    )


def _options(tmp_path: Path, **extra: Any) -> dict[str, Any]:
    options = {
        "provider": "codex",
        "cwd": str(tmp_path),
        "project_dir": str(tmp_path),
        "schema_max_retries": 2,
        "_original_prompt": "native async schema retry contract",
    }
    options.update(extra)
    return options


def _run_schema_retry(
    initial_raw: Any,
    provider: Any,
    tmp_path: Path,
    **options: Any,
) -> Any:
    return asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            initial_raw,
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=provider,
            options=_options(tmp_path, **options),
        )
    )


def test_async_schema_repair_fixture_returns_valid_result_without_harness_retry(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    repair_calls: list[tuple[str, Any, dict[str, Any]]] = []
    provider_calls = 0

    async def async_repair(text: str, schema: Any, options: dict[str, Any]) -> Any:
        repair_calls.append((text, schema, dict(options)))
        return {"status": "repaired"}

    class ForbiddenReplayProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            nonlocal provider_calls
            provider_calls += 1
            raise AssertionError("successful async repair must not replay the harness")

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "try_parse_from_text", lambda text, schema: None)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", async_repair)

    result = _run_schema_retry(
        _raw(result="{malformed-but-repairable", metrics=_metrics(session_id="initial")),
        ForbiddenReplayProvider(),
        tmp_path,
    )

    assert result.is_error is False
    assert result.parsed == {"status": "repaired"}
    assert len(repair_calls) == 1
    assert repair_calls[0][0] == "{malformed-but-repairable"
    assert provider_calls == 0
    assert hashlib_sha256(runner_module.__file__) == RUNNER_SOURCE_SHA256
    assert C4_RESULT_DTO_SIGNATURE.endswith(".v1")


def test_malformed_async_repair_fails_closed_after_bounded_retries(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    repair_calls = 0
    provider_calls = 0

    async def async_repair(text: str, schema: Any, options: dict[str, Any]) -> Any:
        nonlocal repair_calls
        repair_calls += 1
        return None

    class MalformedProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            nonlocal provider_calls
            provider_calls += 1
            return _raw(result="{still malformed", metrics=_metrics(session_id=f"retry-{provider_calls}"))

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "try_parse_from_text", lambda text, schema: None)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", async_repair)
    monkeypatch.setattr(runner_module, "diagnose_output_failure", lambda path, schema: "malformed DTO")
    monkeypatch.setattr(runner_module, "build_followup_prompt", lambda detail, cwd, schema: "repair")

    result = _run_schema_retry(
        _raw(result="{malformed", metrics=_metrics(input_tokens=1)),
        MalformedProvider(),
        tmp_path,
    )

    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == FailureType.SCHEMA
    assert "failed after 2 retry attempt(s)" in result.error_message
    assert provider_calls == 2
    assert repair_calls == 1


def test_terminal_timeout_cannot_become_success_via_async_repair(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    repair_calls = 0

    async def async_repair(text: str, schema: Any, options: dict[str, Any]) -> Any:
        nonlocal repair_calls
        repair_calls += 1
        return {"status": "must-not-accept"}

    class ForbiddenReplayProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            raise AssertionError("terminal timeout must not replay")

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", async_repair)

    timeout_type = FailureType.__members__["TIMEOUT"]
    result = _run_schema_retry(
        _raw(
            is_error=True,
            error_message="provider timed out",
            failure_type=timeout_type,
            metrics=_metrics(input_tokens=5, output_tokens=1),
        ),
        ForbiddenReplayProvider(),
        tmp_path,
        schema_max_retries=3,
    )

    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == timeout_type
    assert "provider timed out" in result.error_message
    assert repair_calls == 0


def test_crash_restart_with_invalid_replacement_remains_failed(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    provider_calls = 0

    async def async_repair(text: str, schema: Any, options: dict[str, Any]) -> Any:
        return None

    class RestartProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            nonlocal provider_calls
            provider_calls += 1
            return _raw(
                result="{replacement still invalid",
                metrics=_metrics(session_id="replacement-session"),
            )

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "try_parse_from_text", lambda text, schema: None)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", async_repair)
    monkeypatch.setattr(runner_module, "diagnose_output_failure", lambda path, schema: "restart invalid")
    monkeypatch.setattr(runner_module, "build_followup_prompt", lambda detail, cwd, schema: "repair")

    result = _run_schema_retry(
        _raw(
            is_error=True,
            error_message="provider crashed",
            failure_type=FailureType.CRASH,
            metrics=_metrics(session_id="stale-session"),
        ),
        RestartProvider(),
        tmp_path,
        schema_max_retries=1,
    )

    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == FailureType.SCHEMA
    assert "failed after 1 retry attempt(s)" in result.error_message
    assert provider_calls == 1


def hashlib_sha256(path: str) -> str:
    import hashlib

    return hashlib.sha256(Path(path).read_bytes()).hexdigest()
