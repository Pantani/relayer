# M1.1b — dependency matrix and migration proof

Consulted: 2026-07-15 19:22 -03  
Repository SHA: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Snapshot at audit start: root `go.mod` selected SDK `v0.50.11`, CometBFT `v0.38.12`, ibc-go `v8.2.0`, store `v1.1.1`, gogoproto `v1.7.0`; the working tree already contained M0/M1.1a changes.  
Scope: read-only audit of production sources. All dependency and source experiments were performed in `/tmp/relayer-m1-1b-audit.pZFBVC`. Other agents began editing the shared migration files while this audit was running; none of those production edits are claimed here.

## Decision

The target is a coherent official release family, not three independent version bumps:

| component | current snapshot | M1.1b target | official constraint |
|---|---:|---:|---|
| Go | `1.25.9` | `1.25.9` | ibc-go `v11.2.0` declares `go 1.25.9` |
| Cosmos SDK | `v0.50.11` | `v0.54.3` | selected patch in the `0.54.x` family |
| CometBFT | `v0.38.12` | `v0.39.3` | SDK `v0.54.3` requires it directly |
| ibc-go | `v8.2.0` | `v11.2.0` | latest verified v11 tag on consultation date |
| SDK store | `cosmossdk.io/store v1.1.1` | `github.com/cosmos/cosmos-sdk/store/v2 v2.0.0` | required by SDK and ibc-go |
| SDK log | `cosmossdk.io/log v1.4.1` | `cosmossdk.io/log/v2 v2.1.0` | required by SDK and ibc-go |
| gogoproto | `v1.7.0` | `v1.7.2` | required by all three target modules |

The Cosmos release-family table explicitly groups SDK `0.54.x`, CometBFT `0.39.x`, and IBC Go `v11.x.y`: <https://docs.cosmos.network/sdk/next/release-family>. Exact primary sources:

- ibc-go tag and module graph: <https://github.com/cosmos/ibc-go/releases/tag/v11.2.0>, <https://github.com/cosmos/ibc-go/blob/v11.2.0/go.mod>
- SDK tag, module graph, and migration guide: <https://github.com/cosmos/cosmos-sdk/releases/tag/v0.54.3>, <https://github.com/cosmos/cosmos-sdk/blob/v0.54.3/go.mod>, <https://docs.cosmos.network/sdk/latest/upgrade/upgrade>
- CometBFT tag and module graph: <https://github.com/cometbft/cometbft/releases/tag/v0.39.3>, <https://github.com/cometbft/cometbft/blob/v0.39.3/go.mod>
- IBC migrations which must be traversed conceptually: <https://docs.cosmos.network/ibc/latest/migrations/v8-to-v8_1>, <https://docs.cosmos.network/ibc/latest/migrations/v8_1-to-v10>, <https://docs.cosmos.network/ibc/latest/migrations/v10-to-v11>

`go mod download -json` verified tag provenance:

```text
ibc-go/v11 v11.2.0      cfc072e53eee42b2ab804cd4344ba610016f793c
cosmos-sdk v0.54.3      046046a6731ddc00bca29193f5f0529d7017b3e3
cometbft v0.39.3        49b82838fcca442b2445f76605c101609ed04130
```

Context7 was unavailable in this session. Only official tagged source, official migration documentation, and Go MVS output were used.

## MVS evidence

The target graph is internally coherent at its core:

```text
github.com/cosmos/ibc-go/v11@v11.2.0 -> github.com/cosmos/cosmos-sdk@v0.54.0
github.com/cosmos/ibc-go/v11@v11.2.0 -> github.com/cometbft/cometbft@v0.39.0
github.com/cosmos/ibc-go/v11@v11.2.0 -> github.com/cosmos/cosmos-sdk/store/v2@v2.0.0
github.com/cosmos/ibc-go/v11@v11.2.0 -> cosmossdk.io/log/v2@v2.1.0
github.com/cosmos/cosmos-sdk@v0.54.3 -> github.com/cometbft/cometbft@v0.39.3
github.com/cosmos/cosmos-sdk@v0.54.3 -> github.com/cosmos/cosmos-sdk/store/v2@v2.0.0
github.com/cosmos/cosmos-sdk@v0.54.3 -> cosmossdk.io/log/v2@v2.1.0
```

MVS therefore selects the requested SDK and Comet patch versions. It also selects `github.com/cometbft/cometbft-db v1.0.4` because ibc-go requests that version, above the `v0.14.3` requested by SDK `v0.54.3`. The isolated root compiled and ran its tests with that selection, but database/RPC integration still requires explicit coverage.

