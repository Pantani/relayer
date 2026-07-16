# Complexity campaign subwave plan

Updated: 2026-07-16
Inventory head: `b91cb7df25cfd5cceafbd31e267aba0400b4bd8a`
Proof: cyclomatic/cognitive/union `63/107/110`, max `48/99`; `110/110` live union members assigned once, `0` missing, `0` duplicated. Score 10 passes; only scores greater than 10 violate.

Every subwave is stacked, characterized before production edits, independently reviewed, and published without automatic merge. Shared ledger/state and Git integration remain orchestrator-only.

| Wave | Branch | Production scope | Violations | Dependency |
|---|---|---|---:|---|
| A01 | `Pantani/cx/complexity-cli-config` | `cmd/config.go`, `cmd/appstate.go` | 5 | #10 |
| A02 | `Pantani/cx/complexity-cli-start` | `cmd/start.go`, `cmd/flags.go` | 3 | A01 |
| A03 | `Pantani/cx/complexity-cli-chains` | `cmd/chains.go` | 5 | A02 |
| A04 | `Pantani/cx/complexity-cli-paths` | `cmd/paths.go` | 5 | A03 |
| A05 | `Pantani/cx/complexity-cli-feegrant` | `cmd/feegrant.go` | 2 | A04 |
| A06 | `Pantani/cx/complexity-cli-query` | `cmd/query.go` | 12 | A05 |
| A07 | `Pantani/cx/complexity-cli-tx` | `cmd/tx.go` | 5 | A06 |
| B01 | `Pantani/cx/complexity-providers-liveliness` | Cosmos/Penumbra `provider.go` | 2 | A07, revalidate #9 |
| B02 | `Pantani/cx/complexity-providers-message-handlers` | Cosmos/Penumbra `message_handlers.go` | 2 | B01 |
| B03 | `Pantani/cx/complexity-providers-log-fields` | Cosmos/Penumbra `log.go` | 2 | B02 |
| B04 | `Pantani/cx/complexity-providers-grpc-query` | Cosmos/Penumbra `grpc_query.go` | 2 | B03 |
| B05 | `Pantani/cx/complexity-cosmos-query` | Cosmos `query.go` | 4 | B04 |
| B06 | `Pantani/cx/complexity-penumbra-keys` | Penumbra `keys.go` | 1 | B01, B05 |
| B07 | `Pantani/cx/complexity-provider-chain-processors` | Cosmos/Penumbra chain processors | 4 | B02, B05 |
| B08 | `Pantani/cx/complexity-cosmos-feegrant` | Cosmos `feegrant.go` | 5 | B05 |
| B09 | `Pantani/cx/complexity-cosmos-tx` | Cosmos `tx.go` | 8 | B05, B08 |
| B10 | `Pantani/cx/complexity-penumbra-tx` | Penumbra `tx.go` | 4 | B06 |
| C01 | `Pantani/cx/complexity-relayer-client-query` | `relayer/client.go`, `relayer/query.go` | 3 | B09, B10 |
| C02 | `Pantani/cx/complexity-relayer-naive-strategy` | `relayer/naive-strategy.go` | 3 | C01 |
| C03 | `Pantani/cx/complexity-relayer-strategies-path` | `relayer/strategies.go`, `relayer/path.go` | 3 | C02 |
| C04 | `Pantani/cx/complexity-relayer-relaymsgs` | `relayer/relayMsgs.go` | 1 | C03 |
| D01 | `Pantani/cx/complexity-processor-path-end` | `path_end.go` | 1 | C04 |
| D02 | `Pantani/cx/complexity-processor-path-lifecycle` | `path_processor.go` | 1 | D01 |
| D03 | `Pantani/cx/complexity-processor-message` | `message_processor.go` | 4 | D02 |
| D04 | `Pantani/cx/complexity-processor-path-end-runtime` | `path_end_runtime.go` | 9 | D01-D03 |
| D05 | `Pantani/cx/complexity-processor-path-internal` | `path_processor_internal.go` | 9 | D04 |
| E01 | `Pantani/cx/complexity-cli-keys` | `cmd/keys.go` | 3 | D05 |
| E02 | `Pantani/cx/complexity-stride-genesis-test` | `interchaintest/stride/setup_test.go` | 1 | E01 |

Phase counts: CLI excluding keys `37`, Cosmos `23`, Penumbra `11`, relayer including its test `11`, processor `24`, closure `4`; total `110`. There are no live EVM/Ethermint/Injective violations in this inventory. `relayer/chains/penumbra/query.go` and `relayer/processor/types_internal.go` no longer contain live targets at maximum 10.

Hazards: preserve nondeterministic map/fan-out ordering where observable; serialize tests that mutate config, Cobra/Viper, listeners, app state, or Bech32 globals; preserve processor cache identity, deduplication, partial effects, signals, cancellation, backpressure, metrics, and ordering. PR #9 changes Cosmos codec construction, so Cosmos package characterization must be rerun against its landed head before B01-B09.
