# cli-feegrant independent review

- Reviewer: `complexity-verifier`
- Functional production base: `2b210d0` (`55dc938` changes campaign state only)
- Production scope: `cmd/feegrant.go`
- Characterization scope: `cmd/feegrant_characterization_test.go`
- Review artifact ownership: `_workspace/complexity/reviews/cli-feegrant.md` only
- Threshold contract: score `10` passes; only scores greater than `10` violate; pinned flags are `-over 10`

## Assertions

| assertion | resultado | evidencia | severidade | acao |
|---|---|---|---|---|
| Ownership and review scope are valid | PASS | `_workspace/complexity/ownership.md` grants the exclusive cli-feegrant review lease to `complexity-verifier` and permits writing only this artifact. Production, tests, characterization, campaign state, harness, inventory, plan, ownership, ledger, and Git state remained read-only. | - | None. |
| Original characterization is valid and unchanged by handoff state | PASS | `_workspace/complexity/characterization/cli-feegrant.md` records 11 focused, 11 race, and 195 package tests passing before refactor. `git diff --quiet 2b210d0 55dc938 -- cmd/feegrant.go cmd/feegrant_characterization_test.go` exited 0. | - | None. |
| Characterization remains green without RPC or public network | PASS | `go test ./cmd -run '^TestCliFeegrant' -count=1` and its `-race` variant passed 11/11. Tests use a concrete CosmosProvider, in-memory keyring, and deterministic consensus fake for feegrant/auth/GetTx queries, async broadcast, and height 91; no real RPC or public request occurs. | - | None. |
| Diff is structural and limited to the assigned production file | PASS | Before this review artifact was written, `git diff --name-only` listed only `cmd/feegrant.go`. The diff extracts unexported provider/granter/configuration/gas/verification/query helpers and private option structs; Cobra API, provider API, network flow, signing, retries, timeouts, cancellation, and concurrency are unchanged. | - | None. |
| Cobra structure, defaults, cardinality, and mutual exclusions are preserved | PASS | Configure retains `MinimumNArgs(1)` and therefore accepts extra positional arguments; Basic Query retains one-or-two. Number of Grantees remains 10; Delete/Overwrite flags false; Grantees nil/empty; Gas and Memo empty. Cobra still rejects combinations among Number of Grantees, Grantees, and Delete before RunE/broadcast. Pagination defaults remain registered on Basic Query. | - | None. |
| Chain and provider lookup order/errors are preserved | PASS | Both run helpers consume `args[0]`, check the chain map, then assert `*cosmos.CosmosProvider`. Missing-chain guidance and `only CosmosProvider can be feegranted` remain exact. No flag, keyring, query, or mutation precedes those checks. | - | None. |
| Granter selection and internal/external classification are preserved | PASS | `resolveFeegrantGranter` chooses explicit `args[1]`, existing configured granter, then provider default key in the original order. It calls `KeyFromKeyOrAddress`; any error marks external, then validates Bech32 with the same exact invalid-address error. Internal keys and external addresses retain their original downstream key/key-or-address values. | - | None. |
| Delete order, logging, lock, mutation, and save are preserved | PASS | Provider and granter resolution still occur before the Delete branch. Delete then logs `Deleting feegrant configuration`, reloads under the same config lock, clears FeeGrants on the reloaded provider, invokes `cobra.CheckErr` on lock/save failure, and returns before Memo, Gas, query, or broadcast. | - | None. |
| Overwrite partial captured-provider behavior is preserved | PASS | `overwriteFeegrantGranter` compares the resolved key with the configured value and retains the exact missing-flag error. With overwrite enabled, the lock reloads configuration but the callback mutates the previously captured `prov`; the locking save therefore retains the old disk granter while the captured provider has the new one. The late Memo getter still fails afterward with the exact missing-flag error, preserving memory/disk divergence and zero broadcasts. | - | None. |
| Grantee choice, ordering, configuration, lock, and save are preserved | PASS | Reconfiguration condition is equivalent to original nil FeeGrants, Overwrite Grantees, or non-empty Grantees. Nil Grantees uses generated internal keys and rejects an external granter with the same error; explicit internal/external paths call the same provider methods. Input order is unchanged. After provider configuration, the same lock reload copies captured FeeGrants into the current provider and persists before broadcast. | - | None. |
| Memo, Gas, context, and EnsureBasicGrants ordering are preserved | PASS | Memo remains a late non-ignored getter after granter/grantee persistence. Gas is read afterward with getter and parse errors intentionally ignored; empty/invalid paths retain original zero/simulation semantics, while numeric `456789` reaches AuthInfo unchanged. `ctx := cmd.Context()` is evaluated once immediately before `EnsureBasicGrants`, and its return value is still discarded except for the error. | - | None. |
| Signed transaction messages and ordering are preserved | PASS | The real decoded TxRaw contains Memo `characterized memo`, Gas Limit `456789`, and exactly two `/cosmos.feegrant.v1beta1.MsgGrantAllowance` messages. Both embed BasicAllowance, use the default-key granter address, and target `grantee1` then `grantee2`, exactly matching input order. | - | None. |
| Broadcast, wait, height, persistence, and success logs are preserved | PASS | EnsureBasicGrants still queries existing grants, logs two `Creating feegrant` entries, signs once, calls async broadcast once, waits through `/cosmos.tx.v1beta1.Service/GetTx`, then logs `Feegrant succeeded`. Only afterward does `QueryLatestHeight` return 91; a fresh locking operation copies FeeGrants, records height 91, and logs `feegrant configured`. Context reuse and fresh lock-context evaluation match the base. | - | None. |
| Broadcast error wrapping and partial persistence are preserved | PASS | Broadcast failure remains exactly `error writing grants on chain: 'broadcast failed'` without `%w`. Managed Grantees were already persisted, BlockHeightVerified remains zero in memory/disk, latest height is not recorded, and neither success nor configured log is emitted; only the two creation logs remain. | - | None. |
| Basic Query's current argument and pagination behavior is preserved | PASS | `runFeegrantBasicGrants` still indexes `args[0]` as chain, then uses that same `args[0]` as key/address because the `len(args)==0` branch is unreachable. Therefore optional `args[1]` remains ignored. It resolves the chain-named key address, calls `QueryFeegrantsByGranter(address, nil)`, does not pass pagination flags, and logs returned grants in response order with Granter, Grantee, and rendered Allowance. | - | None. |
| CodeRabbit completed and all findings were independently adjudicated | PASS | CodeRabbit CLI `0.6.5` reviewed `cmd/feegrant.go` and exited 0 with five findings. Two duplicate major findings proposed assigning an external address into `granter.key`; the base also preserves the direct `KeyFromKeyOrAddress` result and uses it during overwrite. One critical proposed fixing Basic Query to use default/`args[1]`, explicitly contradicting the frozen base behavior. Two minor findings proposed propagating Gas errors and moving Delete before granter resolution, also changing base order/errors. None is introduced by this extraction; applying any would violate the behavior-preservation scope. | - | Track separately if product behavior should change; make no production change in this complexity subwave. |
| Secret review is clean | PASS | Broad keyword and known credential-signature scans over the full production diff returned no matches. The concrete test keys are generated in-memory and no secret material is present in the reviewed diff. | - | None. |
| Pinned production and test complexity gates pass | PASS | `gocyclo@v0.6.0 -over 10` and `gocognit@v1.2.1 -over 10 -test` over `cmd/feegrant.go` and `cmd/feegrant_characterization_test.go` exited 0 with no output. Test maximum remains `7/7`. | - | None. |
| Assigned violations are eliminated | PASS | `feegrantConfigureBasicCmd 26/45 -> 1/0` and `feegrantBasicGrantsCmd 7/13 -> 1/0` (cyclomatic/cognitive). | - | None. |
| Every new helper stays within the maximum | PASS | New helper scores are `runFeegrantConfigureBasic 8/7`, `feegrantCosmosProvider 3/2`, `resolveFeegrantGranter 6/5`, `deleteFeegrantConfiguration 1/0`, `overwriteFeegrantGranter 4/3`, `configureFeegrantGrantees 5/3`, `setFeegrantGrantees 4/4`, `feegrantGas 2/1`, `verifyFeegrantConfiguration 2/1`, and `runFeegrantBasicGrants 6/6`. Maximum is `8/7`. | - | None. |
| Package, formatting, lint, module, and build gates pass | PASS | `go test ./cmd -count=1` passed 195/195. `gofmt -d` produced no output. Unstaged, cached, and combined-from-`2b210d0` diff checks exited 0. `make lint` reported `0 issues.` and verified modules; independent `go mod verify` and `go build -mod=readonly ./...` passed. | - | None. |
| Global inventory reduces strictly with no new violation | PASS | Fresh inventory is cyclomatic/cognitive/union `53/88/90`, maxima `48/99`, versus incoming `54/90/92`, maxima `48/99`. The exact `-1/-2/-2` reduction matches the two target functions and metric membership. Neither production nor characterization test appears in the violation table; every helper passes local scans. | - | None. |
| Repository complexity gate remains honestly enforced | PASS (EXPECTED RED) | `make complexity` invoked the unchanged maximum-10 gate and exited 2 only because 90 unrelated global violations remain. It did not list `cmd/feegrant.go` or its characterization test. | - | Continue subsequent subwaves until terminal `0/0/0`. |