The upgrade also necessarily moves several existing direct dependencies through MVS, including go-ethereum `v1.13.15 -> v1.16.8`, Cobra `v1.8.1 -> v1.10.2`, Viper `v1.19.0 -> v1.21.0`, gRPC `v1.67.1 -> v1.80.0`, Prometheus client `v1.20.1 -> v1.23.2`, `x/sync v0.10.0 -> v0.20.0`, and Testify `v1.9.0 -> v1.11.1`. These are part of the blast radius and must remain visible in review.

## Blocking compatibility matrix

| capability/API | source/target fact | current use | required migration | acceptance test | priority |
|---|---|---|---|---|---:|
| ibc-go imports | major-version imports must change; v8.1-to-v10 and v10-to-v11 guides are breaking | 71 Go files imported `/v8` at audit start | replace Classic imports with `/v11` atomically | root compile plus Classic regression suite | P0 |
| SDK vanity `x/` modules | SDK v0.54 removes separate `cosmossdk.io/x/feegrant`, `/x/tx`, `/x/upgrade` modules | nine files use old store/x paths | import `github.com/cosmos/cosmos-sdk/x/...`; drop old direct requirements | codec, feegrant, query, upgrade tests | P0 |
| store v2 | SDK v0.54 requires `github.com/cosmos/cosmos-sdk/store/v2` | Cosmos and Penumbra use `cosmossdk.io/store/rootmulti` | migrate both to `store/v2/rootmulti`; do not retain a direct store-v1 requirement | Cosmos/Penumbra tx proof tests | P0 |
| `x/crisis` | SDK v0.54 moves it to `contrib/x/crisis` | Cosmos and Penumbra codec module basics | replace import with `github.com/cosmos/cosmos-sdk/contrib/x/crisis`; preserve interface registration | codec construction/Any decode | P0 |
| capability module | ibc-go v10 removes capability and v11 has no capability module | codec basics and localhost interchaintest | remove registration and requirement; no side-by-side v8 module | codec and Classic relay compile | P0 |
| ICS-29 fee middleware | ibc-go v10 removes `modules/apps/29-fee`; v11.2.0 contains no package | construct `MsgRegisterCounterpartyPayee`; decode four fee message types | **do not import v8 beside v11**; add a local wire/protobuf compatibility package for legacy chain messages, or explicitly remove feature | golden TypeURL/protobuf bytes from v8; legacy fee-chain E2E | P0 decision |
| sr25519 | SDK v0.54 and Comet remove sr25519; four files import the removed Comet package | key derivation, keyring algorithms, consensus-client pubkey conversion | recommended: remove support with an explicit config/key error and migration note; never silently fall back to secp256k1. Alternative is a separately maintained local crypto implementation | old sr config rejects safely; existing keys never change address silently | P0 decision |
| misbehaviour tx | v11 removes `MsgSubmitMisbehaviour` | Cosmos and Penumbra construct it; log parser type-switches it | submit the misbehaviour as `NewMsgUpdateClient(clientID, misbehaviour, signer)` and classify its event separately | misbehaviour construction + E2E freeze | P0 |
| `ClientState.GetLatestHeight` | v10 removes most client-state behavioral methods from the state type | relayer, provider, Cosmos, Penumbra use the getter | add an error-returning adapter/type switch for supported clients; use `tmclient.ClientState.LatestHeight`; do not default to zero | TM, solo, localhost, unsupported client tests | P0 |
| localhost client | v10 makes 09-localhost stateless; there is no `localhost.ClientState` or `RegisterInterfaces` | codec registration and processor query/cast | remove registration/state query; derive self-height from current chain context/observed block and chain revision | localhost transfer and current-height tests | P0 |
| transfer denom | v10 replaces `DenomTrace` with `Denom{Base, Trace[]Hop}` and renames RPCs | provider interface, Cosmos/Penumbra queries, CLI/E2E use old types | `DenomTrace -> Denom`, `DenomTrace(s) -> Denom(s)`, request/response renames, `GetFullDenomPath() -> Path()` | single/multi-hop denom and pagination tests | P0 |
| legacy event constants | v10 removes deprecated raw `packet_data`, `packet_ack`, update `header`; `AttributeVersion` is renamed | parser intentionally accepts old-chain events | retain legacy wire keys as relayer-owned constants; use v11 constants for current hex keys and `AttributeKeyVersion` | v8 raw + v11 hex event fixtures | P0 |
| fee/capability coexistence | side-by-side v8/v11 experiment fails graph resolution | tempting compatibility shortcut | prohibit v8 as a module dependency; local compatibility types only | `go mod graph` contains no root v8/capability edge | P0 |
| Comet slim client sr pubkey | modern Comet has no sr public-key type | adapter converts slim sr pubkey to Comet sr pubkey | return a typed unsupported-key error instead of `nil`; ensure caller handles it | conversion test and RPC fixture | P1 |
| channel upgrade APIs | v10 removes channel upgradability | no operational upgrade builder found; parser uses only shared version key | no builder migration; document loss of upstream type/constants and keep Classic handshake behavior | handshake regression | P1 |

