from __future__ import annotations

import asyncio
import hashlib
from pathlib import Path
from types import SimpleNamespace
from typing import Any

import pytest

from agentfield.harness import _runner as runner_module
from agentfield.harness._result import FailureType
from agentfield.harness._runner import HarnessRunner, _resolve_options


RUNNER_SOURCE_SHA256 = (
    "06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8"
)
RESULT_DTO_VERSION = "agentfield.harness.result-metrics.v1"
RESULT_DTO_FIELDS = (
    "input_tokens",
    "output_tokens",
    "cache_read_tokens",
    "cache_creation_tokens",
    "total_tokens",
    "model",
)


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


def _schema_options(tmp_path: Any, **extra: Any) -> dict[str, Any]:
    options = {
        "provider": "codex",
        "cwd": str(tmp_path),
        "project_dir": str(tmp_path),
        "schema_max_retries": 1,
        "_original_prompt": "request replay boundary",
    }
    options.update(extra)
    return options


def test_request_to_result_positive_binds_source_and_dto_boundary(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    observed: dict[str, Any] = {}

    class DeterministicProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            observed["prompt"] = prompt
            observed["options"] = dict(options)
            return _raw(
                result='{"status":"ok","request_id":"req-1"}',
                metrics=_metrics(
                    total_cost_usd=0.25,
                    num_turns=1,
                    session_id="positive-session",
                    input_tokens=8,
                    output_tokens=4,
                    cache_read_tokens=2,
                    cache_creation_tokens=1,
                    model="deterministic-model",
                ),
            )

    runner = HarnessRunner()
    monkeypatch.setattr(
        runner, "_build_provider", lambda provider_name, options: DeterministicProvider()
    )

    result = asyncio.run(
        runner.run(
            "return one accepted result",
            provider="codex",
            project_dir=str(tmp_path),
            max_retries=0,
        )
    )

    assert hashlib.sha256(Path(runner_module.__file__).read_bytes()).hexdigest() == RUNNER_SOURCE_SHA256
    assert RESULT_DTO_VERSION.endswith(".v1")
    assert RESULT_DTO_FIELDS == (
        "input_tokens",
        "output_tokens",
        "cache_read_tokens",
        "cache_creation_tokens",
        "total_tokens",
        "model",
    )
    assert result.is_error is False
    assert result.result == '{"status":"ok","request_id":"req-1"}'
    assert result.session_id == "positive-session"
    assert result.input_tokens == 8
    assert result.output_tokens == 4
    assert result.cache_read_tokens == 2
    assert result.cache_creation_tokens == 1
    assert result.total_tokens == 12
    assert observed["prompt"] == "return one accepted result"
    assert observed["options"]["provider"] == "codex"


def test_option_resolution_rejects_config_leak_but_preserves_explicit_replay_override() -> None:
    config = SimpleNamespace(
        provider="codex",
        model="configured-model",
        max_retries=4,
        schema_mode="single",
        tools=None,
        private_config="must-not-cross-boundary",
    )

    options = _resolve_options(
        config,
        {
            "model": "replay-model",
            "max_retries": None,
            "schema_mode": "incremental",
            "resume_session_id": "session-1",
        },
    )

    assert options["provider"] == "codex"
    assert options["model"] == "replay-model"
    assert options["max_retries"] == 4
    assert options["schema_mode"] == "incremental"
    assert options["resume_session_id"] == "session-1"
    assert "tools" not in options
    assert "private_config" not in options


def test_malformed_result_exhausts_schema_retries_and_fails_closed(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    provider_calls: list[dict[str, Any]] = []

    class MalformedProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            provider_calls.append(dict(options))
            return _raw(
                result="{malformed response",
                metrics=_metrics(session_id="malformed-session", input_tokens=2, output_tokens=1),
            )

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", lambda text, schema, options: None)
    monkeypatch.setattr(
        runner_module,
        "diagnose_output_failure",
        lambda path, schema: "malformed result DTO",
    )
    monkeypatch.setattr(
        runner_module,
        "build_followup_prompt",
        lambda detail, cwd, schema: "repair result DTO",
    )

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(result="{malformed response", metrics=_metrics(input_tokens=1)),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=MalformedProvider(),
            options=_schema_options(tmp_path, schema_max_retries=2),
        )
    )

    assert len(provider_calls) == 2
    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == FailureType.SCHEMA
    assert "failed after 2 retry attempt(s)" in result.error_message
    assert result.total_tokens == 4
    assert all("request replay boundary" in call.get("_original_prompt", "") for call in provider_calls)


