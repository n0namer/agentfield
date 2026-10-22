from __future__ import annotations

import asyncio
import hashlib
from pathlib import Path
from types import SimpleNamespace
from typing import Any

import pytest

from agentfield.harness import _runner as runner_module
from agentfield.harness._result import FailureType
from agentfield.harness._runner import (
    HarnessRunner,
    _accumulate_metrics,
    _resolve_options,
    _token_result_kwargs,
)


PYTHON_RUNNER_SOURCE_SHA256 = (
    "06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8"
)
DTO_CONTRACT_VERSION = "agentfield.harness.result-metrics.v1"
DTO_WIRE_FIELDS = (
    "input_tokens",
    "output_tokens",
    "cache_read_tokens",
    "cache_creation_tokens",
    "total_tokens",
    "model",
)


def _metrics(
    *,
    cost: float | None = None,
    turns: int = 0,
    session_id: str = "",
    input_tokens: int | None = 0,
    output_tokens: int | None = 0,
    cache_read_tokens: int | None = 0,
    cache_creation_tokens: int | None = 0,
    model: str | None = None,
) -> SimpleNamespace:
    return SimpleNamespace(
        total_cost_usd=cost,
        num_turns=turns,
        session_id=session_id,
        input_tokens=input_tokens,
        output_tokens=output_tokens,
        cache_read_tokens=cache_read_tokens,
        cache_creation_tokens=cache_creation_tokens,
        model=model,
    )


def _raw(
    *,
    result: str = "",
    is_error: bool = False,
    error_message: str | None = None,
    failure_type: FailureType = FailureType.NONE,
    metrics: SimpleNamespace | None = None,
    messages: list[dict[str, Any]] | None = None,
) -> SimpleNamespace:
    return SimpleNamespace(
        result=result,
        is_error=is_error,
        error_message=error_message,
        failure_type=failure_type,
        metrics=metrics or _metrics(),
        messages=messages or [],
    )


def test_resolve_options_and_versioned_token_dto_invariants() -> None:
    runner_source = Path(runner_module.__file__).read_bytes()
    assert hashlib.sha256(runner_source).hexdigest() == PYTHON_RUNNER_SOURCE_SHA256

    config = SimpleNamespace(
        provider="codex",
        model="configured-model",
        max_retries=4,
        schema_mode="single",
        tools=None,
        unrelated="must-not-cross-boundary",
    )

    options = _resolve_options(
        config,
        {
            "model": "override-model",
            "max_retries": None,
            "schema_mode": "incremental",
            "foreign_option": "caller-owned",
        },
    )

    assert options["provider"] == "codex"
    assert options["model"] == "override-model"
    assert options["max_retries"] == 4
    assert options["schema_mode"] == "incremental"
    assert options["foreign_option"] == "caller-owned"
    assert "tools" not in options
    assert "unrelated" not in options

    aggregated = _accumulate_metrics(
        [
            _raw(
                metrics=_metrics(
                    cost=0.25,
                    turns=1,
                    session_id="stale-session",
                    input_tokens=10,
                    output_tokens=None,
                    cache_read_tokens=3,
                    model="model-a",
                ),
                messages=[{"role": "assistant", "content": "first"}],
            ),
            _raw(
                metrics=_metrics(
                    cost=0.75,
                    turns=2,
                    session_id="fresh-session",
                    input_tokens=5,
                    output_tokens=7,
                    cache_creation_tokens=2,
                    model="model-b",
                ),
                messages=[{"role": "assistant", "content": "second"}],
            ),
        ]
    )
    cost, turns, session_id, messages, tokens = aggregated
    dto = _token_result_kwargs(tokens)

    assert DTO_CONTRACT_VERSION.endswith(".v1")
    assert tuple(dto) == DTO_WIRE_FIELDS
    assert cost == pytest.approx(1.0)
    assert turns == 3
    assert session_id == "fresh-session"
    assert len(messages) == 2
    assert dto == {
        "input_tokens": 15,
        "output_tokens": 7,
        "cache_read_tokens": 3,
        "cache_creation_tokens": 2,
        "total_tokens": 22,
        "model": "model-a",
    }


def test_native_runner_positive_path_preserves_result_and_metrics(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    observed: dict[str, Any] = {}

    class DeterministicProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            observed["prompt"] = prompt
            observed["options"] = dict(options)
            return _raw(
                result='{"status":"ok"}',
                metrics=_metrics(
                    cost=0.5,
                    turns=2,
                    session_id="native-positive",
                    input_tokens=11,
                    output_tokens=4,
                    cache_read_tokens=6,
                    model="contract-model",
                ),
                messages=[{"role": "assistant", "content": "ok"}],
            )

    native_runner = HarnessRunner()
    monkeypatch.setattr(
        native_runner,
        "_build_provider",
        lambda provider_name, options: DeterministicProvider(),
    )

    result = asyncio.run(
        native_runner.run(
            "return a deterministic result",
            provider="codex",
            project_dir=str(tmp_path),
            max_retries=0,
        )
    )

    assert result.is_error is False
    assert result.result == '{"status":"ok"}'
    assert result.parsed is None
    assert result.session_id == "native-positive"
    assert result.input_tokens == 11
    assert result.output_tokens == 4
    assert result.cache_read_tokens == 6
    assert result.total_tokens == 15
    assert observed["prompt"] == "return a deterministic result"
    assert observed["options"]["provider"] == "codex"


def test_structured_timeout_fails_closed_without_retry(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    assert "TIMEOUT" in FailureType.__members__
    timeout_type = FailureType.__members__["TIMEOUT"]
    provider_calls = 0

    class ForbiddenProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            nonlocal provider_calls
            provider_calls += 1
            raise AssertionError("terminal timeout must not be retried as transient")

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)
    monkeypatch.setattr(
        runner_module, "_ai_schema_repair", lambda text, schema, options: None
    )

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(
                is_error=True,
                error_message="provider timed out",
                failure_type=timeout_type,
                metrics=_metrics(
                    session_id="timeout-session", input_tokens=9, output_tokens=1
                ),
            ),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=ForbiddenProvider(),
            options={
                "provider": "codex",
                "cwd": str(tmp_path),
                "project_dir": str(tmp_path),
                "schema_max_retries": 2,
            },
        )
    )

    assert provider_calls == 0
    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == timeout_type
    assert "provider timed out" in result.error_message
    assert "Output file was not created" in result.error_message
    assert result.total_tokens == 10


