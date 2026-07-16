# M0.2 — resultado consolidado

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
HEAD/base: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Resultado

O lote M0.2 foi executado em fan-out nas fatias FeeGrant, CLI/config e Ethermint, seguido de QA independente. Todas as funcoes criadas ou tocadas ficaram abaixo de 10 nas duas metricas, sem baseline permissivo, `nolint` ou mudanca do limite.

| Fatia | Antes | Depois | Aceitacao principal |
|---|---|---|---|
| FeeGrant | testes `40/169` e `39/166` | ambos `1/0`; arquivo maximo `8/7` | quatro cenarios Docker `-race` PASS em 407.890s |
| CLI/config | builders ate `26/50`; validator `12/15` | builders `1/0`; validator `7/6`; helpers maximo `7/6` | contratos Cobra, erros, retry e ordem de efeitos preservados |
| Ethermint | traversal `33/82`, tipos `31/13`, SignDoc ate `16/15` | escopo manuscrito inteiro maximo `8/7` | goldens EIP-712/BIP-44, unit/race/build PASS |

## Rebaseline global

| Metrica | M0.1 | M0.2 | Delta |
|---|---:|---:|---:|
| Ciclomatica `>=10` | 98 | 86 | -12 |
| Cognitiva `>=10` | 152 | 139 | -13 |
| Uniao | 158 | 145 | -13 |
| Maximo ciclomatico | 48 | 48 | 0 |
| Maximo cognitivo | 169 | 99 | -70 |

`make complexity` continua falhando corretamente. O M0.2 reduz a divida, mas nao conclui o objetivo global.

## Verificacao final do lider

```text
GOTOOLCHAIN=go1.25.9 go test -mod=readonly -count=1 ./...       PASS
GOTOOLCHAIN=go1.25.9 go build -mod=readonly ./...               PASS
GOTOOLCHAIN=go1.25.9 go test -count=1 ./cmd                     PASS
Ethermint go test -race -count=1                               PASS
interchaintest go test -count=1 -run '^$' ./...                 PASS
interchaintest go build ./...                                   PASS
make lint                                                       PASS, 0 issues
metricas focadas das funcoes tocadas                            PASS, <10/<10
git diff --check                                                PASS
make complexity global                                          FAIL esperado, 145 funcoes
```

O warning macOS `LC_DYSYMTAB` apareceu no teste `-race`, sem falha. A revisao CodeRabbit ficou limitada pela quota OSS; self-review e QA independente foram executados.

## Limites e proximo lote

- M0.2 preserva IBC Classic e nao anuncia suporte IBC v2.
- Os codecs globais Ethermint ainda nao possuem initializer; o lifecycle deve ser resolvido em fatia funcional separada.
- M0.3 deve introduzir o modelo protocol-neutral e contratos de coexistencia Classic/v2 antes de decompor os hotspots centrais do processor.
- O working tree permanece sem commit ou push.
