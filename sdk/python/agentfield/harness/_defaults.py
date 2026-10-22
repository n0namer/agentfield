"""Default harness provider selection.

Provider/model/variant precedence is explicit value > the corresponding
environment variable > the platform default.
"""

from __future__ import annotations

import os
import sys
from collections.abc import Mapping
from typing import Any, Optional

WINDOWS_HARNESS_PROVIDER = "codex"
UNIX_HARNESS_PROVIDER = "opencode"
WINDOWS_HARNESS_MODEL = "gpt-5.6-luna"
UNIX_HARNESS_MODEL = "broker/fast-coding"
WINDOWS_HARNESS_VARIANT = "low"
UNIX_HARNESS_VARIANT = None
# Kept as a compatibility name for callers that imported the old constant;
# resolution itself is platform-aware and does not consume this legacy value.
DEFAULT_HARNESS_PROVIDER = "aforge"
HARNESS_PROVIDER_ENV_VAR = "AGENTFIELD_HARNESS_PROVIDER"
HARNESS_MODEL_ENV_VAR = "AGENTFIELD_HARNESS_MODEL"
HARNESS_VARIANT_ENV_VAR = "AGENTFIELD_HARNESS_VARIANT"


def _resolve_string(
    explicit: Optional[str],
    env_vars: tuple[str, ...],
    environ: Mapping[str, str] = os.environ,
) -> Optional[str]:
    if isinstance(explicit, str) and explicit.strip():
        return explicit.strip()
    for env_var in env_vars:
        env_value = environ.get(env_var)
        if isinstance(env_value, str) and env_value.strip():
            return env_value.strip()
    return None


def platform_default_harness_provider(platform: Optional[str] = None) -> str:
    """Return the implicit harness provider for a target Python platform."""
    return (
        WINDOWS_HARNESS_PROVIDER
        if (platform or sys.platform).lower().startswith("win")
        else UNIX_HARNESS_PROVIDER
    )


def resolve_harness_provider(
    explicit: Optional[str] = None,
    *,
    platform: Optional[str] = None,
    environ: Optional[Mapping[str, str]] = None,
) -> str:
    return _resolve_string(
        explicit,
        (HARNESS_PROVIDER_ENV_VAR,),
        os.environ if environ is None else environ,
    ) or platform_default_harness_provider(platform)


def platform_default_harness_model(platform: Optional[str] = None) -> str:
    """Return the implicit model for a target Python platform."""
    return (
        WINDOWS_HARNESS_MODEL
        if (platform or sys.platform).lower().startswith("win")
        else UNIX_HARNESS_MODEL
    )


def platform_default_harness_variant(platform: Optional[str] = None) -> Optional[str]:
    """Return the implicit reasoning variant for a target Python platform."""
    return (
        WINDOWS_HARNESS_VARIANT
        if (platform or sys.platform).lower().startswith("win")
        else UNIX_HARNESS_VARIANT
    )


def resolve_harness_model(
    explicit: Optional[str] = None,
    *,
    provider: Optional[str] = None,
    platform: Optional[str] = None,
    environ: Optional[Mapping[str, str]] = None,
) -> Optional[str]:
    value = _resolve_string(
        explicit,
        (HARNESS_MODEL_ENV_VAR, "HARNESS_MODEL"),
        os.environ if environ is None else environ,
    )
    if value:
        return value
    if provider is None or provider == platform_default_harness_provider(platform):
        return platform_default_harness_model(platform)
    return None


def resolve_harness_variant(
    explicit: Optional[str] = None,
    *,
    provider: Optional[str] = None,
    platform: Optional[str] = None,
    environ: Optional[Mapping[str, str]] = None,
) -> Optional[str]:
    value = _resolve_string(
        explicit,
        (HARNESS_VARIANT_ENV_VAR, "HARNESS_VARIANT"),
        os.environ if environ is None else environ,
    )
    if value:
        return value
    if provider is None or provider == platform_default_harness_provider(platform):
        return platform_default_harness_variant(platform)
    return None


def resolve_harness_options(
    provider: Optional[str] = None,
    model: Optional[str] = None,
    variant: Optional[str] = None,
    *,
    platform: Optional[str] = None,
    environ: Optional[Mapping[str, str]] = None,
) -> dict[str, Any]:
    """Resolve harness defaults without coupling the logic to the host platform.

    Provider resolution happens first. Platform model/variant defaults are only
    eligible when provider was omitted (the implicit platform provider) or when
    it resolves to that provider. A selected non-platform provider retains its
    own defaults when model/variant are unset.
    """
    resolved_provider = resolve_harness_provider(
        provider, platform=platform, environ=environ
    )
    return {
        "provider": resolved_provider,
        "model": resolve_harness_model(
            model,
            provider=resolved_provider,
            platform=platform,
            environ=environ,
        ),
        "variant": resolve_harness_variant(
            variant,
            provider=resolved_provider,
            platform=platform,
            environ=environ,
        ),
    }
