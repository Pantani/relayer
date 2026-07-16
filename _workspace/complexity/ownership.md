# Complexity campaign ownership

Updated: 2026-07-16
Base: `origin/main@4c1029d1177f797f11bcd560f14a18305a2e7353`

## Live external ownership

| PR | Head | Files | Collision |
|---|---|---|---|
| [#9](https://github.com/Pantani/relayer/pull/9) | `Pantani/cx/fix-shared-ci@b4a1b1d` | `Dockerfile`, `interchaintest/misbehaviour_test.go`, `relayer/chains/cosmos/codec.go` | none with cli-version/harness |
| [#10](https://github.com/Pantani/relayer/pull/10) (draft) | `Pantani/cx/complexity-cli-version@c30f6cb` | `cmd/version.go`, `cmd/version_test.go`, harness/state | campaign subwave; no collision with #9 |

## Campaign ownership

Exclusive worktree lease: `complexity-orchestrator` is the only `ACTIVE` editor for maximum-10 integration and publication. Handoff requires the previous holder to be `COMPLETED`, `IDLE`, or `INTERRUPTED` first.

| Subwave | Owner | Writable files | Worktree | Status |
|---|---|---|---|---|
| campaign integration | complexity-orchestrator | `.claude/**`, `CLAUDE.md`, `_workspace/complexity/ledger.md`, `_workspace/complexity/ownership.md`, `_workspace/complexity/inventory.md` and Git integration | `/Users/pantani/.codex/worktrees/complexity-cli-version/relayer` | active; publishing approved maximum-10 update to PR #10 |
| cli-version characterization | complexity-characterization-engineer | `cmd/version_test.go`, `_workspace/complexity/characterization/cli-version.md` | same worktree under exclusive sequential lease | completed |
| cli-version production | complexity-engineer | `cmd/version.go` only | same worktree, after characterization and harness approval | completed |
| cli-version review | complexity-verifier | `_workspace/complexity/reviews/cli-version.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| harness validation | complexity-verifier | `_workspace/complexity/harness-validation.md` only | same worktree under exclusive sequential lease | completed; APPROVED for maximum 10 |

Two agents must never edit the same file or worktree concurrently. Exactly one editor row may be `active`. Git integration belongs only to `complexity-orchestrator`.
