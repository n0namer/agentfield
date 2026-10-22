"""Bounded offline harness fault-cycle and recovery checks.

These tests exercise the Python SDK runner with disposable providers. They prove
SDK-level retry, schema, cancellation, and receipt behavior only; they do not
prove a live native control-plane execution.
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

from agentfield.harness._defaults import resolve_harness_provider
from agentfield.harness._result import FailureType, Metrics, RawResult
from agentfield.harness._runner import HarnessRunner


class WaveTaskResult(BaseModel):
    request_id: str
    status: str
    answer: str
    receipt_id: str


@dataclass
class ReceiptState:
    calls: int = 0
    effects: list[str] = field(default_factory=list)
    receipts: list[dict[str, str]] = field(default_factory=list)


def _output_path(prompt: str) -> str:
    match = re.search(r"(\S*\.agentfield_output\.json)", prompt)
    assert match, "canonical schema prompt must provide an output path"
    return match.group(1)


def _write_result(prompt: str, **payload: str) -> None:
    with open(_output_path(prompt), "w", encoding="utf-8") as output_file:
        json.dump(payload, output_file)


class RecoveryProvider:
    def __init__(self, state: ReceiptState, mode: str) -> None:
        self.state = state
        self.mode = mode
        self.options_seen: list[dict[str, Any]] = []

    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        self.state.calls += 1
        self.options_seen.append(dict(options))
        request_id = "cycle-001"

        if self.mode == "malformed" and self.state.calls == 1:
            with open(_output_path(prompt), "w", encoding="utf-8") as output_file:
                output_file.write('{"request_id":')
            return RawResult(metrics=Metrics(session_id="malformed-001"))

        if self.mode == "crash" and self.state.calls == 1:
            return RawResult(
                is_error=True,
                error_message="provider process crashed",
                failure_type=FailureType.CRASH,
                metrics=Metrics(session_id="crash-001"),
            )

        if self.mode == "malformed":
            assert options.get("resume_session_id") == "malformed-001"
        if self.mode == "crash":
            assert "resume_session_id" not in options

        if request_id not in self.state.effects:
            self.state.effects.append(request_id)
        receipt = {
            "request_id": request_id,
            "run_id": f"run-{self.state.calls:03d}",
            "thread_id": "thread-cycle-001",
        }
        self.state.receipts.append(receipt)
        _write_result(
            prompt,
            request_id=request_id,
            status="completed",
            answer="recovered once",
            receipt_id="receipt-cycle-001",
        )
        return RawResult(
            metrics=Metrics(session_id=f"run-{self.state.calls:03d}"),
            messages=[receipt],
        )


class TransportProvider:
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
        _write_result(
            prompt,
            request_id="transport-001",
            status="completed",
            answer="retried once",
            receipt_id="receipt-transport-001",
        )
        self.effects.append("transport-001")
        return RawResult(metrics=Metrics(session_id="transport-002"))


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


class CancellationProvider:
    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        raise asyncio.CancelledError()


@pytest.mark.parametrize(
    ("explicit", "environment", "expected"),
    [
        ("codex", "gemini", "codex"),
        (None, " gemini ", "gemini"),
        (None, None, "aforge"),
    ],
)
def test_provider_selection_precedence(
    explicit: str | None,
    environment: str | None,
    expected: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    if environment is None:
        monkeypatch.delenv("AGENTFIELD_HARNESS_PROVIDER", raising=False)
    else:
        monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", environment)

    assert resolve_harness_provider(explicit) == expected


@pytest.mark.asyncio
async def test_unknown_provider_is_rejected_at_factory_boundary(tmp_path):
    runner = HarnessRunner()

    with pytest.raises(ValueError):
        await runner.run(
            "Run a disposable task.",
            provider="provider-that-does-not-exist",
            cwd=str(tmp_path),
        )


@pytest.mark.asyncio
@pytest.mark.parametrize("mode", ["malformed", "crash"])
async def test_schema_recovery_replays_without_duplicate_effects(tmp_path, mode):
    state = ReceiptState()
    provider = RecoveryProvider(state, mode)
    runner = HarnessRunner()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await runner.run(
            'Process {"request_id":"cycle-001"}.',
            provider="aforge",
            schema=WaveTaskResult,
            cwd=str(tmp_path),
            schema_max_retries=1,
        )

    assert result.is_error is False
    assert result.parsed == WaveTaskResult(
        request_id="cycle-001",
        status="completed",
        answer="recovered once",
        receipt_id="receipt-cycle-001",
    )
    assert state.calls == 2
    assert state.effects == ["cycle-001"]
    assert len(state.receipts) == 1
    assert state.receipts[0]["thread_id"] == "thread-cycle-001"


@pytest.mark.asyncio
async def test_lost_network_retries_once_and_preserves_single_effect(tmp_path, monkeypatch):
    provider = TransportProvider()
    runner = HarnessRunner()
    sleeps: list[float] = []

    async def fake_sleep(delay: float) -> None:
        sleeps.append(delay)

    monkeypatch.setattr("agentfield.harness._runner.asyncio.sleep", fake_sleep)
    monkeypatch.setattr("agentfield.harness._runner.random.uniform", lambda _a, _b: 0.0)

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await runner.run(
            'Process {"request_id":"transport-001"}.',
            provider="aforge",
            schema=WaveTaskResult,
            cwd=str(tmp_path),
            max_retries=1,
            initial_delay=0.01,
        )

    assert result.is_error is False
    assert result.parsed is not None
    assert result.session_id == "transport-002"
    assert provider.calls == 2
    assert provider.effects == ["transport-001"]
    assert sleeps == [0.01]


@pytest.mark.asyncio
async def test_timeout_fails_closed_without_retry_or_effect(tmp_path):
    provider = TimeoutProvider()
    runner = HarnessRunner()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        result = await runner.run(
            'Process {"request_id":"timeout-001"}.',
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


@pytest.mark.asyncio
async def test_cancellation_propagates_without_creating_effect(tmp_path):
    runner = HarnessRunner()
    provider = CancellationProvider()

    with patch("agentfield.harness._runner.build_provider", return_value=provider):
        with pytest.raises(asyncio.CancelledError):
            await runner.run(
                'Process {"request_id":"cancel-001"}.',
                provider="aforge",
                cwd=str(tmp_path),
            )
