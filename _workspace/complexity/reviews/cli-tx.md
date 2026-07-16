# cli-tx independent review

- Reviewer: `complexity-verifier`
- Functional production base: `f154382` (`de8cb02` changes campaign state only)
- Production scope: `cmd/tx.go`
- Characterization scope: `cmd/tx_characterization_test.go`
- Review artifact ownership: `_workspace/complexity/reviews/cli-tx.md` only
- Threshold contract: score `10` passes; only scores greater than `10` violate; pinned flags are `-over 10`

## Assertions

| assertion | resultado | evidencia | severidade | acao |
|---|---|---|---|---|
| Ownership and review scope are valid | PASS | `_workspace/complexity/ownership.md` grants the exclusive cli-tx review lease to `complexity-verifier` and permits writing only this artifact. Before this artifact was created, `git status --short` listed only `cmd/tx.go`; production, tests, characterization, campaign state, harness, inventory, plan, ownership, ledger, and Git state remained read-only. | - | None. |
| Original characterization is valid and state handoff is production-neutral | PASS | `_workspace/complexity/characterization/cli-tx.md` records 20 focused tests, the same 20 under `-race`, and 236 package tests passing on the original implementation. `git diff --quiet f154382 de8cb02 -- cmd/tx.go cmd/tx_characterization_test.go` exited 0, proving the later handoff commit changed neither production nor characterization. | - | None. |
| Characterization remains green and order-sensitive checks are stable | PASS | The exact focused regex passed 20/20 normally and 20/20 under `-race`; 20 repeated executions passed 400/400. `go test ./cmd -count=1` passed 236/236. In-memory providers, a synchronized call ledger, Cobra, and the observer logger require no RPC, filesystem, or public network. | - | None. |
| Diff is structural and limited to the assigned production file | PASS | The complete `f154382..working-tree` production diff was reviewed hunk by hunk. It extracts unexported option, run, key-check, path/chain-ID, flush-runtime, transfer-preparation, channel, denom, timeout, and path-selection helpers plus two private structs. No exported API, provider API, retry policy, timeout value, cancellation, log, metric, broadcast, or persistence behavior was added or removed. | - | None. |
| Cobra metadata, argument ranges, aliases, and defaults are preserved | PASS | Create remains `channel path_name`/`chan`, exact one arg, timeout `10s`, retries 3, override false, transfer ports, unordered, `ics20-1`, empty memo. Close remains exact three args with timeout `10s`, retries 3, empty memo. Flush remains zero-to-two args/`relay-pkts`, default max message length, empty memo/stuck chain, zero stuck heights. Transfer remains exact five args with empty path/memo and zero height/time offsets. | - | None. |
| `setPathsFromArgs` selection, identity, reverse orientation, and failures are preserved | PASS | `PathsFromChains` still executes first and returns its zero-path error before mutation. Named selection still uses exact `Paths.Get`; unnamed multiple paths retain the exact ambiguity error; unnamed single selection retains the same `*Path` identity. Reverse paths install Dst on the source chain and Src on the destination chain. Source `SetPath` still precedes destination `SetPath`, preserving partial source mutation when the second end is invalid. IBC V2 remains accepted with no new validation. | - | None. |
| Channel-create flag getter, key, validation, and provider order are preserved | PASS | After `ChainsFromPath`, getters remain override, source port, destination port, order, version, timeout, then max retries. Source key lookup/check precedes destination; exact missing-key errors and repeated `Key()` call on failure remain. `CreateOpenChannels` receives the same context, destination, retries, timeout, ports, order, version, override, late memo, and path name. Invalid order is still detected only after both key checks and path mutation; an existing source channel still short-circuits after `QueryLatestHeight` and `QueryConnectionChannels(41, connection-0)`. | - | None. |
| Channel-close getter, key, height, and channel query order are preserved | PASS | `ChainsFromPath` remains before timeout and max-retry getters; channel/port args are then selected; source key check precedes destination. Destination-key failure occurs before height. Source `QueryLatestHeight` precedes `QueryChannel(height, channelID, portID)`, whose error is propagated unchanged. `CloseChannel` receives the same context, destination, retries, timeout, channel, port, memo, and path name. | - | None. |
| Flush path lookup, deduplication, preflight, and side effects are preserved | PASS | A named path still uses `MustGet` and preserves the documented missing-path panic. All paths retain their native map iteration. Chain IDs are first-seen deduplicated, with each path contributing source then destination; repeated focused runs remain stable. `Chains.Gets` precedes `ensureKeysExist`, then max-message flag access. A two-arg flush installs the allowlist filter before stuck-packet parsing, so the mutation survives the exact stuck-packet validation error. | - | None. |
| Flush runtime call, timeout, cancellation, and logging are preserved | PASS | Only after all preflight does the code create the same ten-minute derived context. `StartRelayer` receives the same logger, chains, named paths, max length, global receiver/memo limits, command memo, zero tuning values, `FlushLifecycle`, `ProcessorEvents`, nil callbacks, and stuck packet. It still blocks exclusively on the returned error channel, suppresses `context.Canceled`, and logs/returns other errors as `Relayer start error`. The characterization gap correctly leaves event-processor internals to existing processor tests. | - | None. |
| Deprecated compatibility commands preserve warning and flush order | PASS | Packet and acknowledgement compatibility commands still emit the exact deprecation warning before delegating to flush; a subsequent missing-chain error remains unchanged. Neither helper changes arguments, allowlist behavior, or runtime invocation. | - | None. |
| Transfer lookup, path, amount, height, and channel order are preserved | PASS | Source lookup precedes destination; path flag and `setPathsFromArgs` precede amount parsing, preserving path mutations on an invalid coin. Source latest height then precedes path-side connection selection and `QueryConnectionChannels(height, connectionID)`. Channel scan remains in provider order. A missing channel returns the same formatted error before denom lookup or timeout flag access. | - | None. |
| Transfer denom, timeouts, raw receiver, memo, packet, and broadcast are preserved | PASS | Denom traces are queried with `(0, 100, srcHeight)` and every matching path still rewrites to the same IBC denom. Height-offset getter precedes time-offset getter and both follow channel/denom queries; negative time remains rejected downstream only afterward. `raw:` stripping is unchanged. Memo is resolved after raw handling and passed with the same precedence. `SendTransferMsg` receives the same chains, amount, receiver, offsets, source channel, logger, and context, preserving localhost time math, client-derived timeout height, exact `PacketInfo`, one MsgTransfer, one broadcast, and message count/order. | - | None. |
| CodeRabbit completed and its finding was independently adjudicated | PASS | CodeRabbit CLI `0.6.5` was authenticated; `coderabbit review --agent -t uncommitted --dir <worktree>` exited 0 after reviewing `cmd/tx.go` and returned one major finding asking to replace flush `Paths.MustGet` with an error-returning lookup. The base deliberately uses `MustGet`, and `TestFlushCharacterizesMissingPathPanicAndMissingChainError` freezes its exact panic. Applying the suggestion would change observable behavior and is not a regression fix for this structural subwave. | - | Track separately if product behavior should change; make no production change here. |
| Secret review is clean | PASS | Broad added-line credential-keyword and known credential-signature scans over the complete production diff returned no matches. No secret, token, private-key header, or credential-like value is present in the reviewed change. | - | None. |
| Pinned production and test complexity gates pass | PASS | `gocyclo@v0.6.0 -over 10` and `gocognit@v1.2.1 -over 10 -test` over `cmd/tx.go` and `cmd/tx_characterization_test.go` exited 0 with no output. The characterization file maximum is `3/3`. | - | None. |
| All five assigned violations are eliminated | PASS | Cyclomatic/cognitive changes are: `xfersend 19/36 -> 1/0`; `createChannelCmd 11/20 -> 1/0`; `flushCmd 11/20 -> 1/0`; `setPathsFromArgs 15/14 -> 5/4`; and `closeChannelCmd 8/14 -> 1/0`. | - | None. |
| Every new helper stays within the maximum | PASS | Full pinned scores show new helpers at or below `8/7`; the highest cyclomatic helper is `readCreateChannelOptions 8/7`. The complete production file maximum is `8/10`, where cognitive 10 belongs to unchanged `upgradeClientsCmd` and passes the maximum-10 contract. | - | None. |
| Formatting, diff, package, lint, module, and build gates pass | PASS | `gofmt -d cmd/tx.go` produced no output. Unstaged, cached, and combined-from-`f154382` diff checks exited 0. Focused/package tests passed; `make lint` reported `0 issues.` and verified modules; independent `go mod verify` and `go build -mod=readonly ./...` passed. | - | None. |
| Vet result is clean in scope; global generated baseline remains explicit | PASS (SCOPE) / EXPECTED BASELINE RED | `go vet ./cmd` passed. `go vet ./...` exited 1 only at generated, untouched `relayer/codecs/injective/tx.pb.go:242-243` because `chainId` and `chainIdMul` are unexported fields with JSON tags. The file has the canonical protoc generated marker and `git diff --quiet f154382 -- relayer/codecs/injective/tx.pb.go` passed. No diagnostic references `cmd/tx.go` or the characterization test. | info | Fix or regenerate the protobuf separately; do not edit generated baseline in this subwave. |
| Global inventory reduces strictly with no new violation | PASS | Fresh isolated inventory is cyclomatic/cognitive/union `46/71/73`, maxima `48/99`, versus incoming `50/76/78`, maxima `48/99`. The exact `-4/-5/-5` reduction matches the four cyclomatic and five cognitive target violations. Neither `cmd/tx.go` nor its characterization test appears in the violation table. | - | None. |
| Repository complexity gate remains honestly enforced | PASS (EXPECTED RED) | `make complexity` invoked the unchanged maximum-10 gate and exited 2 only because 73 unrelated global union violations remain. Its output does not list `cmd/tx.go` or `cmd/tx_characterization_test.go`. | - | Continue subsequent subwaves until terminal `0/0/0`. |