def test_invalid_structured_response_exhausts_retries_and_fails_closed(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    provider_calls: list[dict[str, Any]] = []

    class InvalidProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            provider_calls.append(dict(options))
            return _raw(
                result="still not valid structured output",
                metrics=_metrics(cost=0.2, turns=1, input_tokens=2, output_tokens=3),
            )

    monkeypatch.setattr(runner_module, "parse_and_validate", lambda path, schema: None)

    async def no_repair(text: str, schema: Any, options: dict[str, Any]) -> None:
        return None

    monkeypatch.setattr(runner_module, "_ai_schema_repair", no_repair)
    monkeypatch.setattr(
        runner_module,
        "diagnose_output_failure",
        lambda path, schema: "invalid response contract",
    )
    monkeypatch.setattr(
        runner_module,
        "build_followup_prompt",
        lambda detail, cwd, schema: "repair structured response",
    )

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(
                result="not valid structured output",
                metrics=_metrics(cost=0.1, turns=1, input_tokens=1, output_tokens=1),
            ),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=InvalidProvider(),
            options={
                "provider": "codex",
                "cwd": str(tmp_path),
                "project_dir": str(tmp_path),
                "schema_max_retries": 2,
                "_original_prompt": "original task",
            },
        )
    )

    assert len(provider_calls) == 2
    assert result.is_error is True
    assert result.parsed is None
    assert result.failure_type == FailureType.SCHEMA
    assert "failed after 2 retry attempt(s)" in result.error_message
    assert result.cost_usd == pytest.approx(0.5)
    assert result.num_turns == 3
    assert result.total_tokens == 12


def test_crash_restart_uses_original_prompt_without_stale_session(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    parsed_value = {"status": "recovered"}
    parse_results = iter([None, parsed_value])
    observed: dict[str, Any] = {}

    class RestartProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            observed["prompt"] = prompt
            observed["options"] = dict(options)
            return _raw(
                result='{"status":"recovered"}',
                metrics=_metrics(
                    session_id="replacement-session", input_tokens=4, output_tokens=2
                ),
            )

    monkeypatch.setattr(
        runner_module, "parse_and_validate", lambda path, schema: next(parse_results)
    )
    monkeypatch.setattr(
        runner_module,
        "diagnose_output_failure",
        lambda path, schema: "missing output after crash",
    )

    result = asyncio.run(
        HarnessRunner()._handle_schema_with_retry(
            _raw(
                is_error=True,
                error_message="provider crashed",
                failure_type=FailureType.CRASH,
                metrics=_metrics(
                    session_id="stale-session", input_tokens=3, output_tokens=1
                ),
            ),
            schema=object(),
            cwd=str(tmp_path),
            start_time=runner_module.time.monotonic(),
            provider=RestartProvider(),
            options={
                "provider": "codex",
                "cwd": str(tmp_path),
                "project_dir": str(tmp_path),
                "schema_max_retries": 1,
                "_original_prompt": "original structured task",
            },
        )
    )

    assert result.is_error is False
    assert result.parsed == parsed_value
    assert result.session_id == "replacement-session"
    assert result.total_tokens == 10
    assert observed["prompt"] == "original structured task"
    assert "resume_session_id" not in observed["options"]


def test_non_crash_retry_resumes_last_native_session(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Any
) -> None:
    parsed_value = {"status": "resumed"}
    parse_results = iter([None, parsed_value])
    observed: dict[str, Any] = {}

    class ResumeProvider:
        async def execute(self, prompt: str, options: dict[str, Any]) -> Any:
            observed["options"] = dict(options)
            return _raw(
                result='{"status":"resumed"}',
                metrics=_metrics(session_id="same-session", output_tokens=2),
            )

    monkeypatch.setattr(
        runner_module, "parse_and_validate", lambda path, schema: next(parse_results)
    )
    monkeypatch.setattr(
        runner_module,
        "diagnose_output_failure",
        lambda path, schema: "output missing",
    )
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
            options={
                "provider": "codex",
                "cwd": str(tmp_path),
                "project_dir": str(tmp_path),
                "schema_max_retries": 1,
                "_original_prompt": "original task",
            },
        )
    )

    assert result.is_error is False
    assert result.parsed == parsed_value
    assert observed["options"]["resume_session_id"] == "same-session"
