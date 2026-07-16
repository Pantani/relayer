# Complexity campaign ledger

Updated: 2026-07-16
Terminal condition: cyclomatic/cognitive/union `0/0/0`, global max `<=10/<=10`, all gates green, all required PRs open, no automatic merge.

## Live baseline

| Base | Cyclomatic | Cognitive | Union | Max cycle/cognitive | `make complexity` |
|---|---:|---:|---:|---:|---|
| `origin/main@4c1029d1177f797f11bcd560f14a18305a2e7353` | 63 | 108 | 111 | 48/99 | FAIL (expected before campaign; recalculated with maximum 10) |

## PR ledger

| Subwave | Branch/base | PR | Files | Violating functions | Scores before/after | Global before/after | Characterization/tests | Gates | Review | Dependencies/blockers | State |
|---|---|---|---|---|---|---|---|---|---|---|---|
| cli-version | `Pantani/cx/complexity-cli-version` / `origin/main@4c1029d` | [#10](https://github.com/Pantani/relayer/pull/10) (draft) | `cmd/version.go`, `cmd/version_test.go` plus harness/state | `getVersionCmd` | `7/16` -> `5/9`; helper `3/3` | `63/108/111` -> `63/107/110`; max `48/99` unchanged | 7 tests / 14 cases pass original and refactor with race; package `cmd` 83 pass | local scores, focal race, package, lint, readonly build, diff-check, inventory PASS; global `make complexity` remains expected FAIL with 110 | APPROVED under both old maximum 9 and current maximum 10; CodeRabbit major rejected with contract evidence | no live PR collision | PUBLISHED_DRAFT |

## Progress table

| PR | Files | Scores before/after | Global before/after | Gates | State |
|---|---|---|---|---|---|
| [#10](https://github.com/Pantani/relayer/pull/10) | `cmd/version.go`, `cmd/version_test.go` plus harness/state | `getVersionCmd 7/16` -> `5/9`; helper `3/3` | `63/108/111` -> `63/107/110` | PASS under maximum 10 (intermediate global complexity expected red) | PUBLISHED_DRAFT |
