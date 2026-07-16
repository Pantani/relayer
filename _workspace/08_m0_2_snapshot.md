# M0.2 — snapshot de entrada

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
HEAD/base imutavel: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Modo de execucao

Reexecucao parcial do harness. Os artefatos `_workspace/00` a `_workspace/07` e todo o working tree do M0.1 foram preservados. O M0.2 usa fan-out em tres produtores independentes e fan-in com QA incremental:

1. FeeGrant: `interchaintest/feegrant_test.go`;
2. CLI/config: `cmd/tx.go`, `cmd/config.go` e testes diretamente relacionados;
3. Ethermint: `relayer/codecs/ethermint/**`.

Nenhum produtor deve fazer commit, push, trocar de branch ou modificar a fatia de outro produtor.

## Toolchain

- Go declarado e usado pelos checks: `1.25.9` via `GOTOOLCHAIN=go1.25.9`;
- Go do host no inicio: `1.26.5 darwin/arm64`;
- golangci-lint pinado: `2.12.2`;
- complexidade: gate estrito, falha quando ciclomatica ou cognitiva for `>=10`.

## Baseline relevante

| Superficie | Funcao | Ciclomatica | Cognitiva |
|---|---|---:|---:|
| FeeGrant | `TestRelayerFeeGrant` | 40 | 169 |
| FeeGrant | `TestRelayerFeeGrantExternal` | 39 | 166 |
| CLI | `linkCmd` | 26 | 50 |
| CLI | `createClientCmd` | 23 | 45 |
| CLI | `createConnectionCmd` | 21 | 40 |
| Config | `(*Config).ValidatePathEnd` | 12 | 15 |
| Ethermint | `traverseFields` | 33 | 82 |
| Ethermint | `typToEth` | 31 | 11 |
| Ethermint | `decodeProtobufSignDoc` | 16 | pendente de rebaseline focado |

Baseline global herdado no final do M0.1: 98 violacoes ciclomaticas, 152 cognitivas, uniao de 158 funcoes, maximos 48/169. Este lote deve reduzir a divida sem mascarar violacoes fora do escopo.

Nota de rebaseline: o relatorio inicial registrava cognitiva 10 para `ValidatePathEnd`; a execucao direta pinada antes da edicao retornou 15, valor adotado pelos relatorios M0.2.