## Commands and outcomes

```text
coderabbit --version
coderabbit auth status
  PASS; 0.6.5; authenticated
coderabbit review --agent -t uncommitted --dir <worktree>
  PASS; exit 0; reviewed cmd/tx.go; 1 behavior-changing finding rejected against base

go test ./cmd -run 'Test(TxCommandsCharacterize|SetPathsFromArgsCharacterizes|CreateChannelCharacterizes|CloseChannelCharacterizes|FlushCharacterizes|DeprecatedPacketAndAckCommandsCharacterize|TransferCharacterizes)' -count=1
  PASS; 20/20
go test -race ./cmd -run 'Test(TxCommandsCharacterize|SetPathsFromArgsCharacterizes|CreateChannelCharacterizes|CloseChannelCharacterizes|FlushCharacterizes|DeprecatedPacketAndAckCommandsCharacterize|TransferCharacterizes)' -count=1
  PASS; 20/20; race detector clean
go test ./cmd -run 'Test(TxCommandsCharacterize|SetPathsFromArgsCharacterizes|CreateChannelCharacterizes|CloseChannelCharacterizes|FlushCharacterizes|DeprecatedPacketAndAckCommandsCharacterize|TransferCharacterizes)' -count=20
  PASS; 400/400 repeated order-sensitive executions
go test ./cmd -count=1
  PASS; 236/236

go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 10 \
  cmd/tx.go cmd/tx_characterization_test.go
go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 10 -test \
  cmd/tx.go cmd/tx_characterization_test.go
  PASS; no production or test violations; production max 8/10, characterization max 3/3

gofmt -d cmd/tx.go
git diff --check
git diff --cached --check
git diff f154382 --check
  PASS; no output

go vet ./cmd
  PASS
go vet ./...
  EXPECTED BASELINE RED; generated injective tx.pb.go:242-243 only; reviewed scope clean
make lint
  PASS; 0 issues; all modules verified
go mod verify
  PASS; all modules verified
go build -mod=readonly ./...
  PASS

bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/cli-tx-inventory.md
  PASS; 46/71/73; max 48/99; touched production/test files absent
make complexity
  EXPECTED RED; exit 2; 73 unrelated global violations remain under maximum 10
```

## Verdict

The cli-tx refactor preserves characterized command metadata, path selection and mutation, key/flag/provider ordering, flush preflight/filter/runtime behavior, deprecation logs, transfer construction, timeouts, packet data, and broadcast behavior. It eliminates all five assigned violations, keeps the full file at most `8/10`, and passes characterization, race, package, CodeRabbit adjudication, secret, formatting, lint, module, readonly-build, scoped-vet, diff, and inventory gates without an actionable introduced defect. The only global-vet diagnostics are the documented generated protobuf baseline outside this diff.

APPROVED
