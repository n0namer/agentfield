"""Independent offline QA for the Base20 harness lifecycle candidates.

The doubles exercise SDK boundaries only. They do not admit work to, or prove
execution by, an AgentField native control plane.
"""

from __future__ import annotations

import asyncio
import json
import re
from dataclasses import dataclass, field
from typing import Any
from unittest.mock import patch

import pytest
from pydantic import BaseModel

from agentfield.harness._result import Metrics, RawResult
from agentfield.harness._runner import HarnessRunner


class CorrelatedResult(BaseModel):
    request_id: str
    status: str
    answer: str
    receipt_id: str


@dataclass
class InvocationState:
    calls: int = 0
    effects: list[str] = field(default_factory=list)
    options: list[dict[str, Any]] = field(default_factory=list)


def _output_path(prompt: str) -> str:
    match = re.search(r"(\S*\.agentfield_output\.json)", prompt)
    assert match, "schema request must identify its output path"
    return match.group(1)


def _write_result(prompt: str, request_id: str) -> None:
    with open(_output_path(prompt), "w", encoding="utf-8") as output_file:
        json.dump(
            {
                "request_id": request_id,
                "status": "completed",
                "answer": "one durable effect",
                "receipt_id": f"receipt-{request_id}",
            },
            output_file,
        )


class Delayed502Provider:
    def __init__(self, state: InvocationState) -> None:
        self.state = state

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.state.calls += 1
        self.state.options.append(dict(options))
        if self.state.calls == 1:
            return RawResult(
                is_error=True,
                error_message="upstream returned HTTP 502 Bad Gateway",
                metrics=Metrics(session_id="provider-502-first"),
            )

        request_id = "qa-502-001"
        self.state.effects.append(request_id)
        _write_result(prompt, request_id)
        return RawResult(
            metrics=Metrics(session_id="provider-502-second"),
            messages=[
                {
                    "request_id": request_id,
                    "run_id": "run-qa-502-002",
                    "thread_id": "thread-qa-502-001",
                }
            ],
        )


class MalformedThenRecoveredProvider:
    def __init__(self, state: InvocationState) -> None:
        self.state = state

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.state.calls += 1
        self.state.options.append(dict(options))
        if self.state.calls == 1:
            with open(_output_path(prompt), "w", encoding="utf-8") as output_file:
                output_file.write('{"request_id":')
            return RawResult(metrics=Metrics(session_id="malformed-session"))

        assert options.get("resume_session_id") == "malformed-session"
        request_id = "qa-malformed-001"
        self.state.effects.append(request_id)
        _write_result(prompt, request_id)
        return RawResult(metrics=Metrics(session_id="malformed-recovered"))


class TimeoutProvider:
    def __init__(self) -> None:
        self.calls = 0
        self.effects: list[str] = []

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.calls += 1
        return RawResult(
            is_error=True,
            error_message="provider timeout",
            metrics=Metrics(session_id="timeout-qa-001"),
        )


class CancellationProvider:
    def __init__(self) -> None:
        self.calls = 0

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.calls += 1
        raise asyncio.CancelledError()


class NilResultProvider:
    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        return None  # type: ignore[return-value]


class ForeignTraceProvider:
    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        _write_result(prompt, "qa-request-001")
        return RawResult(
            metrics=Metrics(session_id="foreign-trace-session"),
            messages=[
                {
                    "request_id": "other-request-999",
                    "run_id": "run-foreign-001",
                    "thread_id": "thread-foreign-999",
                }
            ],
        )


@pytest.mark.asyncio
async def test_delayed_502_retries_once_and_returns_correlated_typed_result(
    tmp_path, monkeypatch: pytest.MonkeyPatch
):
    state = InvocationState()
    provider = Delayed502Provider(state)
    sleeps: list[float] = []

    async def fake_sleep(delay: float) -> None:
        sleeps.append(delay)

    monkeypatch.setattr("agentfield.harness._runner.asyncio.sleep", fake_sleep)
    monkeypatch.setattr("agentfield.harness._runner.random.uniform", lambda _a, _b: 0.0)

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await HarnessRunner().run(
            'Process {"request_id":"qa-502-001"}.',
            provider="aforge",
            schema=CorrelatedResult,
            cwd=str(tmp_path),
            max_retries=1,
            initial_delay=0.01,
        )

    assert result.is_error is False
    assert result.parsed == CorrelatedResult(
        request_id="qa-502-001",
        status="completed",
        answer="one durable effect",
        receipt_id="receipt-qa-502-001",
    )
    assert result.session_id == "provider-502-second"
    assert state.calls == 2
    assert state.effects == ["qa-502-001"]
    assert sleeps == [0.01]


@pytest.mark.asyncio
async def test_malformed_response_resumes_once_without_duplicate_effect(tmp_path):
    state = InvocationState()
    provider = MalformedThenRecoveredProvider(state)

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await HarnessRunner().run(
            'Process {"request_id":"qa-malformed-001"}.',
            provider="aforge",
            schema=CorrelatedResult,
            cwd=str(tmp_path),
            schema_max_retries=1,
        )

    assert result.is_error is False
    assert result.parsed is not None
    assert result.parsed.request_id == "qa-malformed-001"
    assert result.session_id == "malformed-recovered"
    assert state.calls == 2
    assert state.effects == ["qa-malformed-001"]


@pytest.mark.asyncio
async def test_timeout_fails_closed_without_retry_or_external_effect(tmp_path):
    provider = TimeoutProvider()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await HarnessRunner().run(
            'Process {"request_id":"qa-timeout-001"}.',
            provider="aforge",
            cwd=str(tmp_path),
            max_retries=2,
        )

    assert result.is_error is True
    assert result.parsed is None
    assert "provider timeout" in (result.error_message or "")
    assert result.session_id == "timeout-qa-001"
    assert provider.calls == 1
    assert provider.effects == []


@pytest.mark.asyncio
async def test_cancellation_propagates_without_retry_or_effect(tmp_path):
    provider = CancellationProvider()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        with pytest.raises(asyncio.CancelledError):
            await HarnessRunner().run(
                'Process {"request_id":"qa-cancel-001"}.',
                provider="aforge",
                cwd=str(tmp_path),
                max_retries=1,
            )

    assert provider.calls == 1


@pytest.mark.asyncio
async def test_nil_provider_result_is_a_bounded_failure(tmp_path):
    with patch("agentfield.harness._runner.build_provider", return_value=NilResultProvider()):
        result = await HarnessRunner().run(
            'Process {"request_id":"qa-nil-001"}.',
            provider="aforge",
            cwd=str(tmp_path),
        )

    assert result.is_error is True
    assert result.parsed is None
    assert "nil provider result" in (result.error_message or "").lower()


@pytest.mark.asyncio
async def test_foreign_trace_fails_closed_before_accepting_typed_output(tmp_path):
    with patch("agentfield.harness._runner.build_provider", return_value=ForeignTraceProvider()):
        result = await HarnessRunner().run(
            'Process {"request_id":"qa-request-001"}.',
            provider="aforge",
            schema=CorrelatedResult,
            cwd=str(tmp_path),
        )

    assert result.is_error is True
    assert result.parsed is None
    assert "correlation" in (result.error_message or "").lower()