## Isolated experiment results

### Experiment A — mechanical module bump

After replacing `/v8` with `/v11`, store/x imports, and target requirements, `go mod tidy` failed immediately:

```text
module github.com/cometbft/cometbft@latest found, but does not contain package
github.com/cometbft/cometbft/crypto/sr25519

module github.com/cosmos/cosmos-sdk@latest found, but does not contain package
github.com/cosmos/cosmos-sdk/x/crisis

module github.com/cosmos/ibc-go/v11@latest found, but does not contain package
github.com/cosmos/ibc-go/v11/modules/apps/29-fee[/types]
```

### Experiment B — retaining only v8 fee types

Keeping ibc-go v8 beside v11 for ICS-29 did not isolate the old API. `go mod tidy` traversed the v8 package test graph and failed:

```text
github.com/cosmos/cosmos-sdk/x/group@v0.2.0-rc.1: module declares its path as
cosmossdk.io/x/group but was required as github.com/cosmos/cosmos-sdk/x/group
```

Conclusion: a local minimal ICS-29 compatibility package is the only route that preserves legacy fee-message construction without contaminating the target graph.

### Experiment C — root compile after API adapters

After removing capability, isolating removed features, and adapting the compile-time APIs listed above:

```text
go test -run '^$' ./...    PASS (50 packages compile)
go test ./...              356 passed, 1 failed
```

The only remaining failure was `TestKeyRestoreSr25519`: the temporary probe removed sr25519 and the existing request silently used secp256k1, producing a different address. This is evidence that production must reject the removed algorithm explicitly; a fallback is unsafe. The temporary probe also removed the sr25519 package itself, accounting for one fewer package than the 51-package baseline.

The probe used a temporary Tendermint-only latest-height adapter to expose later compile failures. Its successful tests are not evidence that non-Tendermint client semantics are correct. Production must return an error for unsupported client-state types and add fixtures.

## Interchaintest blocker

The current `github.com/strangelove-ventures/interchaintest/v8` line is a different release family: even its newest `v8.8.1` uses SDK `v0.50.9`, Comet `v0.38.11`, ibc-go `v8.4.0`, store v1, capability, and separate `x/` modules. The current official `github.com/cosmos/interchaintest/v10@v10.0.1` still uses SDK `v0.53.4`, Comet `v0.38.19`, ibc-go `v10.3.0`, and store v1: <https://github.com/cosmos/interchaintest/blob/v10.0.1/go.mod>.

An isolated attempt to align `interchaintest/go.mod` with the target failed `go mod tidy` through the old ibc-go v8 graph and then failed compile in `cosmossdk.io/x/upgrade v0.1.0` because it expects store-v1 keys while SDK v0.54 supplies store-v2 keys.

Therefore M1.1b must not leave the two go.mod files half-aligned. Choose one explicit sub-plan:

1. Preferred: split SDK/IBC white-box fixtures away from the Docker orchestration layer, port localhost/misbehaviour fixtures to ibc-go v11 test APIs, and keep Docker calls black-box until an interchaintest release supports the 2026.1 family.
2. If full in-process framework coverage is mandatory now: maintain a temporary fork of Cosmos interchaintest upgraded to SDK 0.54/Comet 0.39/ibc-go v11.
3. Temporary isolation only: remove the old module from `go.work` and CI while the replacement is built. This avoids graph poisoning but is not M1.1b acceptance because it removes integration coverage.

Upgrading only from the old Strangelove pseudo-version to Cosmos interchaintest `v10.0.1` is useful cleanup but does not solve the SDK 0.54/v11 graph.

## Exact implementation sequence

The following sequence minimizes broken intermediate states. Imports and requirements should land in one change set.

### 1. Add characterization tests before the bump

- v8 ICS-29 golden TypeURLs and protobuf bytes for the four decoded messages plus `MsgRegisterCounterpartyPayee`.
- sr25519 restore/config behavior and an explicit decision test.
- Tendermint and localhost latest-height behavior.
- v8 raw and v11 hex packet event fixtures.
- old/new denomination query response conversion.

