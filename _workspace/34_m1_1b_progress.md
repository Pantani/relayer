# M1.1b — progresso da migração SDK/Comet/IBC

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
Base: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Resultado

O módulo raiz foi migrado para a família oficial 2026.1 e está compilável:

```text
github.com/cosmos/ibc-go/v11                 v11.2.0
github.com/cosmos/cosmos-sdk                 v0.54.3
github.com/cometbft/cometbft                 v0.39.3
github.com/cosmos/cosmos-sdk/store/v2        v2.0.0
cosmossdk.io/log/v2                          v2.1.0
github.com/cosmos/gogoproto                  v1.7.2
```

Não restam referências manuscritas ou edges diretos do módulo raiz para
ibc-go v8, capability, `cosmossdk.io/store` ou os módulos vanity antigos de
feegrant/tx/upgrade.

## Compatibilidade implementada

- Imports e APIs Classic migrados de ibc-go v8 para v11.
- Adapter explícito de latest height para Tendermint, solo machine,
  attestations e localhost; tipo desconhecido retorna erro.
- Localhost v11 tratado como stateless, inclusive timeout de transfer sem
  consulta a um client state removido.
- Denom RPC/model migrado para `Denom`, `Denom(s)` e `DenomHash` em Cosmos e
  Penumbra.
- Misbehaviour passa por `MsgUpdateClient`, como requerido por v11.
- Atributos de evento Classic removidos de v11 foram preservados como
  constantes locais do relayer.
- ICS-29 removido upstream foi preservado por um codec local, sem sidecar v8,
  com TypeURLs/bytes protobuf goldens e o sentinel ABCI legado
  `feeibc:7`.
- sr25519 removido de CometBFT foi preservado localmente com vetores de
  compatibilidade 0.38, serialização Any/Amino, persistência/reabertura,
  export/import armor e comparação de chave privada em tempo constante.
- Codec Penumbra agora usa o AddressCodec do SDK 0.54 e seu próprio registry
  para decodificar Tx/fee payer; transações malformadas não causam panic.
- `QueryDenomHash` Penumbra foi implementado. O builder ICS-29 Penumbra, que
  não possui Action equivalente, agora retorna erro `unsupported` em vez de
  derrubar o processo.
- Verificação ECDSA Injective usa a rota portátil do go-ethereum, liberando
  cross-build Linux sem CGO.

## Evidência de QA

```text
go mod tidy                         PASS
go mod verify                       PASS
go list -deps ./...                 PASS
git diff --check                    PASS
make lint                           PASS (0 issues)
make build                          PASS
CGO_ENABLED=0 linux/amd64 build     PASS
CGO_ENABLED=0 linux/arm64 build     PASS
go test -p 1 ./...                  PASS (384 testes / 52 pacotes)
focused go test -race               PASS (108 testes / 7 pacotes)
focused go vet                      PASS
```

`go test ./...` com paralelismo entre pacotes expôs o flake temporal herdado
`TestMockChainAndPathProcessors`: o teste passou isolado, mas em duas execuções
paralelas encerrou com três ou quatro mensagens na fila para um limite rígido
de duas. A suíte inteira passa com `-p 1`; a asserção não foi afrouxada.

`go vet ./...` continua acusando somente os dois JSON tags em campos não
exportados do protobuf Injective gerado, já inventariado no baseline.

## Complexidade

Todas as funções novas ou com corpo alterado neste lote ficaram abaixo do
limite estrito de 10 nas duas métricas. Destaques:

```text
                                      cyclomatic  cognitive
SendTransferMsg                            6          5
knownMessageFeePayer                       9          1
derivedMessageFeePayer                     5          4
PenumbraProvider.LogSuccessTx              5          9
ICS29 ValidateBasic                        6          5
QueryDenomHash                             2          1
```

O gate global ainda está vermelho por dívida herdada, mas melhorou sem elevar
nenhuma função:

```text
cyclomatic violations: 83  (max 48)
cognitive violations:  137 (max 99)
union:                  141
```

Portanto este lote satisfaz o contrato incremental, mas ainda não satisfaz o
objetivo final do programa de zerar todas as funções com score >= 10.

## Bloqueio do submódulo de integração

O módulo `interchaintest` não pode ser considerado migrado. O teste read-only
isolado falha em `cosmossdk.io/x/upgrade@v0.2.0`: o framework oficial atual
usa Store v1 enquanto o relayer local seleciona SDK 0.54/Store v2. O mesmo
conflito existe nas linhas antigas do framework.

```text
GOWORK=off go test -mod=readonly -run '^$' ./...
FAIL: cosmossdk.io/store/types.CommitMultiStore não implementa
      github.com/cosmos/cosmos-sdk/store/v2/types.CommitMultiStore
```

Próximo sublote recomendado: M1.1b-d, isolando a orquestração Docker do app Go
do interchaintest ou mantendo um fork temporário SDK 0.54. Apenas retirar o
módulo do CI reduziria cobertura e não fecha o critério de aceitação.

## Revisão

A revisão focal paralela terminou sem finding acionável após as correções.
CodeRabbit CLI não aceitou o diff monolítico porque o modo OSS limita a revisão
a 150 arquivos e a árvore acumulada contém 174; a cobertura substituta foi
revisão especializada, race, vet, testes focais e complexidade por função.

