from __future__ import annotations

from typing import Any
from unittest.mock import patch

import pytest

from agentfield.harness._result import RawResult
from agentfield.harness._runner import HarnessRunner
from agentfield.harness._defaults import (
    DEFAULT_HARNESS_PROVIDER,
    platform_default_harness_model,
    platform_default_harness_provider,
    platform_default_harness_variant,
    resolve_harness_options,
    resolve_harness_model,
    resolve_harness_provider,
    resolve_harness_variant,
)
from agentfield.types import HarnessConfig


class _Provider:
    async def execute(self, prompt: str, options: dict[str, Any]) -> RawResult:
        return RawResult(result="ok")


def test_default_provider_is_platform_default(monkeypatch):
    monkeypatch.delenv("AGENTFIELD_HARNESS_PROVIDER", raising=False)
    assert HarnessConfig().provider == platform_default_harness_provider()


def test_legacy_default_provider_constant_does_not_drive_resolution(monkeypatch):
    monkeypatch.delenv("AGENTFIELD_HARNESS_PROVIDER", raising=False)
    assert DEFAULT_HARNESS_PROVIDER == "aforge"
    assert resolve_harness_provider(platform="win32", environ={}) == "codex"
    assert resolve_harness_provider(platform="linux", environ={}) == "opencode"


@pytest.mark.parametrize(
    ("platform", "expected"),
    [("win32", "codex"), ("linux", "opencode"), ("darwin", "opencode")],
)
def test_implicit_provider_matches_platform_matrix(monkeypatch, platform, expected):
    monkeypatch.delenv("AGENTFIELD_HARNESS_PROVIDER", raising=False)
    monkeypatch.setattr("agentfield.harness._defaults.sys.platform", platform)
    assert resolve_harness_provider() == expected


def test_explicit_provider_beats_platform_and_environment(monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "codex")
    monkeypatch.setattr("agentfield.harness._defaults.sys.platform", "win32")
    assert resolve_harness_provider(" gemini ") == "gemini"


def test_env_provider_is_honoured(monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "codex")
    assert HarnessConfig().provider == "codex"


def test_explicit_provider_beats_env(monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "codex")
    assert HarnessConfig(provider="gemini").provider == "gemini"


def test_blank_env_is_ignored(monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "   ")
    assert HarnessConfig().provider == platform_default_harness_provider()


@pytest.mark.parametrize(
    ("platform", "model", "variant"),
    [("win32", "gpt-5.6-luna", "low"), ("linux", "broker/fast-coding", None)],
)
def test_platform_model_and_variant_defaults(monkeypatch, platform, model, variant):
    monkeypatch.delenv("AGENTFIELD_HARNESS_MODEL", raising=False)
    monkeypatch.delenv("AGENTFIELD_HARNESS_VARIANT", raising=False)
    monkeypatch.delenv("HARNESS_MODEL", raising=False)
    monkeypatch.delenv("HARNESS_VARIANT", raising=False)
    monkeypatch.setattr("agentfield.harness._defaults.sys.platform", platform)
    assert resolve_harness_model() == model
    assert resolve_harness_variant() == variant
    assert platform_default_harness_model() == model
    assert platform_default_harness_variant() == variant


def test_model_and_variant_explicit_values_beat_environment(monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_MODEL", "env/model")
    monkeypatch.setenv("AGENTFIELD_HARNESS_VARIANT", "high")
    assert resolve_harness_model(" config/model ") == "config/model"
    assert resolve_harness_variant(" medium ") == "medium"


def test_model_and_variant_environment_aliases_are_honoured(monkeypatch):
    monkeypatch.delenv("AGENTFIELD_HARNESS_MODEL", raising=False)
    monkeypatch.delenv("AGENTFIELD_HARNESS_VARIANT", raising=False)
    monkeypatch.setenv("HARNESS_MODEL", "env/model")
    monkeypatch.setenv("HARNESS_VARIANT", "high")
    assert resolve_harness_model() == "env/model"
    assert resolve_harness_variant() == "high"


def test_non_platform_provider_keeps_provider_owned_unset_defaults(monkeypatch):
    monkeypatch.setattr("agentfield.harness._defaults.sys.platform", "win32")
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "opencode")
    monkeypatch.delenv("AGENTFIELD_HARNESS_MODEL", raising=False)
    monkeypatch.delenv("AGENTFIELD_HARNESS_VARIANT", raising=False)
    assert resolve_harness_options() == {
        "provider": "opencode",
        "model": None,
        "variant": None,
    }


