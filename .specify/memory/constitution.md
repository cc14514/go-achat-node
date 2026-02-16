<!--
Sync Impact Report

- Version change: UNVERSIONED (template) -> 0.1.0
- Modified principles: N/A (initial constitution)
- Added sections: N/A (filled existing template sections)
- Removed sections: N/A
- Templates requiring updates:
  - ✅ updated: .specify/templates/plan-template.md
  - ✅ updated: .specify/templates/tasks-template.md
  - ✅ no change needed: .specify/templates/spec-template.md
  - N/A (not present): .specify/templates/commands/*.md
- Runtime guidance updated:
  - ✅ updated: docs/development-guide.zh-CN.md
-->

# go-achat-node Constitution

## Core Principles

### I. Go-First, Module-Aware

- The codebase MUST remain buildable with Go 1.22+.
- The repository MUST keep the two-module structure intact:
  - Root module: `github.com/cc14514/go-achat-node` (core library)
  - App module: `app/achat` (reference node CLI)
- Shared types and core behavior MUST live in the root module; `app/achat` MUST depend on the
  root module rather than duplicating logic.

Rationale: keeps the core library reusable while preserving a working reference node.

### II. Clear Boundary: P2P Core vs Local RPC Gateway

- P2P protocol handling MUST remain in the core service layer (e.g., `ChatService`).
- Local RPC (HTTP + WebSocket) MUST remain a thin gateway around the core service.
- Any new external interface (CLI flags, RPC methods, WebSocket payloads) MUST be documented.

Rationale: prevents transport concerns from leaking into core logic and keeps integration stable.

### III. Tests For Behavior Changes (Non-Negotiable)

- A bug fix MUST include a test that fails before the fix and passes after.
- Any change to an RPC contract or P2P protocol behavior MUST include an automated test (unit or
  integration) covering the new/changed behavior.
- If tests are intentionally omitted, the change MUST include a written waiver in the feature
  spec/plan explaining why tests are impractical and how the change is verified instead.

Rationale: preserves protocol and gateway correctness as the primary public surface.

### IV. Compatibility & Versioning Discipline

- Public-facing changes MUST be versioned and communicated:
  - CLI flag behavior changes MUST be reflected in `--help` output and docs.
  - RPC contract changes MUST be documented and, if breaking, require a migration note.
  - P2P protocol identifiers that embed versions MUST NOT be changed silently.
- Backward compatibility SHOULD be preserved by default; breaking changes MUST be explicit.

Rationale: consumers depend on stable contracts more than internal structure.

### V. Operational Safety: Secure Defaults, Simple Surfaces

- The default deployment posture MUST remain safe for local development:
  - RPC binds to loopback by default.
  - Local persistence uses on-disk storage (LevelDB) under a configurable home directory.
- Production deployments MUST set a non-empty `--pwd` (or equivalent) and MUST use distinct
  `--homedir` per node instance.
- Changes MUST prefer the simplest working design; complexity MUST be justified in the plan.

Rationale: reduces accidental exposure and keeps the node reliable on constrained environments.

## Project Constraints

- **Language**: Go 1.22+.
- **Storage**: LevelDB for local persistence.
- **Networking**: libp2p-based P2P; local RPC over HTTP + WebSocket.
- **Build**: `make build` produces `dist/achat`; Docker image is produced via `make docker`.
- **Portability**: `CGO_ENABLED=0` builds are preferred for container/embedded portability.

## Development Workflow

- **Formatting**: Go code MUST be formatted with `gofmt`.
- **Testing**: Changes MUST run relevant tests:
  - Root module: `go test ./...`
  - App module: `cd app/achat && go test ./...`
- **Docs**: Any change to CLI flags, RPC methods, or protocol behavior MUST update user-facing
  docs (`README.md`, `docs/development-guide.zh-CN.md`, and feature specs as applicable).
- **Release readiness**: Before publishing images/binaries, `make build` MUST succeed; Docker
  changes MUST build via `make docker`.

## Governance

- **Authority**: This constitution supersedes feature specs, plans, and task lists.
- **Amendments**:
  - Any change MUST update this file and include a brief rationale in the Sync Impact Report.
  - Breaking governance changes (principle removals or redefinitions) MUST bump MAJOR.
  - Material new principles/sections MUST bump MINOR.
  - Clarifications/typos MUST bump PATCH.
- **Compliance review**: Every PR/review MUST explicitly check for constitution compliance
  (tests, versioning, docs, security posture).
- **Guidance sources**: Runtime/developer guidance lives in `README.md` and
  `docs/development-guide.zh-CN.md`.

**Version**: 0.1.0 | **Ratified**: 2026-02-16 | **Last Amended**: 2026-02-16