def test_terminal_timeout_is_not_replayed_or_retried(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    timeout_type = FailureType.__members__["TIMEOUT"]
    provider_calls = 0

    class ForbiddenReplayProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            nonlocal provider_calls
            provider_calls += 1
            raise AssertionError("terminal timeout must not trigger a provider replay")

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(runner_module, "_ai_schema_repair", lambda text, schema, options: None)

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(
                is_error=True,
                error_message="provider timed out",
                failure_type=timeout_type,
                metrics=_metrics(session_id="timeout-session", input_tokens=5, output_tokens=1),
            ),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=ForbiddenReplayProvider(),
            options=_schema_options(tmp_path, schema_max_retries=3),
        )
    )

    assert provider_calls == 0
    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == timeout_type
    assert "provider timed out" in result.error_message
    assert "Output file was not created" in result.error_message
    assert result.total_tokens == 6


def test_restart_discards_stale_session_and_reuses_original_request(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    parse_results = iter([None, {"status": "recovered"}])
    observed: dict[str, Any] = {}

    class RestartProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            observed["prompt"] = prompt
            observed["options"] = dict(options)
            return _raw(
                result='{"status":"recovered"}',
                metrics=_metrics(session_id="replacement-session", input_tokens=4, output_tokens=2),
            )

    monkeypatch.setattr(
        runner_module, "parse_and_validate", lambda path, schema: next(parse_results)
    )
    monkeypatch.setattr(
        runner_module,
        "diagnose_output_failure",
        lambda path, schema: "missing output after restart",
    )

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(
                is_error=True,
                error_message="provider crashed",
                failure_type=FailureType.CRASH,
                metrics=_metrics(session_id="stale-session", input_tokens=3, output_tokens=1),
            ),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=RestartProvider(),
            options=_schema_options(tmp_path),
        )
    )

    assert result.is_error is False
    assert result.parsed == {"status": "recovered"}
    assert result.session_id == "replacement-session"
    assert observed["prompt"] == "request replay boundary"
    assert "resume_session_id" not in observed["options"]
    assert result.total_tokens == 10


def test_non_crash_replay_resumes_latest_session_once(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    parse_results = iter([None, {"status": "resumed"}])
    observed: list[dict[str, Any]] = []

    class ResumeProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            observed.append(dict(options))
            return _raw(
                result='{"status":"resumed"}',
                metrics=_metrics(session_id="same-session", output_tokens=2),
            )

    monkeypatch.setattr(
        runner_module, "parse_and_validate", lambda path, schema: next(parse_results)
    )
    monkeypatch.setattr(runner_module, "_ai_schema_repair", lambda text, schema, options: None)
    monkeypatch.setattr(runner_module, "diagnose_output_failure", lambda path, schema: "invalid output")
    monkeypatch.setattr(
        runner_module,
        "build_followup_prompt",
        lambda detail, cwd, schema: "repair output",
    )

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(
                is_error=True,
                error_message="no output",
                failure_type=FailureType.NO_OUTPUT,
                metrics=_metrics(session_id="same-session", input_tokens=2),
            ),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=ResumeProvider(),
            options=_schema_options(tmp_path),
        )
    )

    assert result.is_error is False
    assert result.parsed == {"status": "resumed"}
    assert len(observed) == 1
    assert observed[0]["resume_session_id"] == "same-session"
    assert result.total_tokens == 4