## Commands and outcomes

```text
coderabbit --version
coderabbit auth status
  PASS; 0.6.5; authenticated
coderabbit review --agent -t uncommitted --dir <worktree>
  PASS; exit 0; reviewed cmd/feegrant.go; 5 behavior-changing findings rejected against base

go test ./cmd -run '^TestCliFeegrant' -count=1
  PASS; 11/11
go test -race ./cmd -run '^TestCliFeegrant' -count=1
  PASS; 11/11; race detector clean
go test ./cmd -count=1
  PASS; 195/195

go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 10 \
  cmd/feegrant.go cmd/feegrant_characterization_test.go
go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 10 -test \
  cmd/feegrant.go cmd/feegrant_characterization_test.go
  PASS; no production or test violations

gofmt -d cmd/feegrant.go cmd/feegrant_characterization_test.go
git diff --check
git diff --cached --check
git diff 2b210d0 --check
  PASS; no output

make lint
  PASS; 0 issues; all modules verified
go mod verify
  PASS; all modules verified
go build -mod=readonly ./...
  PASS

bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/cli-feegrant-inventory.md
  PASS; 53/88/90; max 48/99; touched production/test files absent
make complexity
  EXPECTED RED; exit 2; 90 unrelated global violations remain under maximum 10
```

## Verdict

The cli-feegrant refactor preserves characterized CLI, granter, locking, partial-state, signed-transaction, broadcast/wait, height, persistence, error, and query behavior; eliminates both assigned violations; keeps every helper at most `8/7`; and passes characterization, race, package, secret, formatting, lint, module, readonly-build, diff, and inventory gates. CodeRabbit's five findings all request unrelated changes to pre-existing frozen behavior and do not identify a regression in this structural extraction.

APPROVED