### 2. Add compatibility/adaptation boundaries

- Add a relayer-owned `legacy/ics29` wire package; register its exact message TypeURLs directly with the codec. Do not add ibc-go v8.
- Add one error-returning client-state height adapter used by relayer, Cosmos, and Penumbra.
- Add relayer-owned legacy event-key constants for `packet_data`, `packet_ack`, and `header`.
- Decide sr25519. Recommended implementation is explicit removal/error plus a migration note; do not silently select secp256k1.

### 3. Migrate source imports atomically

```text
github.com/cosmos/ibc-go/v8                         -> github.com/cosmos/ibc-go/v11
cosmossdk.io/store                                  -> github.com/cosmos/cosmos-sdk/store/v2
cosmossdk.io/x/feegrant                             -> github.com/cosmos/cosmos-sdk/x/feegrant
cosmossdk.io/x/tx                                   -> github.com/cosmos/cosmos-sdk/x/tx
cosmossdk.io/x/upgrade                              -> github.com/cosmos/cosmos-sdk/x/upgrade
github.com/cosmos/cosmos-sdk/x/crisis               -> github.com/cosmos/cosmos-sdk/contrib/x/crisis
```

Remove capability and upstream fee AppModuleBasic registration. Remove `localhost.RegisterInterfaces`. Apply the API migrations in the matrix before running tidy.

### 4. Change the root module graph

Equivalent `go mod edit` plan:

```sh
go mod edit \
  -droprequire=cosmossdk.io/store \
  -droprequire=cosmossdk.io/x/feegrant \
  -droprequire=cosmossdk.io/x/tx \
  -droprequire=cosmossdk.io/x/upgrade \
  -droprequire=github.com/cosmos/ibc-go/modules/capability \
  -droprequire=github.com/cosmos/ibc-go/v8

go mod edit \
  -require=cosmossdk.io/api@v1.0.0 \
  -require=cosmossdk.io/errors@v1.1.0 \
  -require=cosmossdk.io/math@v1.5.3 \
  -require=github.com/cometbft/cometbft@v0.39.3 \
  -require=github.com/cosmos/cosmos-sdk@v0.54.3 \
  -require=github.com/cosmos/cosmos-sdk/store/v2@v2.0.0 \
  -require=github.com/cosmos/gogoproto@v1.7.2 \
  -require=github.com/cosmos/ibc-go/v11@v11.2.0
```

Then run `go mod tidy`, review every direct-version movement, `go mod verify`, and assert:

```sh
go list -m github.com/cosmos/ibc-go/v11 github.com/cosmos/cosmos-sdk github.com/cometbft/cometbft github.com/cosmos/cosmos-sdk/store/v2 cosmossdk.io/log/v2 github.com/cosmos/gogoproto
```

Expected selection:

```text
github.com/cosmos/ibc-go/v11 v11.2.0
github.com/cosmos/cosmos-sdk v0.54.3
github.com/cometbft/cometbft v0.39.3
github.com/cosmos/cosmos-sdk/store/v2 v2.0.0
cosmossdk.io/log/v2 v2.1.0
github.com/cosmos/gogoproto v1.7.2
```

### 5. Keep `interchaintest/go.mod` coherent

Do not copy the root edits blindly. First choose the framework sub-plan above. Only after the old v8/SDK0.50 framework edge is removed or forked should `interchaintest/go.mod` select the same SDK/Comet/IBC family and be tidied. Keep the known pre-existing standalone tidy drift (`glog` and `x/net`) separate from M1.1b findings.

### 6. Gates

Run in this order so failures identify their layer:

1. `go test -run '^$' ./...`
2. focused codec, key, event, denom, client, Cosmos, and Penumbra tests
3. `go test ./...`
4. focused race tests; treat the pre-existing parallel `./cmd` SDK Config-prefix race separately
5. interchaintest compile and then selected live Classic fixtures
6. build, lint, vet with the known generated Injective vet baseline separated, module verification, diff check, complexity
7. assert no root dependency on ibc-go v8, capability, or old direct SDK x/store modules

## Acceptance boundary

M1.1b is complete only when:

- root source and root graph compile on the official 2026.1 family;
- Classic packet, handshake, proof, misbehaviour, localhost, denom, fee-compat, signing, and broadcast behavior is covered;
- no v8 sidecar module is used to preserve ICS-29;
- sr25519 has an explicit supported/unsupported contract with no silent key substitution;
- the integration-test module is either coherently upgraded/forked or an explicit replacement coverage plan is active; and
- baseline failures are not misreported as migration regressions.

