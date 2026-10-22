---
title: 'Platform-aware harness defaults and Windows runtime smoke'
type: 'feature'
created: '2026-09-29'
baseline_commit: '44167dc1050f5bff79abc32fab19c0bb50643093'
status: 'in-review'
route: 'dispatch'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/docs/harness-providers.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** AgentField harness defaults still need one accepted cross-SDK state: on Windows, zero-configuration harness execution must use Codex/OpenCodex with `gpt-5.6-luna` and reasoning `low`; on Unix/Linux, implicit routing must preserve the existing OpenCode/FCM fast-coding path. Configuration changes and unit tests are insufficient without a real Windows harness execution.

**Approach:** Consolidate the already-developed Go and Python candidates into one clean integration worktree, preserve explicit invocation/config/environment overrides above platform defaults, independently review the integrated diff, then prove a real structured Codex harness completion on A55 using the curated Factory Codex home.

## Boundaries & Constraints

**Always:** Preserve explicit provider/model/variant values; preserve `AGENTFIELD_HARNESS_PROVIDER` and relevant model/variant environment overrides; apply platform model/reasoning defaults only to the implicit platform route; keep `Agent.ai()` semantics separate from `Agent.harness()`; use curated `ruflo/codex-home` for Windows smoke; preserve the legacy Python `DEFAULT_HARNESS_PROVIDER = "aforge"` compatibility constant without letting it drive runtime resolution.

**Never:** Modify unrelated control-plane/desktop dirty files; accept generated `.claude-flow`, `.graphify-code`, `.swarm`, or `graft` artifacts as product changes; change Ruflo/Autopilot code; deploy, publish, or open a PR before GREEN.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Windows implicit | provider/model/variant unset | Codex + gpt-5.6-luna + low | Fail visibly if Codex/OpenCodex unavailable |
| Unix implicit | provider/model/variant unset | OpenCode + existing FCM fast-coding model; no invented variant | Preserve existing provider-owned behavior |
| Explicit provider | provider explicitly set | Explicit provider wins; platform model/variant not forced onto a non-platform provider | Reject unknown provider through existing validation |
| Explicit model/variant | provider implicit or explicit | Explicit model/variant wins over environment/platform | Existing provider validation remains authoritative |
| Environment override | supported harness env vars set | Environment values beat platform defaults | Blank values are treated as unset |

</frozen-after-approval>

## Code Map

- `sdk/go/harness/factory.go` -- provider precedence and implicit platform provider.
- `sdk/go/harness/provider.go` -- Go platform provider/model/variant resolver.
- `sdk/go/harness/runner.go` -- actual dispatch wiring; resolved options must reach provider construction.
- `sdk/go/harness/factory_test.go`, `platform_defaults_test.go`, `runner_test.go` -- Go precedence and runtime wiring coverage.
- `sdk/python/agentfield/harness/_defaults.py` -- provider-aware Python platform resolver and legacy constant compatibility.
- `sdk/python/agentfield/harness/_runner.py` -- Python runtime propagation of provider/model/variant.
- `sdk/python/agentfield/harness/providers/_factory.py` and `agentfield/types.py` -- provider factory/config surface.
- `sdk/python/tests/test_harness_defaults.py`, `test_harness_factory.py` -- Python precedence, compatibility, and propagation coverage.

## Tasks & Acceptance

**Execution:**
- [ ] Consolidate only the accepted Go routing diff into a clean AgentField integration worktree.
- [ ] Consolidate only the accepted Python routing diff into the same integration worktree.
- [ ] Run focused and relevant regression suites for both SDKs; reject out-of-scope/generated artifacts.
- [ ] Perform fresh adversarial and edge-case review on the integrated diff and repair load-bearing findings.
- [ ] Run a real Windows AgentField harness smoke through Codex/OpenCodex using `gpt-5.6-luna`, `low`, and structured output.
- [ ] Update this BMad artifact with verification evidence and exact commit.

**Acceptance Criteria:**
- Given Windows with no explicit harness overrides, when AgentField resolves and executes a harness call, then Codex is invoked with `gpt-5.6-luna` and reasoning `low`.
- Given Unix/Linux with no explicit overrides, when harness defaults resolve, then the existing OpenCode/FCM fast-coding route is selected without forcing a reasoning variant.
- Given explicit provider/model/variant or supported environment overrides, when defaults resolve, then the explicit value wins.
- Given the integrated candidate, when focused Go and Python tests and independent review run, then no load-bearing regression remains.
- Given curated OpenCodex is available on A55, when a bounded structured harness prompt is executed, then AgentField returns a substantive schema-valid completion.

## Implementation Notes

## Spec Change Log

## Review Triage Log

## Verification

**Commands:**
- `go test ./harness -run "TestPlatformDefaults|TestResolveProviderName|TestRunner_Run_NoSchema" -count=1` from `sdk/go` using the pinned Ruflo Go toolchain -- expected: PASS.
- `uv run --extra dev python -m pytest -q tests/test_harness_defaults.py tests/test_harness_factory.py` from `sdk/python` -- expected: PASS.
- `git diff --check` -- expected: no whitespace errors.
- Bounded real harness smoke with curated `CODEX_HOME` -- expected: provider=codex, model=gpt-5.6-luna, reasoning=low, schema-valid result.
