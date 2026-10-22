from __future__ import annotations

import asyncio
import hashlib
import inspect
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


class DisposableProvider:
    def __init__(self, responses: list[Any]) -> None:
        self._responses = list(responses)
        self.calls: list[tuple[str, dict[str, Any]]] = []

    async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
        self.calls.append((prompt, dict(options)))
        if not self._responses:
            raise AssertionError("disposable provider was retried beyond its fixture")
        return self._responses.pop(0)


def _options(tmp_path: Path, **extra: Any) -> dict[str, Any]:
    options = {
        "provider": "codex",
        "cwd": str(tmp_path),
        "project_dir": str(tmp_path),
        "schema_max_retries": 2,
        "_original_prompt": "northstar longbundle live result",
    }
    options.update(extra)
    return options


def _run_schema_retry(
    initial_raw: Any,
    provider: DisposableProvider,
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


def _disable_file_and_stdout_parsers(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "try_parse_from_text", lambda text, schema: None)


def test_live_result_contract_has_immutable_runner_and_dto_signatures() -> None:
    runner_path = Path(runner_module.__file__)
    assert hashlib.sha256(runner_path.read_bytes()).hexdigest() == RUNNER_SOURCE_SHA256

    repair_parameters = list(inspect.signature(runner_module._ai_schema_repair).parameters)
    assert repair_parameters == ["text", "schema", "options"]

    retry_parameters = list(inspect.signature(HarnessRunner._handle_schema_with_retry).parameters)
    assert retry_parameters == [
        "self",
        "initial_raw",
        "schema",
        "cwd",
        "start_time",
        "provider",
        "options",
    ]
    assert C4_RESULT_DTO_SIGNATURE == "agentfield.harness.result-metrics.v1"


def test_async_repair_returns_valid_dto_without_provider_replay(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    _disable_file_and_stdout_parsers(monkeypatch)
    repair_calls: list[tuple[str, Any, dict[str, Any]]] = []

    async def repair(text: str, schema: Any, options: dict[str, Any]) -> Any:
        repair_calls.append((text, schema, dict(options)))
        return {"status": "repaired"}

    monkeypatch.setattr(runner_module, "_ai_schema_repair", repair)
    provider = DisposableProvider([])

    result = _run_schema_retry(
        _raw(result="{malformed-but-repairable", metrics=_metrics(session_id="initial")),
        provider,
        tmp_path,
    )

    assert result.is_error is False
    assert result.parsed == {"status": "repaired"}
    assert len(repair_calls) == 1
    assert repair_calls[0][0] == "{malformed-but-repairable"
    assert provider.calls == []
    assert result.session_id == "initial"


def test_malformed_result_fails_closed_with_bounded_resume_retries(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    _disable_file_and_stdout_parsers(monkeypatch)
    repair_calls = 0

    async def repair(text: str, schema: Any, options: dict[str, Any]) -> Any:
        nonlocal repair_calls
        repair_calls += 1
        return None

    monkeypatch.setattr(runner_module, "_ai_schema_repair", repair)
    monkeypatch.setattr(
        runner_module, "diagnose_output_failure", lambda path, schema: "malformed DTO"
    )
    monkeypatch.setattr(
        runner_module, "build_followup_prompt", lambda detail, cwd, schema: "repair"
    )
    provider = DisposableProvider(
        [
            _raw(result="{still malformed", metrics=_metrics(session_id="retry-1")),
            _raw(result="{still malformed", metrics=_metrics(session_id="retry-2")),
        ]
    )

    result = _run_schema_retry(
        _raw(result="{malformed", metrics=_metrics(session_id="initial")),
        provider,
        tmp_path,
    )

    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == FailureType.SCHEMA
    assert "failed after 2 retry attempt(s)" in result.error_message
    assert repair_calls == 1
    assert len(provider.calls) == 2
    assert provider.calls[1][1]["resume_session_id"] == "retry-1"
    assert result.session_id == "retry-2"


def test_crash_restart_with_invalid_replacement_remains_failed_and_non_idempotent(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    _disable_file_and_stdout_parsers(monkeypatch)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", lambda *args: None)
    monkeypatch.setattr(
        runner_module, "diagnose_output_failure", lambda path, schema: "restart invalid"
    )
    provider = DisposableProvider(
        [_raw(result="{replacement still invalid", metrics=_metrics(session_id="replacement"))]
    )

    result = _run_schema_retry(
        _raw(
            is_error=True,
            error_message="provider crashed",
            failure_type=FailureType.CRASH,
            metrics=_metrics(session_id="stale-session"),
        ),
        provider,
        tmp_path,
        schema_max_retries=1,
    )

    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == FailureType.SCHEMA
    assert "failed after 1 retry attempt(s)" in result.error_message
    assert len(provider.calls) == 1
    assert provider.calls[0][0] == "northstar longbundle live result"
    assert "resume_session_id" not in provider.calls[0][1]
    assert result.session_id == "replacement"
