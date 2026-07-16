# Complexity campaign ownership

Updated: 2026-07-16
Base: `origin/main@4c1029d1177f797f11bcd560f14a18305a2e7353`

## Live external ownership

| PR | Head | Files | Collision |
|---|---|---|---|
| [#9](https://github.com/Pantani/relayer/pull/9) | `Pantani/cx/fix-shared-ci@388d661` | `Dockerfile`, `interchaintest/feegrant_test.go`, `interchaintest/misbehaviour_test.go`, `relayer/chains/cosmos/codec.go` | none with cli-start; revalidate Cosmos waves after #9 lands |
| [#10](https://github.com/Pantani/relayer/pull/10) (draft) | `Pantani/cx/complexity-cli-version@b91cb7d` | `cmd/version.go`, `cmd/version_test.go`, harness/state | maximum-10 campaign base; no collision with cli-start |
| [#11](https://github.com/Pantani/relayer/pull/11) (draft) | `Pantani/cx/complexity-cli-config@ee1d834` | `cmd/config.go`, `cmd/appstate.go`, characterization/state | direct base for cli-start; no collision with #9/#10 |

## Campaign ownership

Exclusive worktree lease: `complexity-orchestrator` is the only `ACTIVE` editor for cli-start integration and publication. Handoff requires the previous holder to be `COMPLETED`, `IDLE`, or `INTERRUPTED` first.

| Subwave | Owner | Writable files | Worktree | Status |
|---|---|---|---|---|
| campaign integration | complexity-orchestrator | `.claude/**`, `CLAUDE.md`, `_workspace/complexity/ledger.md`, `_workspace/complexity/ownership.md`, `_workspace/complexity/inventory.md`, `_workspace/complexity/plan.md` and Git integration | `/Users/pantani/.codex/worktrees/complexity-cli-start/relayer` | active; integrating approved cli-start subwave |
| cli-version characterization | complexity-characterization-engineer | `cmd/version_test.go`, `_workspace/complexity/characterization/cli-version.md` | same worktree under exclusive sequential lease | completed |
| cli-version production | complexity-engineer | `cmd/version.go` only | same worktree, after characterization and harness approval | completed |
| cli-version review | complexity-verifier | `_workspace/complexity/reviews/cli-version.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| harness validation | complexity-verifier | `_workspace/complexity/harness-validation.md` only | same worktree under exclusive sequential lease | completed; APPROVED for maximum 10 |
| cli-config characterization | complexity-characterization-engineer | characterization tests for `cmd/config.go` and `cmd/appstate.go`; `_workspace/complexity/characterization/cli-config.md` | `/Users/pantani/.codex/worktrees/complexity-cli-config/relayer` | completed; 31 focused cases and race pass |
| cli-config production | complexity-engineer | `cmd/config.go`, `cmd/appstate.go` only | same worktree, after characterization approval | completed; projected `62/102/105` |
| cli-config review | complexity-verifier | `_workspace/complexity/reviews/cli-config.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| cli-start characterization | complexity-characterization-engineer | characterization tests for `cmd/start.go` and `cmd/flags.go`; `_workspace/complexity/characterization/cli-start.md` | `/Users/pantani/.codex/worktrees/complexity-cli-start/relayer` | completed; 20 scenarios, 47 focused/race checks, package 138 pass |
| cli-start production | complexity-engineer | `cmd/start.go`, `cmd/flags.go` only | same worktree, after characterization approval | completed; targets `1/0`, `4/3`, `5/4`; helpers max `9/9` |
| cli-start review | complexity-verifier | `_workspace/complexity/reviews/cli-start.md` only | same worktree under exclusive sequential lease | completed; APPROVED |

Two agents must never edit the same file or worktree concurrently. Exactly one editor row may be `active`. Git integration belongs only to `complexity-orchestrator`.
