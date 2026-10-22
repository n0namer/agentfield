"""Deterministic AgentField SDK full-cycle and failure-path checks.

RUNTIME_MATERIALIZATION_GAP: this worker cannot execute Python pytest or reach
an authorized Docker control plane. The tests exercise the canonical
``HarnessRunner`` path with disposable providers; native coordinator execution
is required for live node/control-plane confirmation.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from typing import Any
from unittest.mock import patch

import pytest
from pydantic import BaseModel

from agentfield.harness._result import Metrics, RawResult
from agentfield.harness._runner import HarnessRunner


class WaveTaskResult(BaseModel):
    request_id: str
    status: str
    answer: str
    receipt_id: str


@dataclass
class DisposableTaskState:
    calls: int = 0
    effects: list[str] = field(default_factory=list)
    receipts: list[dict[str, str]] = field(default_factory=list)


class ResumingTaskProvider:
    def __init__(self, state: DisposableTaskState) -> None:
        self.state = state
        self.options_seen: list[dict[str, Any]] = []

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.state.calls += 1
        self.options_seen.append(dict(options))
        match = re.search(r"(\S*\.agentfield_output\.json)", prompt)
        assert match, "canonical schema prompt must provide an output path"
        output_path = match.group(1)
        request_id = "wave-local-001"

        if self.state.calls == 1:
            with open(output_path, "w", encoding="utf-8") as output_file:
                json.dump({"request_id": request_id, "status": "pending"}, output_file)
            return RawResult(
                result="",
                metrics=Metrics(session_id="run-001", num_turns=1),
                messages=[{"thread_id": "thread-001", "state": "partial"}],
            )

        assert options.get("resume_session_id") == "run-001"
        if request_id not in self.state.effects:
            self.state.effects.append(request_id)
        receipt = {
            "request_id": request_id,
            "run_id": "run-002",
            "thread_id": "thread-001",
        }
        self.state.receipts.append(receipt)
        with open(output_path, "w", encoding="utf-8") as output_file:
            json.dump(
                {
                    "request_id": request_id,
                    "status": "completed",
                    "answer": "disposable task completed",
                    "receipt_id": "receipt-001",
                },
                output_file,
            )
        return RawResult(
            result="",
            metrics=Metrics(session_id="run-002", num_turns=1),
            messages=[receipt],
        )


class TransportRetryProvider:
    def __init__(self) -> None:
        self.calls = 0
        self.effects: list[str] = []

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.calls += 1
        if self.calls == 1:
            return RawResult(
                is_error=True,
                error_message="connection reset by peer",
                metrics=Metrics(session_id="transport-001"),
            )

        match = re.search(r"(\S*\.agentfield_output\.json)", prompt)
        assert match
        with open(match.group(1), "w", encoding="utf-8") as output_file:
            json.dump(
                {
                    "request_id": "transport-001",
                    "status": "completed",
                    "answer": "retried once",
                    "receipt_id": "receipt-transport-001",
                },
                output_file,
            )
        self.effects.append("transport-001")
        return RawResult(
            metrics=Metrics(session_id="transport-002"),
            messages=[
                {
                    "request_id": "transport-001",
                    "run_id": "transport-002",
                    "thread_id": "thread-transport-001",
                }
            ],
        )


class TimeoutProvider:
    def __init__(self) -> None:
        self.calls = 0
        self.effects: list[str] = []

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.calls += 1
        return RawResult(
            is_error=True,
            error_message="provider timeout",
            metrics=Metrics(session_id="timeout-001"),
        )


@pytest.mark.asyncio
async def test_full_cycle_resumes_partial_task_with_one_durable_effect(tmp_path):
    state = DisposableTaskState()
    provider = ResumingTaskProvider(state)
    runner = HarnessRunner()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await runner.run(
            'Process request {"request_id":"wave-local-001","action":"summarize"}.',
            provider="aforge",
            schema=WaveTaskResult,
            cwd=str(tmp_path),
            schema_max_retries=1,
        )

    assert result.is_error is False
    assert result.parsed == WaveTaskResult(
        request_id="wave-local-001",
        status="completed",
        answer="disposable task completed",
        receipt_id="receipt-001",
    )
    assert state.calls == 2
    assert state.effects == ["wave-local-001"]
    assert state.receipts == [
        {"request_id": "wave-local-001", "run_id": "run-002", "thread_id": "thread-001"}
    ]
    assert result.session_id == "run-002"
    assert result.messages[-1]["thread_id"] == "thread-001"
    assert provider.options_seen[1]["resume_session_id"] == "run-001"


@pytest.mark.asyncio
async def test_transport_failure_retries_without_duplicate_effect(tmp_path, monkeypatch):
    provider = TransportRetryProvider()
    runner = HarnessRunner()
    sleeps: list[float] = []

    async def fake_sleep(delay: float) -> None:
        sleeps.append(delay)

    monkeypatch.setattr("agentfield.harness._runner.asyncio.sleep", fake_sleep)
    monkeypatch.setattr("agentfield.harness._runner.random.uniform", lambda _a, _b: 0.0)

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await runner.run(
            'Process request {"request_id":"transport-001"}.',
            provider="aforge",
            schema=WaveTaskResult,
            cwd=str(tmp_path),
            max_retries=1,
            initial_delay=0.01,
        )

    assert result.is_error is False
    assert result.parsed is not None
    assert provider.calls == 2
    assert provider.effects == ["transport-001"]
    assert sleeps == [0.01]


@pytest.mark.asyncio
async def test_timeout_provider_failure_fails_closed_without_resume_or_effect(tmp_path):
    provider = TimeoutProvider()
    runner = HarnessRunner()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await runner.run(
            'Process request {"request_id":"timeout-001"}.',
            provider="aforge",
            cwd=str(tmp_path),
            max_retries=2,
        )

    assert result.is_error is True
    assert result.parsed is None
    assert "provider timeout" in (result.error_message or "")
    assert provider.calls == 1
    assert provider.effects == []
    assert result.session_id == "timeout-001"
