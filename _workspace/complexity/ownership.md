# Complexity campaign ownership

Updated: 2026-07-16
Base: `origin/main@43612fc928c447d70c7179b21912e38c55761cfa`

## Live external ownership

| PR | Head | Files | Collision |
|---|---|---|---|
| [#9](https://github.com/Pantani/relayer/pull/9) (merged) | `43612fc` | `Dockerfile`, `interchaintest/feegrant_test.go`, `interchaintest/misbehaviour_test.go`, `relayer/chains/cosmos/codec.go`, `relayer/chains/cosmos/provider.go` | absorbed through #16/#17; Cosmos liveliness revalidated at `9/25` |
| [#10](https://github.com/Pantani/relayer/pull/10) (draft) | `Pantani/cx/complexity-cli-version@b91cb7d` | `cmd/version.go`, `cmd/version_test.go`, harness/state | maximum-10 campaign base; no collision with cli-start |
| [#11](https://github.com/Pantani/relayer/pull/11) (draft) | `Pantani/cx/complexity-cli-config@ee1d834` | `cmd/config.go`, `cmd/appstate.go`, characterization/state | direct base for cli-start; no collision with #9/#10 |
| [#12](https://github.com/Pantani/relayer/pull/12) (draft) | `Pantani/cx/complexity-cli-start@56b54b8` | `cmd/start.go`, `cmd/flags.go`, characterization/state | direct base for cli-chains; no collision with #9/#10/#11 |
| [#13](https://github.com/Pantani/relayer/pull/13) (draft) | `Pantani/cx/complexity-cli-chains@c5f2735` | `cmd/chains.go`, characterization/state | direct base for cli-paths; no collision with #9/#10/#11/#12 |
| [#14](https://github.com/Pantani/relayer/pull/14) (draft) | `Pantani/cx/complexity-cli-paths@714750d` | `cmd/paths.go`, characterization/state | direct base for cli-feegrant; no collision with #9/#10/#11/#12/#13 |
| [#15](https://github.com/Pantani/relayer/pull/15) (draft) | `Pantani/cx/complexity-cli-feegrant@7b5e7d1` | `cmd/feegrant.go`, characterization/state | direct base for cli-query; no collision with #9/#10/#11/#12/#13/#14 |
| [#16](https://github.com/Pantani/relayer/pull/16) (draft) | `Pantani/cx/complexity-cli-query` | `cmd/query.go`, characterization/state | merged updated #15; contains landed #9 |
| [#17](https://github.com/Pantani/relayer/pull/17) (draft) | `Pantani/cx/complexity-cli-tx@412d681` | `cmd/tx.go`, characterization/state | direct base for provider liveliness; contains landed #9 |
| [#18](https://github.com/Pantani/relayer/pull/18) (draft) | `Pantani/cx/complexity-providers-liveliness@e36ad51` | Cosmos/Penumbra `provider.go`, characterization/state | direct base for message handlers; contains landed #9 |

## Campaign ownership

Exclusive worktree lease: `complexity-characterization-engineer` is the only `ACTIVE` editor for provider-message-handlers characterization. Handoff requires the previous holder to be `COMPLETED`, `IDLE`, or `INTERRUPTED` first.

| Subwave | Owner | Writable files | Worktree | Status |
|---|---|---|---|---|
| campaign integration | complexity-orchestrator | `.claude/**`, `CLAUDE.md`, `_workspace/complexity/ledger.md`, `_workspace/complexity/ownership.md`, `_workspace/complexity/inventory.md`, `_workspace/complexity/plan.md` and Git integration | `/Users/pantani/.codex/worktrees/complexity-providers-message-handlers/relayer` | idle during exclusive message-handlers characterization lease; branch based on #18 head |
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
| cli-chains characterization | complexity-characterization-engineer | characterization tests for `cmd/chains.go`; `_workspace/complexity/characterization/cli-chains.md` | `/Users/pantani/.codex/worktrees/complexity-cli-chains/relayer` | completed; 20 scenarios, 21 focused/race checks, package 159 pass |
| cli-chains production | complexity-engineer | `cmd/chains.go` only | same worktree, after characterization approval | completed; targets `1/0, 1/0, 1/0, 1/0, 5/3`; helpers max `8/8` |
| cli-chains review | complexity-verifier | `_workspace/complexity/reviews/cli-chains.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| cli-paths characterization | complexity-characterization-engineer | `cmd/paths_characterization_test.go`; `_workspace/complexity/characterization/cli-paths.md` | `/Users/pantani/.codex/worktrees/complexity-cli-paths/relayer` | completed; introduced test `5/13` reduced to `1/1`, helpers max `3/5`, assertions/subtests preserved |
| cli-paths production | complexity-engineer | `cmd/paths.go` only | same worktree, after characterization approval | completed; five targets `1/0`, helpers max `10/9`; production read-only |
| cli-paths review | complexity-verifier | `_workspace/complexity/reviews/cli-paths.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| cli-feegrant characterization | complexity-characterization-engineer | characterization tests for `cmd/feegrant.go`; `_workspace/complexity/characterization/cli-feegrant.md` | `/Users/pantani/.codex/worktrees/complexity-cli-feegrant/relayer` | completed; 9 scenarios, 11 focused/race checks, package 195 pass, tests max `7/7` |
| cli-feegrant production | complexity-engineer | `cmd/feegrant.go` only | same worktree, after characterization approval | completed; targets `1/0, 1/0`, helpers max `8/7` |
| cli-feegrant review | complexity-verifier | `_workspace/complexity/reviews/cli-feegrant.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| cli-query characterization | complexity-characterization-engineer | characterization tests for `cmd/query.go`; `_workspace/complexity/characterization/cli-query.md` | `/Users/pantani/.codex/worktrees/complexity-cli-query/relayer` | completed; 21 focused/race checks, package 216 pass, tests max `5/5` |
| cli-query production | complexity-engineer | `cmd/query.go` only | same worktree, after characterization approval | completed; 12 targets eliminated; helpers max `9/8`, file max `9/10` |
| cli-query review | complexity-verifier | `_workspace/complexity/reviews/cli-query.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| cli-tx characterization | complexity-characterization-engineer | characterization tests for `cmd/tx.go`; `_workspace/complexity/characterization/cli-tx.md` | `/Users/pantani/.codex/worktrees/complexity-cli-tx/relayer` | completed; 20 focused/race checks, package 236 pass, tests max `3/3` |
| cli-tx production | complexity-engineer | `cmd/tx.go` only | same worktree, after characterization approval | completed; five targets eliminated; helpers max `8/7`, file max `8/10` |
| cli-tx review | complexity-verifier | `_workspace/complexity/reviews/cli-tx.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| provider-liveliness characterization | complexity-characterization-engineer | tests for Cosmos/Penumbra `provider.go`; `_workspace/complexity/characterization/provider-liveliness.md` | `/Users/pantani/.codex/worktrees/complexity-providers-liveliness/relayer` | completed; 16 focused/race checks, packages 55 pass, tests max `4/4` |
| provider-liveliness production | complexity-engineer | `relayer/chains/cosmos/provider.go`, `relayer/chains/penumbra/provider.go` only | same worktree, after characterization approval | completed; targets `5/6, 7/9`, helpers `5/7`, files max `8/9` |
| provider-liveliness review | complexity-verifier | `_workspace/complexity/reviews/provider-liveliness.md` only | same worktree under exclusive sequential lease | completed; APPROVED |
| provider-message-handlers characterization | complexity-characterization-engineer | tests for Cosmos/Penumbra `message_handlers.go`; `_workspace/complexity/characterization/provider-message-handlers.md` | `/Users/pantani/.codex/worktrees/complexity-providers-message-handlers/relayer` | active; production files read-only |
| provider-message-handlers production | complexity-engineer | Cosmos/Penumbra `message_handlers.go` only | same worktree, after characterization approval | pending |
| provider-message-handlers review | complexity-verifier | `_workspace/complexity/reviews/provider-message-handlers.md` only | same worktree under exclusive sequential lease | pending |

Two agents must never edit the same file or worktree concurrently. Exactly one editor row may be `active`. Git integration belongs only to `complexity-orchestrator`.
