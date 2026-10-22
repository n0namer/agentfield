# Basefive R14 — C4 Level 1: system context

Status: CANDIDATE / UNACCEPTED. Date: 2026-10-14.
Target: agentfield at the launcher Git identity `44167dc1050f5bff79abc32fab19c0bb50643093`.

FACT: This is a bounded execution API view, not the whole multi-agent orchestration framework. The caller is a logical actor, not an asserted SDK implementation. Agent dispatch is referenced by `execute.go`; its implementation, network protocol and deployment are outside authorized evidence. Router registration is not visible, so no HTTP URL or method is invented.

```mermaid
flowchart LR
  Caller["Caller / execution client"]
  Agentfield["Agentfield execution system"]
  Agent["External agent target [referenced]"]
  Caller -->|Execution request / status| Agentfield
  Agentfield -->|Agent call [implementation unverified]| Agent
```

## Evidence and navigation

- Level 2: `basefive-r14-container.mmd` separates the Go control plane, logical SQL store and external agent; the engine and physical deployment are unverified.
- Level 3: `basefive-r14-component.mmd` expands optional async idempotency and admission, without treating a process-local queue as durable.
- Machine contract: `basefive-r14-contracts.json` binds every node and edge to a source anchor and exact file SHA-256, and records actual DTO field names, types, JSON tags and omitempty.
- Drift / decision record: `basefive-r14-drift-analysis.md` preserves blind spots and release-gate requirements.
- Executable check: `node --test docs/architecture/basefive-r14-contract.test.mjs` validates source fingerprints, DTO/API signatures, diagram inventory and mutation negatives. This is static source-contract evidence, not Go runtime or HTTP E2E evidence.

## Scope and source of truth

The existing `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md` remains the BMad acceptance source of truth. Its blocked release gate is not lifted by these artifacts. This task supplies the previously missing bounded interface/source bindings; independent QA, native task/result failure-restart evidence and coordinator terminal readback remain separate gates.

FACT: Exact source identities are in the JSON contract and rechecked against file bytes by Node. Imported DTOs such as `types.Execution`, `RunMetadataInput` and webhook event definitions are referenced only; their definitions and wire versions are UNVERIFIED in this scope. No explicit API version appears in the authorized DTO declarations; the documentation contract schema is not a product API version.

Method limits: no shell, graph MCP, Graft MCP, Git inspection, BMad config resolver or unrestricted project scan is authorized/available. The launcher supplies the target root and Git identity. Canonical BMAD-MNNZ supplies advisory skills, not project facts. Quick Dev's extra spec/config/sprint-status writes and interactive workflow steps are not executed because they conflict with the immutable scope and autonomous task contract. No competing PLAN.md or tracker is created.
