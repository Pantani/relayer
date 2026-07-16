# cli-query characterization

- Base SHA: `3d18d1593c9c0ed26198f65f201b11f828c69361`
- Ownership: exclusive characterization lease; production `cmd/query.go` remained read-only. Writable files were `cmd/query_characterization_test.go` and this document only.
- Threshold: scores of 10 are allowed; only scores greater than 10 fail.
- Functions/scores:
  - `queryBalancesCmd` `12/27`
  - `queryChannelsPaginated` `13/22`
  - `queryChannelsToChain` `10/21`
  - `queryClientsExpiration` `12/20`
  - `queryBalanceCmd` `10/16`
  - `queryHeaderCmd` `10/16`
  - `queryChannel` `8/15`
  - `queryClientCmd` `8/15`
  - `queryConnectionsUsingClient` `8/15`
  - `queryUnrelayedAcknowledgements` `7/12`
  - `queryUnrelayedPackets` `7/12`
  - `queryConnectionChannels` `6/11`

## Observable contracts

- Metadata: exact `Use`, aliases, output/height/pagination/key/IBC-denom flags for the targeted commands.
- Balance commands: default versus explicit key selection; `KeyExists` before `ShowAddress`; exact missing-key and missing-chain errors; provider error propagation; exact legacy output and JSON fields; `balances` preserves the original `args[0]` error lookup when a later chain is absent.
- Header: latest-height query only when no height is supplied; explicit base-10 height propagation; exact JSON output; legacy output prints the marshaled byte slice rather than the JSON text.
- Client, connection, and channel: exact `AddPath` mutations; explicit versus latest height; exact provider method arguments; exact `Sprint` output and stderr; collection commands continue after per-item `Sprint` failures.
- Paginated channels: unique connection lookup; connection/client enrichment; channel output stays in provider input order; concurrent lookup call sets are asserted without ordering; the observed maximum is exactly ten concurrent connection queries for a twelve-connection input.
- Destination-filtered channels: only client states whose `ChainID` matches the destination are expanded; malformed client states are ignored; initial `QueryClients` errors are returned; destination lookup errors preserve their exact text.
- Unrelayed packets and acknowledgements: both path ends are installed before queries; exact chain heights, channel IDs, port IDs, and sequence arguments; concurrent call sets are compared after sorting; provider sequence order is preserved (there is no CLI sorting); empty results marshal as `[]`, not `null`.
- Expiration: source output precedes destination output; expiration is counterparty block time plus trusting period; RFC822 timestamps, client/chain labels, update height, trusting period, unbonding period, and `GOOD`/`EXPIRED` status are frozen. The variable remaining duration is checked structurally because it depends on `time.Now()`.
- Error ordering: a non-`light client not found` source expiration error returns before the destination expiration query.

## Pre-existing panics frozen

- `queryChannelsPaginated` panics during its print pass when a returned channel has an empty `ConnectionHops` slice. The discovery pass skips it, but the print pass indexes element zero unconditionally.
- `queryClientsExpiration` panics after a `light client not found` error. It intentionally accepts that error, then formats the zero `ClientStateInfo` before checking `errSrc == nil`, dereferencing a nil `LatestHeight`.

## Tests added before refactor

- `cmd/query_characterization_test.go`: 21 tests covering metadata, balances/header, client/connection/channel, pagination/concurrency, unrelayed queries, expiration math/status, exact outputs/errors, empty values, filtering, order, and the two pre-existing panics.
- Providers are in-memory interface fakes; no RPC, filesystem, or network access is used.

## Original implementation evidence

- `rtk go test ./cmd -run '^TestQuery' -count=1` -> 21 passed.
- `rtk go test -race ./cmd -run '^TestQuery' -count=1` -> passed with the race detector.
- `rtk go test ./cmd -count=1` -> 216 passed.
- `rtk go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 10 cmd/query_characterization_test.go` -> no violations.
- `rtk go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 10 -test cmd/query_characterization_test.go` -> no violations.
- `rtk git diff --check` -> clean.

## Gaps/blockers

- `gocyclo@v0.6.0` does not define a `-test` flag (`flag provided but not defined: -test`); unlike `gocognit`, it scans the explicitly supplied test file without that flag.
- Cosmos-provider-only `QueryChannelsPaginated` pagination-next-key behavior is not instantiated because it requires a concrete `*cosmos.CosmosProvider`; non-Cosmos pagination, filtering, enrichment, ordering, concurrency, errors, and panic behavior are covered through the provider interface.
- Marshal-error branches from `json.Marshal` are unreachable with the concrete maps, byte slices, and `RelaySequences` used by these commands.
- No blocker for structural refactoring. Preserve the two documented panic contracts unless the campaign explicitly approves a behavior fix with separate tests.
