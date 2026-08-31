## Learned User Preferences

- Open-source AgentField should prioritize stable APIs and primitives so integrators can build advanced observability themselves; large packaged business or fleet observability belongs in Enterprise.
- The embedded OSS UI should stay a lightweight convenience layer, not the primary surface for org-wide analytics or governance-heavy views.
- Developer-facing observability belongs in OSS; deeper reliability and governance programs may span OSS and Enterprise.
- Avoid empty or placeholder PRs when stacking branches; prefer draft PRs with real implementation, then thorough review before marking ready.
- When designing or documenting control plane behavior, treat YAML configuration (`config/agentfield.yaml` and `AGENTFIELD_CONFIG_FILE`) as a first-class surface alongside environment variables.
- AgentField Desktop targets GitHub-comfortable developers (not infra experts); primary jobs are installing agent nodes from GitHub and seeing runs/cost as a local sub-harness for coding agents.
- Desktop UI should use shared theme tokens rather than hardcoded page styles; treat Agents as a marketplace-style library (installed agents + add), and design Activity for high-volume dense/filterable runs rather than large cards.
- Locked desktop decisions: gold/amber accent; cold-launch to Home when agents exist (add/empty flow when none); usage totals on Home plus Activity per-row when the API allows; keep the update banner across views.

## Learned Workspace Facts

- Monorepo: Go control plane in `control-plane/`, SDKs in `sdk/`, embedded admin UI in `control-plane/web/client/`, Electron desktop app in `desktop/`.
- Agent-node manifests (`agentfield-package.yaml`) carry a `config_version` (schema version, e.g. `v1`; absent = `v0`) that is separate from the node's own `version:`. Bump `config_version` only for breaking format changes, never for additive fields. The single reader is `packages.ParsePackageMetadata` (`control-plane/internal/packages/installer.go`); the authoring contract lives in `docs/installing-agent-nodes.md`.
- Desktop design/product specs live in `DESIGN.md` and `PRODUCT.md`.
- Desktop featured-catalog copy is maintained in `desktop/src/shared/catalog.ts`; post-install descriptions come from each agent's `agentfield-package.yaml` (marketplace cards do not fetch YAML live).

## Fast Verified Engineering

Canonical organization standard: `n0namer/server-ops:docs/standards/FAST_VERIFIED_ENGINEERING.md`.

Optimize engineering for **time-to-verified-running-change**. Before material mutation read `ERRORS.md`, resolve project SoT/DoD, observe exact source and runtime identity where relevant, and define rollback plus required evidence.

Keep source, loaded runtime, execution status, deterministic checks and user-visible outcome separate. Prefer permanent DEV runtime-first for defects that depend on real control-plane/node/config/provider state; prefer exact-SHA Coding Station for source-bound or multi-file work; treat GitHub/CI/deploy as the canonical publication/release boundary rather than the default inner debug loop.

Preserve the exact verified delta between lanes. Validate progressively: `syntax/static -> affected tests -> related regression -> full required suite -> runtime smoke/integration -> semantic/E2E` as applicable. Use execution-scoped evidence/logs before broad logs and correlate by node/version/runtime plus execution/request identity.

Production live editing is forbidden by default. Final status is `DONE | PARTIAL | BLOCKED | FAILED | EVIDENCE_MISSING`; DONE requires every task-specific DoD evidence item. Existing AgentField-specific rules in this file remain stricter where applicable.