def test_resolver_injects_platform_without_reading_host_platform_or_environment():
    assert resolve_harness_options(
        platform="win32", environ={},
    ) == {
        "provider": "codex",
        "model": "gpt-5.6-luna",
        "variant": "low",
    }
    assert resolve_harness_options(
        provider="opencode", platform="win32", environ={},
    ) == {"provider": "opencode", "model": None, "variant": None}


async def _run_and_capture_options(
    tmp_path, config: HarnessConfig | None = None, **overrides: Any
) -> dict[str, Any]:
    captured: dict[str, Any] = {}

    def fake_build_provider(factory_config):
        captured["provider"] = factory_config.provider
        captured["model"] = factory_config.model
        captured["variant"] = factory_config.variant
        return _Provider()

    with patch("agentfield.harness._runner.build_provider", fake_build_provider):
        await HarnessRunner(config=config).run(
            "hello", cwd=str(tmp_path), **overrides
        )
    return captured


async def _run_and_capture_provider(
    tmp_path, config: HarnessConfig | None = None, **overrides: Any
) -> str:
    return (await _run_and_capture_options(tmp_path, config, **overrides))["provider"]


@pytest.mark.asyncio
async def test_zero_setup_runner_selects_platform_default(tmp_path, monkeypatch):
    monkeypatch.delenv("AGENTFIELD_HARNESS_PROVIDER", raising=False)
    assert (
        await _run_and_capture_provider(tmp_path)
        == platform_default_harness_provider()
    )


@pytest.mark.asyncio
async def test_runner_honours_env_provider(tmp_path, monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "opencode")
    assert await _run_and_capture_provider(tmp_path) == "opencode"


@pytest.mark.asyncio
async def test_runner_explicit_provider_beats_env_and_config(tmp_path, monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "gemini")
    config = HarnessConfig(provider="codex")
    assert (
        await _run_and_capture_provider(tmp_path, config, provider="opencode")
        == "opencode"
    )


@pytest.mark.asyncio
async def test_runner_passes_platform_model_and_variant_to_provider(tmp_path, monkeypatch):
    monkeypatch.delenv("AGENTFIELD_HARNESS_PROVIDER", raising=False)
    monkeypatch.delenv("AGENTFIELD_HARNESS_MODEL", raising=False)
    monkeypatch.delenv("AGENTFIELD_HARNESS_VARIANT", raising=False)
    monkeypatch.setattr("agentfield.harness._defaults.sys.platform", "win32")
    captured = await _run_and_capture_options(tmp_path)
    assert captured["provider"] == "codex"
    assert captured["model"] == "gpt-5.6-luna"
    assert captured["variant"] == "low"


@pytest.mark.asyncio
async def test_runner_does_not_inject_platform_defaults_for_env_provider(
    tmp_path, monkeypatch
):
    monkeypatch.setattr("agentfield.harness._defaults.sys.platform", "win32")
    monkeypatch.setenv("AGENTFIELD_HARNESS_PROVIDER", "opencode")
    monkeypatch.delenv("AGENTFIELD_HARNESS_MODEL", raising=False)
    monkeypatch.delenv("AGENTFIELD_HARNESS_VARIANT", raising=False)
    captured = await _run_and_capture_options(tmp_path)
    assert captured == {"provider": "opencode", "model": None, "variant": None}


@pytest.mark.asyncio
async def test_runner_explicit_model_variant_beat_env(tmp_path, monkeypatch):
    monkeypatch.setenv("AGENTFIELD_HARNESS_MODEL", "env/model")
    monkeypatch.setenv("AGENTFIELD_HARNESS_VARIANT", "high")
    captured = await _run_and_capture_options(
        tmp_path, model="config/model", variant="medium"
    )
    assert captured["model"] == "config/model"
    assert captured["variant"] == "medium"
