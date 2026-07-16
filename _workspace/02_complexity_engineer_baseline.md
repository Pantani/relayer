# Baseline de complexidade do Go relayer

Data da auditoria: 2026-07-15  
SHA congelado: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Contrato: toda funcao Go manuscrita, inclusive testes, deve ter complexidade ciclomatica **e** cognitiva estritamente menor que 10 (maximo permitido: 9).  
Estado: **REPROVADO** — 158 funcoes violam ao menos uma metrica; nao e seguro nem realista tratar isso como uma refatoracao monolitica.

## Resumo executivo

| Medida | Resultado |
|---|---:|
| Arquivos `.go` rastreados no SHA | 173 |
| Arquivos gerados excluidos | 38 |
| Arquivos manuscritos auditados | 135 |
| Arquivos manuscritos de producao | 95 |
| Arquivos manuscritos de teste/test-support | 40 |
| Funcoes auditadas | 1327 |
| Funcoes com ciclomatica >= 10 | 98 |
| Funcoes com cognitiva >= 10 | 152 |
| Funcoes que violam ambas | 92 |
| Funcoes que violam uma ou ambas (uniao) | 158 |
| Violacoes em producao | 151 |
| Violacoes em teste/test-support | 7 |
| Arquivos manuscritos com pelo menos uma violacao | 55 |
| Maximo ciclomatico | 48 |
| Maximo cognitivo | 169 |

O volume (158 funcoes em 55 arquivos), a concentracao no estado central do processador e os picos de 48/169 inviabilizam uma unica PR de refatoracao. A unidade de entrega deve ser um lote pequeno, com gate local das duas metricas, testes focados e preservacao explicita da ordem de efeitos, erros, logs e retry/ack semantics.

## Escopo e reprodutibilidade

- O worktree estava detached exatamente em `bef2e868f157659b403fe1303ee121fb69fec9e6`; `.claude/`, `CLAUDE.md` e `_workspace/` eram artefatos nao rastreados do harness e nao contem Go auditado.
- Enumeracao: `git ls-tree -r --name-only bef2e868f157659b403fe1303ee121fb69fec9e6` filtrado por `*.go`. Isso evita incluir arquivos nao rastreados ou fora do snapshot.
- Exclusao unica e deterministica: arquivo cujo primeiro bloco de 40 linhas contem `^// Code generated .* DO NOT EDIT\.$`. Extensao/nome `.pb.go` sozinho nao decide a exclusao.
- Categoria `test`: sufixo `_test.go`, arvore `interchaintest/` ou `internal/relayertest/`; o restante e `production`.
- Chave estavel de acompanhamento: `arquivo:funcao`; linha e apenas localizacao. Colisoes de chave estavel neste snapshot: 0.
- `gocognit` omite funcoes de score zero; a juncao usa `arquivo:linha:coluna` com o inventario completo do `gocyclo`, preenchendo cognitiva `0`. Entradas nao casadas: 0.

## Tooling pinado e comandos

Repositorio: `go 1.21`; ambiente de auditoria: `go version go1.26.5 darwin/arm64`.

| Ferramenta | Pin | Motivo |
|---|---|---|
| `github.com/fzipp/gocyclo/cmd/gocyclo` | `v0.6.0` | ultima tag disponivel e compativel com o modulo atual |
| `github.com/uudashr/gocognit/cmd/gocognit` | `v1.1.4` | pin compativel com Go 1.21 (`GoVersion: 1.19`); `v1.2.1` exige Go 1.24 e nao serve ao CI atual |

Instalacao fora do modulo, sem alterar `go.mod`/`go.sum`:

```sh
mkdir -p /tmp/relayer-complexity-tools
GOBIN=/tmp/relayer-complexity-tools go install github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0
GOBIN=/tmp/relayer-complexity-tools go install github.com/uudashr/gocognit/cmd/gocognit@v1.1.4
go version -m /tmp/relayer-complexity-tools/gocyclo
go version -m /tmp/relayer-complexity-tools/gocognit
```

Medicao completa (Bash; o array `manual` e formado pela mesma regra de escopo):

```sh
manual=()
while IFS= read -r file; do
  if ! sed -n '1,40p' "$file" | grep -Eq '^// Code generated .* DO NOT EDIT[.]$'; then
    manual+=("$file")
  fi
done < <(git ls-tree -r --name-only bef2e868f157659b403fe1303ee121fb69fec9e6 | grep -E '[.]go$')

/tmp/relayer-complexity-tools/gocyclo "${manual[@]}"
/tmp/relayer-complexity-tools/gocognit -test "${manual[@]}"
/tmp/relayer-complexity-tools/gocyclo -over 9 "${manual[@]}"
/tmp/relayer-complexity-tools/gocognit -test -over 9 "${manual[@]}"
```

Para o gate estrito, `-over 9` e a formulacao correta: falha se qualquer score for maior que 9, isto e, `>= 10`.

Hashes antes e depois da instalacao/medicao (inalterados):

```text
2d272a3b750b084760e52a81fd5488f8c9f97b9f  go.mod
7d9294772fbc5182453b823d30782a1bd9581458  go.sum
89c29c14bc079df86c0198a6cfd6499243ec49a5  interchaintest/go.mod
52dbaae324ff05a55ada59b79d9c8f1ef1e41ed5  interchaintest/go.sum
```

## Contrato atual de lint/build

- `Makefile:lint` executa `golangci-lint run`, `gofmt -d -s` e `go mod verify`, mas nao instala/pina `golangci-lint`.
- `.golangci.yml` deixa `gocyclo` e `gocognit` comentados; o unico `min-complexity: 10` tambem esta comentado.
- Com `golangci-lint v2.12.2`, `golangci-lint run` encerra antes da analise com status 3: `unsupported version of the configuration: ""`. A configuracao antiga nao declara `version` e referencia linters removidos; a migracao para schema v2 deve ser um lote de tooling separado.
- Nenhum workflow executa `make lint` ou uma medicao de complexidade. `build.yml` executa `make test` e `make build` com Go 1.21.
- Portanto nao existe hoje gate de complexidade local nem CI, e o lint existente tambem nao e totalmente reproduzivel por nao pinar a versao do agregador.
- A futura implementacao deve adicionar `make complexity` com as duas ferramentas pinadas fora do modulo e um job CI Go 1.21 que execute o alvo. Nao habilitar apenas um linter dentro de uma versao nao pinada de `golangci-lint`.

## Build/test baseline separado da complexidade

- `go build ./...`: passa no snapshot, conforme execucao paralela do lider.
- `go test -mod=readonly ./...`: 96 testes passaram e 1 falhou em 48 pacotes no ambiente auditado.
- Falha preexistente: `relayer/chains/cosmos.TestQueryBaseFee` em `fee_market_test.go:53`; consulta live do Osmosis devolve bytes protobuf iniciados por `0x11`, enquanto o parser tenta ler decimal base 10 (`failed to set decimal string with base 10`). Essa falha e de fixture/endpoint externo e nao foi causada pela medicao.
- Nao foram executados cenarios Docker de `interchaintest`; eles exigem infraestrutura externa e nao sao pre-condicao para produzir o baseline estatico.

## Distribuicao por componente

| Componente | Funcoes | Ciclo >=10 | Cognitiva >=10 | Uniao | Max ciclo | Max cognitiva |
|---|---:|---:|---:|---:|---:|---:|
| `cmd` | 199 | 28 | 50 | 51 | 26 | 67 |
| `relayer/processor` | 205 | 23 | 27 | 29 | 48 | 99 |
| `relayer/chains/cosmos` | 301 | 18 | 27 | 28 | 26 | 46 |
| `relayer/core+legacy` | 140 | 11 | 19 | 19 | 23 | 46 |
| `relayer/chains/penumbra` | 226 | 9 | 13 | 15 | 26 | 48 |
| `relayer/codecs` | 81 | 5 | 7 | 7 | 33 | 82 |
| `interchaintest` | 76 | 3 | 6 | 6 | 40 | 169 |
| `relayer/chains/other` | 30 | 1 | 2 | 2 | 20 | 15 |
| `support` | 67 | 0 | 1 | 1 | 8 | 12 |
| `root` | 2 | 0 | 0 | 0 | 1 | 0 |

## Hotspots que precisam de coordenacao com IBC/SDK

Nao refatorar estes pontos como limpeza isolada antes de fechar as decisoes de IBC v2. Eles codificam o fluxo v1 `events -> cache -> processor -> provider/messages -> broadcast -> ack/timeout/retry` e sao candidatos a substituicao, nao apenas extracao mecanica.

| Fronteira | Funcoes principais (ciclo/cognitiva) | Risco/decisao |
|---|---|---|
| Selecao e ordenacao de pacotes | `getMessagesToSend` 42/99; `queuePendingRecvAndAcks` 34/63; `unrelayedPacketFlowMessages` 14/22 | Modelo v1 baseado em channel/sequence/event type. Definir abstracao de payload/identifier IBC v2 antes de quebrar em helpers. |
| Lifecycle e terminacao | `queuePreInitMessages` 48/90; `shouldTerminate` 45/82; `shouldTerminateForFlushComplete` 24/43 | Type-switch central mistura packet/connection/channel/close. Separar por lifecycle somente junto da estrategia de coexistencia v1/v2. |
| Cache, ack, timeout e retry | `mergeMessageCache` 26/70; `shouldSendPacketMessage` 13/13; `shouldSendConnectionMessage` 13/11; `shouldSendChannelMessage` 24/48; `trackProcessingMessage` 16/27; `trackFinishedProcessingMessage` 16/23 | Ordem de efeitos, remocao de retencao e retry sao semantica critica. Criar testes de estado/transicao antes da extracao. |
| Orquestracao/flush | `PathProcessor.Run` 17/24; `flush` 20/40; `handleLocalhostData` 22/52 | Decidir se localhost, flush de connection/channel e processador legado permanecem no desenho novo. |
| Cosmos event/query/provider | `CosmosChainProcessor.queryCycle` 26/46; `queryIBCMessages` 8/16; `handleChannelMessage` 11/16; `CosmosProvider.buildMessages` 15/20; `SendMsgsWith` 15/16; `RelayPacketFromSequence` 14/29 | Adaptadores v1 atuais dependem de tipos/eventos ibc-go v8. Isolar interfaces antes de mudar SDK/ibc-go. |
| Penumbra event/query/provider | `PenumbraChainProcessor.queryCycle` 16/28; `handleChannelMessage` 11/16; `acknowledgementsFromResultTx` 26/48; `RelayPacketFromSequence` 14/29 | Manter paridade de provider sem forcar tipos Cosmos/Ibc-go v1 sobre Penumbra. |
| Parsing de eventos | `PacketInfo.parsePacketAttribute` 20/15; `getChannelsIfPresent` Cosmos 10/15 e Penumbra 10/15 | IBC v2 altera eventos/identificadores; parser deve virar adaptador versionado com fixtures. |
| Broadcast e batching | `messageProcessor.trackAndSendMessages` 12/14; `sendBatchMessages` 14/25; `RelayMsgs.send` 9/16; `waitForTx` 9/17 | Preservar batching, gas, confirmacao, retry e telemetria ao introduzir mensagens v2. |
| CLI/config | `linkCmd` 26/50; `createClientCmd` 23/45; `createConnectionCmd` 21/40; `flushCmd` 11/20; `Config.ValidatePathEnd` 12/10 | Nao cristalizar flags/objetos v1. Projetar UX/config de coexistencia v1/v2 antes de extrair builders. |
| Legado | `UnrelayedSequences` 23/46; `UnrelayedAcknowledgements` 14/23; `RelayAcknowledgements` 14/18; `relayerStartLegacy` 12/21 | Candidatos a remocao/substituicao, nao a uma grande refatoracao, depois de confirmar cobertura do processador event-based. |

Branches remotas antigas que podem servir apenas como fonte de ideias/testes, nunca para merge direto: `andrew/remove_legacy` (remove processador legado/consolida broadcast), `andrew/flush_channels_connections` (extrai `processor/flush.go`), `andrew/eibc` (MessageSender), `andrew/event_processor` (WIP event-based), `justin/run-logic` (lifecycle de Run) e `reece/gordian-consensus` (interface de consenso). Estao 18-369 commits atras da base; extrair seletivamente depois de rediff/testes.

## Proposta de lotes

A estimativa inicial e **8 trilhas / aproximadamente 12-18 PRs pequenas**, revisada depois de cada rebaseline. Uma PR nao deve misturar mudanca funcional IBC/SDK e reducao mecanica, exceto quando o codigo antigo sera removido pela nova API.

1. **Gate e caracterizacao (sem reduzir scores):** adicionar script/`make complexity`, pins, CI Go 1.21, fixtures deterministicas para `TestQueryBaseFee`, e testes de transicao do processor. O baseline continua vermelho ate os lotes seguintes; merge do gate pode usar modo report-only por prazo curto, mas o gate final nao pode aceitar baseline nem `nolint`.
2. **Testes gigantes:** decompor `TestRelayerFeeGrant` 40/169 e `TestRelayerFeeGrantExternal` 39/166 em setup/assertions/subtests independentes; depois demais testes >=10. Nao alterar comportamento de producao.
3. **CLI/config:** builders Cobra, validadores e I/O em helpers testaveis. Separar parsing/validacao/efeito; coordenar comandos v1/v2 antes de consolidar flags.
4. **Codecs Ethermint:** dividir `traverseFields` 33/82, `typToEth` 31/11 e EIP-712 em classificacao, conversao e traversal; adicionar tabelas de tipos e golden tests.
5. **Lifecycle processor:** substituir type-switches de `queuePreInitMessages`/`shouldTerminate` por estrategias pequenas com contratos de cache; somente apos o modelo de coexistencia IBC v1/v2.
6. **Packet state machine:** decompor selecao ordenada, cache merge, ack/timeout, retry e cleanup. E o lote de maior risco; exigir testes de transicao, ordered/unordered, timeout e duplicate/retry a cada PR.
7. **Providers/parsers/broadcast:** criar adaptadores versionados para evento/query/message e reduzir Cosmos/Penumbra em paralelo sem perder paridade; incluir metrics/log assertions e falhas de broadcast.
8. **Remocao do legado e consolidacao:** decidir com evidencias se `naive-strategy`, `relayerStartLegacy`, localhost especial e fluxos de flush antigos podem ser removidos. So depois limpar o restante e ativar gate obrigatorio zero-violacao.

Sequenciamento recomendado: 1 -> (2, 3, 4 em paralelo) -> 5 -> 6 -> 7 -> 8. Os lotes 5-8 dependem diretamente do roadmap IBC; refatorar todos agora geraria retrabalho e aumentaria risco de regressao.

## Exclusoes deterministicas de codigo gerado

Todos os 38 arquivos abaixo possuem marcador canonico no primeiro bloco de 40 linhas; nenhum arquivo foi excluido apenas pelo nome.

| Arquivo | Evidencia |
|---|---|
| `relayer/chains/cosmos/keys/sr25519/keys.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/cosmos/stride/messages.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/cnidarium/v1/cnidarium.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/app/v1/app.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/asset/v1/asset.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/auction/v1/auction.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/community_pool/v1/community_pool.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/compact_block/v1/compact_block.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/dex/v1/dex.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/distributions/v1/distributions.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/fee/v1/fee.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/funding/v1/funding.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/governance/v1/governance.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/ibc/v1/ibc.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/sct/v1/sct.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/shielded_pool/v1/shielded_pool.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/component/stake/v1/stake.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/keys/v1/keys.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/num/v1/num.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/transaction/v1/transaction.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/core/txhash/v1/txhash.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/crypto/decaf377_fmd/v1/decaf377_fmd.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/crypto/decaf377_frost/v1/decaf377_frost.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/crypto/decaf377_rdsa/v1/decaf377_rdsa.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/crypto/tct/v1/tct.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/custody/threshold/v1/threshold.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/custody/v1/custody.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/tools/summoning/v1/summoning.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/util/tendermint_proxy/v1/tendermint_proxy.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/chains/penumbra/view/v1/view.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/ethermint/account.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/ethermint/keys.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/ethermint/web3.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/injective/account.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/injective/evm.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/injective/keys.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/codecs/injective/tx.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |
| `relayer/ethermint/dynamic_fee.pb.go` | `// Code generated by protoc-gen-gogo. DO NOT EDIT.` |

## Ranking completo

Ordenacao: maior score entre as duas metricas, depois soma dos scores, caminho, funcao e linha. `FAIL` significa ciclomatica >=10 ou cognitiva >=10. Este inventario contem todas as 1327 funcoes manuscritas detectadas, inclusive scores baixos/zero cognitivo.

| Rank | Categoria | Pacote | Chave estavel (`arquivo:funcao`) | Linha | Ciclomatica | Cognitiva | Gate |
|---:|---|---|---|---:|---:|---:|---|
| 1 | test | `interchaintest` | `interchaintest/feegrant_test.go:TestRelayerFeeGrant` | 66 | 40 | 169 | FAIL |
| 2 | test | `interchaintest` | `interchaintest/feegrant_test.go:TestRelayerFeeGrantExternal` | 556 | 39 | 166 | FAIL |
| 3 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).getMessagesToSend` | 28 | 42 | 99 | FAIL |
| 4 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).queuePreInitMessages` | 741 | 48 | 90 | FAIL |
| 5 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).shouldTerminate` | 285 | 45 | 82 | FAIL |
| 6 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:traverseFields` | 169 | 33 | 82 | FAIL |
| 7 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).mergeMessageCache` | 150 | 26 | 70 | FAIL |
| 8 | production | `cmd` | `cmd/keys.go:keysRestoreCmd` | 147 | 23 | 67 | FAIL |
| 9 | production | `cmd` | `cmd/paths.go:pathsFetchCmd` | 363 | 19 | 65 | FAIL |
| 10 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).queuePendingRecvAndAcks` | 1200 | 34 | 63 | FAIL |
| 11 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).handleLocalhostData` | 442 | 22 | 52 | FAIL |
| 12 | production | `cmd` | `cmd/tx.go:linkCmd` | 658 | 26 | 50 | FAIL |
| 13 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).acknowledgementsFromResultTx` | 1779 | 26 | 48 | FAIL |
| 14 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).shouldSendChannelMessage` | 722 | 24 | 48 | FAIL |
| 15 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).queryCycle` | 342 | 26 | 46 | FAIL |
| 16 | production | `relayer` | `relayer/naive-strategy.go:UnrelayedSequences` | 16 | 23 | 46 | FAIL |
| 17 | production | `cmd` | `cmd/feegrant.go:feegrantConfigureBasicCmd` | 29 | 26 | 45 | FAIL |
| 18 | production | `cmd` | `cmd/tx.go:createClientCmd` | 144 | 23 | 45 | FAIL |
| 19 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).shouldTerminateForFlushComplete` | 1550 | 24 | 43 | FAIL |
| 20 | production | `cmd` | `cmd/tx.go:createConnectionCmd` | 372 | 21 | 40 | FAIL |
| 21 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).flush` | 1422 | 20 | 40 | FAIL |
| 22 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).EnsureBasicGrants` | 249 | 25 | 39 | FAIL |
| 23 | production | `cmd` | `cmd/chains.go:chainsListCmd` | 267 | 17 | 37 | FAIL |
| 24 | production | `cmd` | `cmd/tx.go:xfersend` | 1039 | 19 | 36 | FAIL |
| 25 | production | `cmd` | `cmd/paths.go:pathsUpdateCmd` | 268 | 14 | 36 | FAIL |
| 26 | production | `cmd` | `cmd/start.go:startCmd` | 36 | 17 | 33 | FAIL |
| 27 | production | `relayer` | `relayer/client.go:CreateClient` | 119 | 21 | 32 | FAIL |
| 28 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:typToEth` | 404 | 31 | 13 | FAIL |
| 29 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).RelayPacketFromSequence` | 1390 | 14 | 29 | FAIL |
| 30 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).RelayPacketFromSequence` | 1666 | 14 | 29 | FAIL |
| 31 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).queryCycle` | 281 | 16 | 28 | FAIL |
| 32 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).startLivelinessChecks` | 330 | 11 | 28 | FAIL |
| 33 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).startLivelinessChecks` | 283 | 11 | 28 | FAIL |
| 34 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).trackProcessingMessage` | 868 | 16 | 27 | FAIL |
| 35 | production | `cmd` | `cmd/query.go:queryBalancesCmd` | 329 | 12 | 27 | FAIL |
| 36 | production | `cmd` | `cmd/chains.go:chainsAddCmd` | 342 | 11 | 27 | FAIL |
| 37 | production | `cmd` | `cmd/tx.go:createClientsCmd` | 59 | 14 | 26 | FAIL |
| 38 | production | `cmd` | `cmd/config.go:configInitCmd` | 113 | 9 | 26 | FAIL |
| 39 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).sendBatchMessages` | 415 | 14 | 25 | FAIL |
| 40 | production | `cmd` | `cmd/keys.go:keysAddCmd` | 77 | 12 | 25 | FAIL |
| 41 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).Run` | 399 | 17 | 24 | FAIL |
| 42 | test | `stride_test` | `interchaintest/stride/setup_test.go:ModifyGenesisStride` | 120 | 13 | 24 | FAIL |
| 43 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).trackFinishedProcessingMessage` | 953 | 16 | 23 | FAIL |
| 44 | production | `relayer` | `relayer/naive-strategy.go:UnrelayedAcknowledgements` | 244 | 14 | 23 | FAIL |
| 45 | production | `relayer` | `relayer/packet-tx.go:(*Chain).SendTransferMsg` | 19 | 19 | 22 | FAIL |
| 46 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).unrelayedPacketFlowMessages` | 189 | 14 | 22 | FAIL |
| 47 | production | `cmd` | `cmd/query.go:queryChannelsPaginated` | 957 | 13 | 22 | FAIL |
| 48 | production | `relayer` | `relayer/strategies.go:relayerStartLegacy` | 205 | 12 | 21 | FAIL |
| 49 | production | `cmd` | `cmd/query.go:queryChannelsToChain` | 906 | 10 | 21 | FAIL |
| 50 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).buildMessages` | 618 | 15 | 20 | FAIL |
| 51 | production | `chains` | `relayer/chains/parsing.go:(*PacketInfo).parsePacketAttribute` | 216 | 20 | 15 | FAIL |
| 52 | production | `provider` | `relayer/provider/matcher.go:cometMatcher` | 82 | 13 | 20 | FAIL |
| 53 | production | `cmd` | `cmd/query.go:queryClientsExpiration` | 1225 | 12 | 20 | FAIL |
| 54 | production | `cmd` | `cmd/config.go:configShowCmd` | 57 | 11 | 20 | FAIL |
| 55 | production | `cmd` | `cmd/tx.go:createChannelCmd` | 504 | 11 | 20 | FAIL |
| 56 | production | `cmd` | `cmd/tx.go:flushCmd` | 879 | 11 | 20 | FAIL |
| 57 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).buildSignerConfig` | 568 | 10 | 20 | FAIL |
| 58 | production | `relayer` | `relayer/naive-strategy.go:RelayAcknowledgements` | 385 | 14 | 18 | FAIL |
| 59 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).PrepareFactory` | 1654 | 13 | 18 | FAIL |
| 60 | production | `cmd` | `cmd/chains.go:chainsRegistryList` | 217 | 11 | 18 | FAIL |
| 61 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GetGranteeValidBasicGrants` | 65 | 10 | 18 | FAIL |
| 62 | production | `cmd` | `cmd/config.go:addChainsFromDirectory` | 180 | 8 | 18 | FAIL |
| 63 | production | `cmd` | `cmd/config.go:addPathsFromDirectory` | 232 | 8 | 18 | FAIL |
| 64 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).Run` | 215 | 14 | 17 | FAIL |
| 65 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).waitForTx` | 431 | 10 | 17 | FAIL |
| 66 | production | `cmd` | `cmd/paths.go:pathsAddCmd` | 176 | 6 | 17 | FAIL |
| 67 | production | `penumbra` | `relayer/chains/penumbra/tx.go:msgToPenumbraAction` | 103 | 17 | 2 | FAIL |
| 68 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).SendMsgsWith` | 269 | 15 | 16 | FAIL |
| 69 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:decodeProtobufSignDoc` | 142 | 16 | 15 | FAIL |
| 70 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).handleChannelMessage` | 66 | 11 | 16 | FAIL |
| 71 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).handleChannelMessage` | 51 | 11 | 16 | FAIL |
| 72 | production | `relayer` | `relayer/query.go:QueryBalance` | 234 | 11 | 16 | FAIL |
| 73 | production | `cmd` | `cmd/query.go:queryBalanceCmd` | 258 | 10 | 16 | FAIL |
| 74 | production | `cmd` | `cmd/query.go:queryHeaderCmd` | 400 | 10 | 16 | FAIL |
| 75 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).queryIBCMessages` | 48 | 10 | 16 | FAIL |
| 76 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:validatePayloadMessages` | 255 | 10 | 16 | FAIL |
| 77 | production | `relayer` | `relayer/relayMsgs.go:(*RelayMsgs).send` | 178 | 10 | 16 | FAIL |
| 78 | production | `cmd` | `cmd/paths.go:pathsListCmd` | 64 | 9 | 16 | FAIL |
| 79 | production | `cmd` | `cmd/version.go:getVersionCmd` | 28 | 7 | 16 | FAIL |
| 80 | production | `cmd` | `cmd/tx.go:setPathsFromArgs` | 1165 | 15 | 14 | FAIL |
| 81 | production | `cmd` | `cmd/config.go:(*Config).ValidatePathEnd` | 617 | 12 | 15 | FAIL |
| 82 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).Run` | 156 | 12 | 15 | FAIL |
| 83 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryFeegrantsByGranter` | 237 | 9 | 15 | FAIL |
| 84 | production | `cmd` | `cmd/query.go:queryChannel` | 800 | 8 | 15 | FAIL |
| 85 | production | `cmd` | `cmd/query.go:queryClientCmd` | 504 | 8 | 15 | FAIL |
| 86 | production | `cmd` | `cmd/query.go:queryConnectionsUsingClient` | 650 | 8 | 15 | FAIL |
| 87 | production | `cosmos` | `relayer/chains/cosmos/log.go:getChannelsIfPresent` | 21 | 6 | 15 | FAIL |
| 88 | production | `penumbra` | `relayer/chains/penumbra/log.go:getChannelsIfPresent` | 19 | 6 | 15 | FAIL |
| 89 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).trackAndSendMessages` | 327 | 12 | 14 | FAIL |
| 90 | production | `cosmos` | `relayer/chains/cosmos/grpc_query.go:(*CosmosProvider).Invoke` | 34 | 11 | 14 | FAIL |
| 91 | production | `penumbra` | `relayer/chains/penumbra/grpc_query.go:(*PenumbraProvider).Invoke` | 34 | 11 | 14 | FAIL |
| 92 | production | `relayer` | `relayer/path.go:(*Path).QueryPathStatus` | 210 | 14 | 11 | FAIL |
| 93 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).sendSingleMessage` | 513 | 9 | 14 | FAIL |
| 94 | production | `cmd` | `cmd/tx.go:closeChannelCmd` | 596 | 8 | 14 | FAIL |
| 95 | test | `interchaintest_test` | `interchaintest/path_filter_test.go:TestScenarioPathFilterDeny` | 171 | 8 | 14 | FAIL |
| 96 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryFeegrantsByGrantee` | 198 | 8 | 14 | FAIL |
| 97 | production | `mock` | `relayer/chains/mock/mock_chain_processor.go:(*MockChainProcessor).queryCycle` | 110 | 8 | 14 | FAIL |
| 98 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).shouldSendPacketMessage` | 528 | 13 | 13 | FAIL |
| 99 | production | `cmd` | `cmd/start.go:setupDebugServer` | 219 | 12 | 13 | FAIL |
| 100 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).KeyAddOrRestore` | 92 | 10 | 13 | FAIL |
| 101 | production | `processor` | `relayer/processor/path_end.go:(PathEnd).checkChannelMatch` | 38 | 10 | 13 | FAIL |
| 102 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).shouldSendConnectionMessage` | 650 | 13 | 10 | FAIL |
| 103 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).unrelayedChannelHandshakeMessages` | 456 | 10 | 13 | FAIL |
| 104 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).unrelayedConnectionHandshakeMessages` | 327 | 10 | 13 | FAIL |
| 105 | production | `cmd` | `cmd/paths.go:pathsShowCmd` | 127 | 9 | 13 | FAIL |
| 106 | production | `cosmos` | `relayer/chains/cosmos/tx.go:parseEventsFromTxResponse` | 530 | 8 | 13 | FAIL |
| 107 | production | `cmd` | `cmd/feegrant.go:feegrantBasicGrantsCmd` | 182 | 7 | 13 | FAIL |
| 108 | production | `ethermint` | `relayer/codecs/ethermint/algorithm.go:(ethSecp256k1Algo).Derive` | 56 | 7 | 13 | FAIL |
| 109 | production | `injective` | `relayer/codecs/injective/algorithm.go:(ethSecp256k1Algo).Derive` | 55 | 7 | 13 | FAIL |
| 110 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).handleCallbacks` | 269 | 6 | 13 | FAIL |
| 111 | production | `relayer` | `relayer/client.go:(*Chain).CreateClients` | 20 | 10 | 12 | FAIL |
| 112 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).broadcastTx` | 379 | 9 | 12 | FAIL |
| 113 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).KeyAddOrRestore` | 88 | 9 | 12 | FAIL |
| 114 | test | `relayer_test` | `relayer/relaymsgs_test.go:TestRelayMsgs_Send_Errors` | 148 | 9 | 12 | FAIL |
| 115 | production | `cregistry` | `cregistry/chain_info.go:(ChainInfo).GetBackupRPCEndpoints` | 221 | 8 | 12 | FAIL |
| 116 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GetValidBasicGrants` | 22 | 8 | 12 | FAIL |
| 117 | production | `cmd` | `cmd/query.go:queryUnrelayedAcknowledgements` | 1173 | 7 | 12 | FAIL |
| 118 | production | `cmd` | `cmd/query.go:queryUnrelayedPackets` | 1120 | 7 | 12 | FAIL |
| 119 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).ShouldRelayChannel` | 1052 | 7 | 12 | FAIL |
| 120 | production | `cmd` | `cmd/chains.go:chainsShowCmd` | 131 | 6 | 12 | FAIL |
| 121 | production | `cosmos` | `relayer/chains/cosmos/log.go:getFeePayer` | 148 | 12 | 4 | FAIL |
| 122 | production | `penumbra` | `relayer/chains/penumbra/log.go:getFeePayer` | 138 | 12 | 4 | FAIL |
| 123 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GrantAllGranteesBasicAllowance` | 366 | 11 | 11 | FAIL |
| 124 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GrantAllGranteesBasicAllowanceWithExpiration` | 403 | 11 | 11 | FAIL |
| 125 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:decodeAminoSignDoc` | 70 | 11 | 11 | FAIL |
| 126 | production | `cmd` | `cmd/flags.go:getAddInputs` | 272 | 11 | 10 | FAIL |
| 127 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).assembleMsgUpdateClient` | 254 | 10 | 11 | FAIL |
| 128 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).broadcastTx` | 2146 | 8 | 11 | FAIL |
| 129 | production | `cmd` | `cmd/appstate.go:(*appState).updatePathConfig` | 248 | 7 | 11 | FAIL |
| 130 | production | `cmd` | `cmd/chains.go:addChainsFromRegistry` | 494 | 7 | 11 | FAIL |
| 131 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryConnectionsUsingClient` | 736 | 7 | 11 | FAIL |
| 132 | production | `relayer` | `relayer/strategies.go:applyChannelFilterRule` | 358 | 7 | 11 | FAIL |
| 133 | production | `cmd` | `cmd/keys.go:keysDeleteCmd` | 269 | 6 | 11 | FAIL |
| 134 | production | `cmd` | `cmd/query.go:queryConnectionChannels` | 750 | 6 | 11 | FAIL |
| 135 | production | `relayer` | `relayer/events.go:ParseChannelIDFromEvents` | 44 | 6 | 11 | FAIL |
| 136 | production | `relayer` | `relayer/events.go:ParseConnectionIDFromEvents` | 29 | 6 | 11 | FAIL |
| 137 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).processLatestMessages` | 916 | 6 | 11 | FAIL |
| 138 | production | `cmd` | `cmd/appstate.go:(*appState).addPathFromUserInput` | 127 | 10 | 9 | FAIL |
| 139 | production | `cmd` | `cmd/start.go:setupMetricsServer` | 175 | 9 | 10 | FAIL |
| 140 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).shouldUpdateClientNow` | 133 | 9 | 10 | FAIL |
| 141 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryRecvPacket` | 1059 | 8 | 10 | FAIL |
| 142 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QuerySendPacket` | 1031 | 8 | 10 | FAIL |
| 143 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryRecvPacket` | 981 | 8 | 10 | FAIL |
| 144 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QuerySendPacket` | 948 | 8 | 10 | FAIL |
| 145 | production | `provider` | `relayer/provider/matcher.go:checkTendermintMisbehaviour` | 170 | 7 | 10 | FAIL |
| 146 | production | `relayer` | `relayer/strategies.go:StartRelayer` | 36 | 7 | 10 | FAIL |
| 147 | production | `cmd` | `cmd/query.go:queryConnection` | 705 | 6 | 10 | FAIL |
| 148 | production | `cmd` | `cmd/query.go:queryTxs` | 205 | 6 | 10 | FAIL |
| 149 | production | `cmd` | `cmd/tx.go:upgradeClientsCmd` | 329 | 6 | 10 | FAIL |
| 150 | test | `interchaintest_test` | `interchaintest/path_filter_test.go:TestScenarioPathFilterAllow` | 24 | 6 | 10 | FAIL |
| 151 | test | `interchaintest_test` | `interchaintest/relay_many_test.go:TestRelayerMultiplePathsSingleProcess` | 23 | 6 | 10 | FAIL |
| 152 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).handleConnectionMessage` | 108 | 6 | 10 | FAIL |
| 153 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).handleConnectionMessage` | 92 | 6 | 10 | FAIL |
| 154 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).getUnrelayedClientICQMessages` | 659 | 6 | 10 | FAIL |
| 155 | production | `relayer` | `relayer/client.go:parseClientIDFromEvents` | 557 | 5 | 10 | FAIL |
| 156 | production | `relayer` | `relayer/events.go:ParseClientIDFromEvents` | 14 | 5 | 10 | FAIL |
| 157 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).processAvailableSignals` | 316 | 10 | 5 | FAIL |
| 158 | production | `processor` | `relayer/processor/types_internal.go:(channelIBCMessage).assemble` | 143 | 10 | 5 | FAIL |
| 159 | production | `relayer` | `relayer/log-chain.go:logFailedTx` | 13 | 9 | 9 | PASS |
| 160 | production | `relayer` | `relayer/naive-strategy.go:RelayPackets` | 473 | 9 | 9 | PASS |
| 161 | production | `cmd` | `cmd/flags.go:pathFilterFlags` | 198 | 9 | 8 | PASS |
| 162 | production | `cosmos` | `relayer/chains/cosmos/grpc_query.go:(*CosmosProvider).RunGRPCQuery` | 97 | 8 | 9 | PASS |
| 163 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).WaitForNBlocks` | 430 | 8 | 9 | PASS |
| 164 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).ValidatePacket` | 824 | 9 | 8 | PASS |
| 165 | production | `chains` | `relayer/chains/parsing.go:(*ClientInfo).parseClientAttribute` | 151 | 8 | 9 | PASS |
| 166 | production | `penumbra` | `relayer/chains/penumbra/grpc_query.go:(*PenumbraProvider).RunGRPCQuery` | 98 | 8 | 9 | PASS |
| 167 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).WaitForNBlocks` | 383 | 8 | 9 | PASS |
| 168 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ValidatePacket` | 1190 | 9 | 8 | PASS |
| 169 | production | `relayer` | `relayer/channel.go:(*Chain).CreateOpenChannels` | 16 | 8 | 9 | PASS |
| 170 | production | `relayer` | `relayer/strategies.go:relayUnrelayedPackets` | 412 | 8 | 9 | PASS |
| 171 | production | `cosmos` | `relayer/chains/cosmos/log.go:(*CosmosProvider).LogFailedTx` | 47 | 7 | 9 | PASS |
| 172 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).waitForBlockInclusion` | 489 | 7 | 9 | PASS |
| 173 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).AcknowledgementFromSequence` | 1734 | 9 | 7 | PASS |
| 174 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).waitForBlockInclusion` | 2240 | 7 | 9 | PASS |
| 175 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).waitForTx` | 2191 | 7 | 9 | PASS |
| 176 | production | `cregistry` | `cregistry/chain_info.go:(ChainInfo).GetAllRPCEndpoints` | 107 | 6 | 9 | PASS |
| 177 | production | `cmd` | `cmd/query.go:queryChannels` | 1038 | 5 | 9 | PASS |
| 178 | production | `cmd` | `cmd/query.go:queryClientsCmd` | 556 | 5 | 9 | PASS |
| 179 | production | `cmd` | `cmd/query.go:queryConnections` | 602 | 5 | 9 | PASS |
| 180 | production | `cosmos` | `relayer/chains/cosmos/log.go:(*CosmosProvider).LogSuccessTx` | 90 | 5 | 9 | PASS |
| 181 | production | `chains` | `relayer/chains/parsing.go:(*ClientICQInfo).parseAttribute` | 406 | 9 | 5 | PASS |
| 182 | production | `penumbra` | `relayer/chains/penumbra/log.go:(*PenumbraProvider).LogSuccessTx` | 79 | 5 | 9 | PASS |
| 183 | production | `processor` | `relayer/processor/types_internal.go:(channelProcessingCache).deleteMessages` | 439 | 5 | 9 | PASS |
| 184 | production | `processor` | `relayer/processor/types_internal.go:(connectionProcessingCache).deleteMessages` | 482 | 5 | 9 | PASS |
| 185 | production | `processor` | `relayer/processor/types_internal.go:(packetChannelMessageCache).deleteMessages` | 396 | 5 | 9 | PASS |
| 186 | test | `cmd_test` | `cmd/start_test.go:TestDebugServerFlags` | 160 | 4 | 9 | PASS |
| 187 | test | `cmd_test` | `cmd/start_test.go:TestMetricsServerFlags` | 22 | 4 | 9 | PASS |
| 188 | production | `cmd` | `cmd/root.go:newRootLogger` | 178 | 9 | 2 | PASS |
| 189 | production | `cmd` | `cmd/appstate.go:(*appState).loadConfigFile` | 55 | 8 | 8 | PASS |
| 190 | production | `cmd` | `cmd/appstate.go:(*appState).performConfigLockingOperation` | 199 | 8 | 8 | PASS |
| 191 | production | `cmd` | `cmd/flags.go:parseStuckPacketFromFlags` | 575 | 8 | 7 | PASS |
| 192 | production | `cmd` | `cmd/root.go:NewRootCmd` | 52 | 7 | 8 | PASS |
| 193 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).CurrentRelayerBalance` | 560 | 7 | 8 | PASS |
| 194 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).CalculateGas` | 1768 | 7 | 8 | PASS |
| 195 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).assembleMessages` | 184 | 7 | 8 | PASS |
| 196 | production | `cmd` | `cmd/chains.go:addChainFromURL` | 457 | 8 | 6 | PASS |
| 197 | test | `interchaintest_test` | `interchaintest/misbehaviour_test.go:MakeCommit` | 352 | 6 | 8 | PASS |
| 198 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:isValidGrant` | 115 | 6 | 8 | PASS |
| 199 | production | `relayer` | `relayer/client.go:findMatchingClient` | 513 | 6 | 8 | PASS |
| 200 | production | `cmd` | `cmd/keys.go:keysListCmd` | 330 | 5 | 8 | PASS |
| 201 | production | `cmd` | `cmd/query.go:queryIBCDenoms` | 77 | 5 | 8 | PASS |
| 202 | production | `cmd` | `cmd/query.go:queryNodeStateCmd` | 464 | 5 | 8 | PASS |
| 203 | production | `cmd` | `cmd/query.go:queryPacketCommitment` | 1077 | 5 | 8 | PASS |
| 204 | test | `interchaintest_test` | `interchaintest/client_threshold_test.go:TestScenarioClientThresholdUpdate` | 29 | 5 | 8 | PASS |
| 205 | test | `interchaintest_test` | `interchaintest/client_threshold_test.go:TestScenarioClientTrustingPeriodUpdate` | 184 | 5 | 8 | PASS |
| 206 | test | `stride_test` | `interchaintest/stride/setup_test.go:ModifyGenesisStrideCounterparty` | 166 | 5 | 8 | PASS |
| 207 | production | `relayer` | `relayer/relayMsgs.go:(*RelayMsgs).PrependMsgUpdateClient` | 45 | 5 | 8 | PASS |
| 208 | production | `cregistry` | `cregistry/chain_info.go:(ChainInfo).GetAssetList` | 262 | 7 | 7 | PASS |
| 209 | production | `cregistry` | `cregistry/cosmos_github_registry.go:(CosmosGithubRegistry).GetChain` | 55 | 7 | 7 | PASS |
| 210 | test | `interchaintest_test` | `interchaintest/multi_channel_test.go:TestMultipleChannelsOneConnection` | 21 | 7 | 7 | PASS |
| 211 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GetTxFeeGrant` | 213 | 7 | 7 | PASS |
| 212 | production | `cosmos` | `relayer/chains/cosmos/grpc_query.go:(*CosmosProvider).TxServiceBroadcast` | 151 | 7 | 7 | PASS |
| 213 | production | `penumbra` | `relayer/chains/penumbra/grpc_query.go:(*PenumbraProvider).TxServiceBroadcast` | 152 | 7 | 7 | PASS |
| 214 | production | `relayer` | `relayer/path.go:(Paths).PathsFromChains` | 71 | 7 | 7 | PASS |
| 215 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).mergeCacheData` | 465 | 7 | 7 | PASS |
| 216 | production | `relayer` | `relayer/strategies.go:relayUnrelayedAcks` | 495 | 7 | 7 | PASS |
| 217 | production | `cmd` | `cmd/query.go:printChannelWithExtendedInfo` | 864 | 7 | 6 | PASS |
| 218 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).clientState` | 171 | 6 | 7 | PASS |
| 219 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryTendermintProof` | 392 | 7 | 6 | PASS |
| 220 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).SendMessagesToMempool` | 159 | 6 | 7 | PASS |
| 221 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryTendermintProof` | 165 | 7 | 6 | PASS |
| 222 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ConnectionOpenTry` | 543 | 7 | 6 | PASS |
| 223 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).InjectTrustedFields` | 1924 | 6 | 7 | PASS |
| 224 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).SetChainProviderIfApplicable` | 212 | 6 | 7 | PASS |
| 225 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).unrelayedChannelCloseMessages` | 585 | 6 | 7 | PASS |
| 226 | production | `processor` | `relayer/processor/types_internal.go:(packetIBCMessage).assemble` | 55 | 7 | 6 | PASS |
| 227 | production | `provider` | `relayer/provider/matcher.go:CheckForMisbehaviour` | 49 | 6 | 7 | PASS |
| 228 | production | `relayer` | `relayer/query.go:QueryPortChannel` | 134 | 6 | 7 | PASS |
| 229 | production | `relayer` | `relayer/strategies.go:relayUnrelayedPacketsAndAcks` | 384 | 6 | 7 | PASS |
| 230 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).GetWallet` | 338 | 5 | 7 | PASS |
| 231 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).ConfigureWithGrantees` | 146 | 5 | 7 | PASS |
| 232 | production | `penumbra` | `relayer/chains/penumbra/tx.go:parseEventsFromTxResponse` | 418 | 5 | 7 | PASS |
| 233 | production | `relayer` | `relayer/naive-strategy.go:AddMessagesForSequences` | 550 | 5 | 7 | PASS |
| 234 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessorBuilder).Build` | 73 | 5 | 7 | PASS |
| 235 | production | `processor` | `relayer/processor/types.go:(ChannelStateCache).FilterForClient` | 286 | 5 | 7 | PASS |
| 236 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgRelayAcknowledgement` | 951 | 7 | 4 | PASS |
| 237 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessorBuilder).WithChainProcessors` | 34 | 4 | 7 | PASS |
| 238 | production | `processor` | `relayer/processor/types_internal.go:(connectionIBCMessage).assemble` | 223 | 7 | 4 | PASS |
| 239 | production | `relayer` | `relayer/query.go:QueryClientStates` | 39 | 4 | 7 | PASS |
| 240 | production | `chains` | `relayer/chains/parsing.go:(*ChannelInfo).parseChannelAttribute` | 330 | 7 | 1 | PASS |
| 241 | production | `cregistry` | `cregistry/chain_info.go:(ChainInfo).GetChainConfig` | 298 | 6 | 6 | PASS |
| 242 | production | `cregistry` | `cregistry/cosmos_github_registry.go:(CosmosGithubRegistry).ListChains` | 29 | 6 | 6 | PASS |
| 243 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).InjectTrustedFields` | 1498 | 6 | 6 | PASS |
| 244 | production | `penumbra` | `relayer/chains/penumbra/log.go:(*PenumbraProvider).LogFailedTx` | 45 | 6 | 6 | PASS |
| 245 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgRelayTimeout` | 1029 | 6 | 6 | PASS |
| 246 | production | `processor` | `relayer/processor/types.go:(ChannelPacketMessagesCache).ShouldRetainSequence` | 466 | 6 | 6 | PASS |
| 247 | production | `cmd` | `cmd/chains.go:addChainFromFile` | 423 | 6 | 5 | PASS |
| 248 | production | `cmd` | `cmd/config.go:UnmarshalJSONProviderConfig` | 391 | 6 | 5 | PASS |
| 249 | production | `cmd` | `cmd/flags.go:clientParameterFlags` | 325 | 6 | 5 | PASS |
| 250 | production | `cmd` | `cmd/flags.go:paginationFlags` | 90 | 6 | 5 | PASS |
| 251 | production | `cregistry` | `cregistry/chain_info.go:(ChainInfo).GetRPCEndpoints` | 155 | 5 | 6 | PASS |
| 252 | test | `interchaintest_test` | `interchaintest/backup_rpc_test.go:TestBackupRpcs` | 25 | 5 | 6 | PASS |
| 253 | test | `interchaintest_test` | `interchaintest/misbehaviour_test.go:createTMClientHeader` | 270 | 5 | 6 | PASS |
| 254 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).GetChannels` | 133 | 5 | 6 | PASS |
| 255 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).GetClients` | 154 | 5 | 6 | PASS |
| 256 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).GetConnections` | 189 | 5 | 6 | PASS |
| 257 | production | `cosmos` | `relayer/chains/cosmos/account.go:(*CosmosProvider).GetAccountWithHeight` | 28 | 6 | 5 | PASS |
| 258 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(latestClientState).update` | 92 | 5 | 6 | PASS |
| 259 | production | `cosmos` | `relayer/chains/cosmos/fee_market.go:(*CosmosProvider).QueryBaseFee` | 33 | 6 | 5 | PASS |
| 260 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).ListAddresses` | 161 | 5 | 6 | PASS |
| 261 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryConnections` | 710 | 5 | 6 | PASS |
| 262 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryDenomTraces` | 1237 | 5 | 6 | PASS |
| 263 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryTxs` | 142 | 6 | 5 | PASS |
| 264 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).queryParamsSubspaceTime` | 320 | 6 | 5 | PASS |
| 265 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).ListAddresses` | 146 | 5 | 6 | PASS |
| 266 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryTxs` | 62 | 6 | 5 | PASS |
| 267 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).queryIBCMessages` | 912 | 6 | 5 | PASS |
| 268 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelOpenTry` | 731 | 6 | 5 | PASS |
| 269 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ConnectionOpenAck` | 610 | 6 | 5 | PASS |
| 270 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgUpdateClient` | 470 | 6 | 5 | PASS |
| 271 | production | `relayer` | `relayer/client.go:UpgradeClient` | 437 | 6 | 5 | PASS |
| 272 | production | `relayer` | `relayer/ics24.go:(*PathEnd).ValidateFull` | 46 | 5 | 6 | PASS |
| 273 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).checkForMisbehaviour` | 437 | 6 | 5 | PASS |
| 274 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).shouldSendClientICQMessage` | 836 | 6 | 5 | PASS |
| 275 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).updateClientTrustedState` | 694 | 6 | 5 | PASS |
| 276 | test | `processor_test` | `relayer/processor/types_test.go:TestPacketSequenceStateCachePrune` | 46 | 5 | 6 | PASS |
| 277 | production | `cmd` | `cmd/keys.go:keysExportCmd` | 368 | 4 | 6 | PASS |
| 278 | production | `cmd` | `cmd/query.go:queryTx` | 171 | 4 | 6 | PASS |
| 279 | production | `cmd` | `cmd/tx.go:updateClientsCmd` | 298 | 4 | 6 | PASS |
| 280 | production | `ethermint` | `relayer/codecs/ethermint/chain_id.go:ParseChainID` | 29 | 6 | 4 | PASS |
| 281 | production | `processor` | `relayer/processor/types.go:(ChannelMessagesCache).DeleteMessages` | 570 | 4 | 6 | PASS |
| 282 | production | `processor` | `relayer/processor/types.go:(ConnectionMessagesCache).DeleteMessages` | 535 | 4 | 6 | PASS |
| 283 | production | `processor` | `relayer/processor/types.go:(PacketMessagesCache).DeleteMessages` | 345 | 4 | 6 | PASS |
| 284 | production | `cmd` | `cmd/paths.go:pathsNewCmd` | 234 | 3 | 6 | PASS |
| 285 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgRelayRecvPacket` | 1150 | 6 | 3 | PASS |
| 286 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).handleMessage` | 16 | 6 | 1 | PASS |
| 287 | production | `chains` | `relayer/chains/parsing.go:ParseIBCMessageFromEvent` | 56 | 6 | 1 | PASS |
| 288 | production | `cmd` | `cmd/appstate.go:(*appState).useKey` | 279 | 5 | 5 | PASS |
| 289 | production | `cmd` | `cmd/keys.go:(*appState).showAddressByChainAndKey` | 402 | 5 | 5 | PASS |
| 290 | test | `interchaintest_test` | `interchaintest/memo_receiver_limit_test.go:TestMemoAndReceiverLimit` | 22 | 5 | 5 | PASS |
| 291 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).ConfigureWithExternalGranter` | 170 | 5 | 5 | PASS |
| 292 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).KeyFromKeyOrAddress` | 244 | 5 | 5 | PASS |
| 293 | production | `mock` | `relayer/chains/mock/mock_chain_processor.go:(*MockChainProcessor).Run` | 76 | 5 | 5 | PASS |
| 294 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).SendMessages` | 358 | 5 | 5 | PASS |
| 295 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).sendMessagesInner` | 304 | 5 | 5 | PASS |
| 296 | production | `relayer` | `relayer/client.go:UpdateClients` | 376 | 5 | 5 | PASS |
| 297 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:WrapTxToTypedData` | 29 | 5 | 5 | PASS |
| 298 | production | `relayer` | `relayer/query.go:QueryChannel` | 91 | 5 | 5 | PASS |
| 299 | production | `cmd` | `cmd/appstate.go:(*appState).addPathFromFile` | 102 | 5 | 4 | PASS |
| 300 | production | `cmd` | `cmd/config.go:(*Config).AddPath` | 561 | 5 | 4 | PASS |
| 301 | production | `cmd` | `cmd/config.go:(*Config).ChainsFromPath` | 446 | 5 | 4 | PASS |
| 302 | production | `cmd` | `cmd/config.go:(*ConfigInputWrapper).RuntimeConfig` | 333 | 4 | 5 | PASS |
| 303 | production | `cmd` | `cmd/tx.go:linkThenStartCmd` | 834 | 4 | 5 | PASS |
| 304 | production | `cregistry` | `cregistry/chain_info.go:(ChainInfo).GetRandomRPCEndpoint` | 196 | 4 | 5 | PASS |
| 305 | test | `interchaintest` | `interchaintest/docker.go:handleDockerBuildOutput` | 80 | 4 | 5 | PASS |
| 306 | test | `interchaintest_test` | `interchaintest/ica_channel_close_test.go:TestScenarioICAChannelClose` | 24 | 5 | 4 | PASS |
| 307 | test | `interchaintest_test` | `interchaintest/interchain_accounts_test.go:TestScenarioInterchainAccounts` | 25 | 5 | 4 | PASS |
| 308 | production | `relayer` | `relayer/chain.go:ValidateConnectionPaths` | 56 | 5 | 4 | PASS |
| 309 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).handlePacketMessage` | 31 | 5 | 4 | PASS |
| 310 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).logPacketMessage` | 154 | 5 | 4 | PASS |
| 311 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).Init` | 288 | 5 | 4 | PASS |
| 312 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryBalanceWithAddress` | 294 | 4 | 5 | PASS |
| 313 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryChannels` | 900 | 4 | 5 | PASS |
| 314 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryClientConsensusState` | 485 | 5 | 4 | PASS |
| 315 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryClientStateResponse` | 436 | 5 | 4 | PASS |
| 316 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryClients` | 632 | 4 | 5 | PASS |
| 317 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryConnectionChannels` | 873 | 4 | 5 | PASS |
| 318 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryConsensusStateABCI` | 1295 | 5 | 4 | PASS |
| 319 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryPacketAcknowledgements` | 974 | 4 | 5 | PASS |
| 320 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryPacketCommitments` | 945 | 4 | 5 | PASS |
| 321 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryUpgradedClient` | 552 | 5 | 4 | PASS |
| 322 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryUpgradedConsState` | 579 | 5 | 4 | PASS |
| 323 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).AdjustEstimatedGas` | 1724 | 5 | 4 | PASS |
| 324 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).AwaitTx` | 237 | 4 | 5 | PASS |
| 325 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgUpdateClientHeader` | 1302 | 5 | 4 | PASS |
| 326 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).QueryABCI` | 1843 | 5 | 4 | PASS |
| 327 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).SetWithExtensionOptions` | 1745 | 4 | 5 | PASS |
| 328 | production | `cosmos` | `relayer/chains/cosmos/tx.go:BuildSimTx` | 1902 | 5 | 4 | PASS |
| 329 | test | `mock_test` | `relayer/chains/mock/mock_chain_processor_test.go:getMockMessages` | 141 | 5 | 4 | PASS |
| 330 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).logPacketMessage` | 129 | 5 | 4 | PASS |
| 331 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).Init` | 248 | 5 | 4 | PASS |
| 332 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(PenumbraProviderConfig).NewProvider` | 81 | 5 | 4 | PASS |
| 333 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryClientConsensusState` | 260 | 5 | 4 | PASS |
| 334 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryClientStateResponse` | 210 | 5 | 4 | PASS |
| 335 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryConsensusStateABCI` | 878 | 5 | 4 | PASS |
| 336 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryHeaderAtHeight` | 800 | 5 | 4 | PASS |
| 337 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryUpgradedClient` | 327 | 5 | 4 | PASS |
| 338 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryUpgradedConsState` | 354 | 5 | 4 | PASS |
| 339 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelOpenAck` | 786 | 5 | 4 | PASS |
| 340 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelOpenConfirm` | 825 | 5 | 4 | PASS |
| 341 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ConnectionOpenConfirm` | 662 | 5 | 4 | PASS |
| 342 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgUpdateClientHeader` | 1630 | 5 | 4 | PASS |
| 343 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).QueryABCI` | 2075 | 5 | 4 | PASS |
| 344 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).getAnchor` | 250 | 5 | 4 | PASS |
| 345 | production | `relayer` | `relayer/channel.go:ValidateChannelParams` | 194 | 5 | 4 | PASS |
| 346 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:GetEIP712TypedDataForMsg` | 47 | 5 | 4 | PASS |
| 347 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:sanitizeTypedef` | 370 | 4 | 5 | PASS |
| 348 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).processMessages` | 95 | 4 | 5 | PASS |
| 349 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).trackers` | 61 | 5 | 4 | PASS |
| 350 | production | `processor` | `relayer/processor/types.go:(ChannelPacketStateCache).UpdateState` | 369 | 4 | 5 | PASS |
| 351 | production | `relayer` | `relayer/query.go:QueryClientExpiration` | 278 | 5 | 4 | PASS |
| 352 | production | `cmd` | `cmd/chains.go:cmdChainsUseBackupRpcAddr` | 73 | 3 | 5 | PASS |
| 353 | test | `cmd_test` | `cmd/config_test.go:TestDefaultConfig` | 14 | 3 | 5 | PASS |
| 354 | test | `cmd_test` | `cmd/start_test.go:TestDebugServerConfig` | 247 | 3 | 5 | PASS |
| 355 | test | `cmd_test` | `cmd/start_test.go:TestMetricsServerConfig` | 98 | 3 | 5 | PASS |
| 356 | test | `cosmos` | `relayer/chains/cosmos/tx_test.go:TestCosmosProvider_AdjustEstimatedGas` | 20 | 3 | 5 | PASS |
| 357 | production | `cosmos` | `relayer/chains/cosmos/tx.go:isQueryStoreWithProof` | 1881 | 5 | 2 | PASS |
| 358 | production | `penumbra` | `relayer/chains/penumbra/tx.go:isQueryStoreWithProof` | 2113 | 5 | 2 | PASS |
| 359 | production | `chains` | `relayer/chains/parsing.go:(*ConnectionInfo).parseConnectionAttribute` | 364 | 5 | 1 | PASS |
| 360 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).handleMessage` | 13 | 5 | 1 | PASS |
| 361 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).trackMessage` | 47 | 5 | 1 | PASS |
| 362 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).removePacketRetention` | 611 | 5 | 1 | PASS |
| 363 | production | `processor` | `relayer/processor/types.go:stateValue` | 355 | 5 | 1 | PASS |
| 364 | production | `cmd` | `cmd/config.go:(*Config).validateConfig` | 589 | 4 | 4 | PASS |
| 365 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).Flush` | 323 | 4 | 4 | PASS |
| 366 | test | `stride_test` | `interchaintest/stride/stride_icq_test.go:TestScenarioStrideICAandICQ` | 27 | 4 | 4 | PASS |
| 367 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).initializeChannelState` | 309 | 4 | 4 | PASS |
| 368 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).TrustingPeriod` | 234 | 4 | 4 | PASS |
| 369 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).setRpcClient` | 398 | 4 | 4 | PASS |
| 370 | production | `chains` | `relayer/chains/parsing.go:IbcMessagesFromEvents` | 33 | 4 | 4 | PASS |
| 371 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).initializeChannelState` | 243 | 4 | 4 | PASS |
| 372 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).setRpcClient` | 351 | 4 | 4 | PASS |
| 373 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:walkFields` | 143 | 4 | 4 | PASS |
| 374 | production | `injective` | `relayer/codecs/injective/params.go:validateEIPs` | 90 | 4 | 4 | PASS |
| 375 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).metricParseTxFailureCatagory` | 600 | 4 | 4 | PASS |
| 376 | production | `processor` | `relayer/processor/types.go:(IBCHeaderCache).Prune` | 642 | 4 | 4 | PASS |
| 377 | production | `cmd` | `cmd/appstate.go:(*appState).initLogger` | 34 | 4 | 3 | PASS |
| 378 | production | `cmd` | `cmd/chains.go:chainsAddrCmd` | 104 | 3 | 4 | PASS |
| 379 | production | `cmd` | `cmd/config.go:(*Config).AddChain` | 517 | 4 | 3 | PASS |
| 380 | production | `cmd` | `cmd/config.go:(*Config).ValidateConnection` | 674 | 4 | 3 | PASS |
| 381 | production | `cmd` | `cmd/config.go:checkPathEndConflict` | 540 | 4 | 3 | PASS |
| 382 | production | `cmd` | `cmd/flags.go:debugServerFlags` | 424 | 4 | 3 | PASS |
| 383 | production | `cmd` | `cmd/flags.go:stuckPacketFlags` | 559 | 4 | 3 | PASS |
| 384 | production | `cmd` | `cmd/keys.go:askForConfirmation` | 310 | 4 | 3 | PASS |
| 385 | production | `cmd` | `cmd/query.go:queryBaseDenomFromIBCDenom` | 113 | 3 | 4 | PASS |
| 386 | production | `cmd` | `cmd/query.go:queryIBCDenomHash` | 142 | 3 | 4 | PASS |
| 387 | production | `cmd` | `cmd/root.go:Execute` | 135 | 4 | 3 | PASS |
| 388 | production | `cmd` | `cmd/tx.go:registerCounterpartyCmd` | 1218 | 3 | 4 | PASS |
| 389 | production | `cregistry` | `cregistry/chain_info.go:IsHealthyRPC` | 136 | 4 | 3 | PASS |
| 390 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).StartRelayer` | 256 | 4 | 3 | PASS |
| 391 | production | `cosmos` | `relayer/chains/cosmos/codec.go:MakeCodec` | 72 | 4 | 3 | PASS |
| 392 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).getMsgGrantBasicAllowance` | 470 | 4 | 3 | PASS |
| 393 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).getMsgGrantBasicAllowanceWithExpiration` | 439 | 4 | 3 | PASS |
| 394 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).ShowAddress` | 144 | 4 | 3 | PASS |
| 395 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/algo.go:(sr25519Algo).Derive` | 21 | 3 | 4 | PASS |
| 396 | production | `cosmos` | `relayer/chains/cosmos/msg.go:CosmosMsgs` | 35 | 3 | 4 | PASS |
| 397 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).Address` | 193 | 4 | 3 | PASS |
| 398 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).GenerateConnHandshakeProof` | 768 | 4 | 3 | PASS |
| 399 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryChannel` | 806 | 4 | 3 | PASS |
| 400 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryConnection` | 658 | 4 | 3 | PASS |
| 401 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryUnbondingPeriod` | 356 | 4 | 3 | PASS |
| 402 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryUpgradeProof` | 519 | 4 | 3 | PASS |
| 403 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).queryChannelABCI` | 832 | 4 | 3 | PASS |
| 404 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).queryConnectionABCI` | 682 | 4 | 3 | PASS |
| 405 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).AcknowledgementFromSequence` | 1458 | 4 | 3 | PASS |
| 406 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgCreateClient` | 727 | 4 | 3 | PASS |
| 407 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).SendMessages` | 113 | 4 | 3 | PASS |
| 408 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).UpdateFeesSpent` | 1625 | 4 | 3 | PASS |
| 409 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).handleAccountSequenceMismatchError` | 710 | 4 | 3 | PASS |
| 410 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).queryLocalhostClientState` | 1571 | 4 | 3 | PASS |
| 411 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).queryTMClientState` | 1549 | 4 | 3 | PASS |
| 412 | production | `penumbra` | `relayer/chains/penumbra/codec.go:makeCodec` | 65 | 4 | 3 | PASS |
| 413 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).ShowAddress` | 129 | 4 | 3 | PASS |
| 414 | production | `penumbra` | `relayer/chains/penumbra/msg.go:PenumbraMsgs` | 46 | 4 | 3 | PASS |
| 415 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).Address` | 182 | 4 | 3 | PASS |
| 416 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).GenerateConnHandshakeProof` | 496 | 4 | 3 | PASS |
| 417 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryChannel` | 535 | 4 | 3 | PASS |
| 418 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryConnection` | 420 | 4 | 3 | PASS |
| 419 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryUpgradeProof` | 294 | 4 | 3 | PASS |
| 420 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).queryChannelABCI` | 562 | 4 | 3 | PASS |
| 421 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).queryConnectionABCI` | 444 | 4 | 3 | PASS |
| 422 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).GetLightSignedHeaderAtHeight` | 1897 | 4 | 3 | PASS |
| 423 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgCreateClient` | 441 | 4 | 3 | PASS |
| 424 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).orderedChannelTimeoutMsg` | 1065 | 4 | 3 | PASS |
| 425 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).unorderedChannelTimeoutMsg` | 1107 | 4 | 3 | PASS |
| 426 | production | `relayer` | `relayer/client.go:MsgUpdateClient` | 277 | 4 | 3 | PASS |
| 427 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:getMsgType` | 305 | 4 | 3 | PASS |
| 428 | production | `processor` | `relayer/processor/path_end_runtime.go:checkMaxReceiverSize` | 128 | 4 | 3 | PASS |
| 429 | production | `processor` | `relayer/processor/path_end_runtime.go:checkMemoLimit` | 108 | 4 | 3 | PASS |
| 430 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).HandleNewData` | 286 | 4 | 3 | PASS |
| 431 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).channelPairs` | 189 | 4 | 3 | PASS |
| 432 | production | `processor` | `relayer/processor/path_processor_internal.go:queryPacketCommitments` | 1166 | 3 | 4 | PASS |
| 433 | production | `processor` | `relayer/processor/types.go:(ChannelPacketMessagesCache).IsCached` | 426 | 4 | 3 | PASS |
| 434 | production | `processor` | `relayer/processor/types.go:(ClientICQMessagesCache).Merge` | 597 | 3 | 4 | PASS |
| 435 | production | `processor` | `relayer/processor/types.go:(PacketSequenceStateCache).Prune` | 409 | 4 | 3 | PASS |
| 436 | production | `provider` | `relayer/provider/matcher.go:ClientsMatch` | 27 | 4 | 3 | PASS |
| 437 | production | `relayer` | `relayer/relayMsgs.go:(*RelayMsgs).IsMaxTx` | 75 | 4 | 3 | PASS |
| 438 | production | `relayer` | `relayer/relayMsgs.go:(*RelayMsgs).Ready` | 34 | 4 | 3 | PASS |
| 439 | production | `relayer` | `relayer/relayMsgs.go:(SendMsgsResult).MarshalLogObject` | 135 | 3 | 4 | PASS |
| 440 | production | `relayer` | `relayer/relayMsgs.go:(SendMsgsResult).PartiallySent` | 122 | 4 | 3 | PASS |
| 441 | production | `cmd` | `cmd/config.go:(*ProviderConfigYAMLWrapper).UnmarshalYAML` | 421 | 4 | 2 | PASS |
| 442 | production | `relayer` | `relayer/path.go:(*Path).ValidateChannelFilterRule` | 145 | 4 | 2 | PASS |
| 443 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertPubKey` | 568 | 4 | 1 | PASS |
| 444 | production | `cosmos` | `relayer/chains/cosmos/tx.go:sdkErrorToGRPCError` | 1866 | 4 | 1 | PASS |
| 445 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgPacketAck).FetchCommitResponse` | 219 | 4 | 1 | PASS |
| 446 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).FetchCommitResponse` | 121 | 4 | 1 | PASS |
| 447 | production | `penumbra` | `relayer/chains/penumbra/tx.go:sdkErrorToGRPCError` | 2098 | 4 | 1 | PASS |
| 448 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:isValidEIP712Payload` | 64 | 4 | 1 | PASS |
| 449 | production | `cclient` | `cclient/cmbft_client_wrapper.go:converStringEvents` | 431 | 3 | 3 | PASS |
| 450 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertEvents` | 408 | 3 | 3 | PASS |
| 451 | production | `cmd` | `cmd/tx.go:ensureKeysExist` | 1207 | 3 | 3 | PASS |
| 452 | test | `interchaintest_test` | `interchaintest/fee_middleware_test.go:TestRelayerFeeMiddleware` | 20 | 3 | 3 | PASS |
| 453 | test | `interchaintest_test` | `interchaintest/localhost_client_test.go:TestLocalhost_InterchainAccounts` | 286 | 3 | 3 | PASS |
| 454 | test | `interchaintest_test` | `interchaintest/localhost_client_test.go:TestLocalhost_TokenTransfers` | 70 | 3 | 3 | PASS |
| 455 | test | `interchaintest_test` | `interchaintest/misbehaviour_test.go:TestRelayerMisbehaviourDetection` | 42 | 3 | 3 | PASS |
| 456 | production | `relayer` | `relayer/chain.go:(Chains).Get` | 108 | 3 | 3 | PASS |
| 457 | production | `relayer` | `relayer/chain.go:(Chains).Gets` | 127 | 3 | 3 | PASS |
| 458 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).CollectMetrics` | 541 | 3 | 3 | PASS |
| 459 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*FeeGrantConfiguration).AddGranteeKeys` | 194 | 3 | 3 | PASS |
| 460 | production | `cosmos` | `relayer/chains/cosmos/query.go:parseEventsFromResponseDeliverTx` | 178 | 3 | 3 | PASS |
| 461 | production | `chains` | `relayer/chains/parsing.go:(*ClientICQInfo).ParseAttrs` | 398 | 3 | 3 | PASS |
| 462 | production | `chains` | `relayer/chains/parsing.go:(*IbcMessage).parseIBCPacketReceiveMessageFromEvent` | 92 | 3 | 3 | PASS |
| 463 | production | `penumbra` | `relayer/chains/penumbra/query.go:parseEventsFromResponseDeliverTx` | 98 | 3 | 3 | PASS |
| 464 | production | `penumbra` | `relayer/chains/penumbra/tx.go:parseEventsFromABCIResponse` | 285 | 3 | 3 | PASS |
| 465 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:doRecover` | 465 | 3 | 3 | PASS |
| 466 | production | `relayer` | `relayer/path.go:(*ChannelFilter).InChannelList` | 154 | 3 | 3 | PASS |
| 467 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).assembledCount` | 220 | 3 | 3 | PASS |
| 468 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).isRelevantChannel` | 98 | 3 | 3 | PASS |
| 469 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).isRelevantConnection` | 89 | 3 | 3 | PASS |
| 470 | production | `processor` | `relayer/processor/path_processor.go:(PathProcessors).IsRelayedChannel` | 89 | 3 | 3 | PASS |
| 471 | production | `processor` | `relayer/processor/types.go:(ChannelMessagesCache).Merge` | 553 | 3 | 3 | PASS |
| 472 | production | `processor` | `relayer/processor/types.go:(ChannelPacketMessagesCache).Merge` | 456 | 3 | 3 | PASS |
| 473 | production | `processor` | `relayer/processor/types.go:(ChannelStateCache).SetOpen` | 270 | 3 | 3 | PASS |
| 474 | production | `processor` | `relayer/processor/types.go:(ClientICQMessagesCache).DeleteMessages` | 616 | 3 | 3 | PASS |
| 475 | production | `processor` | `relayer/processor/types.go:(ConnectionMessagesCache).Merge` | 518 | 3 | 3 | PASS |
| 476 | production | `processor` | `relayer/processor/types.go:(ConnectionStateCache).FilterForClient` | 309 | 3 | 3 | PASS |
| 477 | production | `processor` | `relayer/processor/types.go:(PacketMessagesCache).Clone` | 333 | 3 | 3 | PASS |
| 478 | production | `processor` | `relayer/processor/types.go:(PacketMessagesCache).Merge` | 501 | 3 | 3 | PASS |
| 479 | production | `relayer` | `relayer/query.go:SPrintClientExpirationJson` | 329 | 3 | 3 | PASS |
| 480 | production | `relayer` | `relayer/strategies.go:filterOpenChannels` | 341 | 3 | 3 | PASS |
| 481 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BlockResults` | 247 | 3 | 2 | PASS |
| 482 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BlockSearch` | 371 | 3 | 2 | PASS |
| 483 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BlockchainInfo` | 277 | 3 | 2 | PASS |
| 484 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).Commit` | 308 | 3 | 2 | PASS |
| 485 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).TxSearch` | 348 | 3 | 2 | PASS |
| 486 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).Validators` | 152 | 3 | 2 | PASS |
| 487 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertBlock` | 487 | 3 | 2 | PASS |
| 488 | production | `cmd` | `cmd/chains.go:chainsDeleteCmd` | 180 | 2 | 3 | PASS |
| 489 | production | `cmd` | `cmd/config.go:(*Config).ValidateClient` | 660 | 3 | 2 | PASS |
| 490 | production | `cmd` | `cmd/config.go:(*Config).ValidatePath` | 606 | 3 | 2 | PASS |
| 491 | production | `cmd` | `cmd/config.go:checkPathConflict` | 530 | 3 | 2 | PASS |
| 492 | production | `cmd` | `cmd/config.go:rlyMemo` | 288 | 3 | 2 | PASS |
| 493 | production | `cmd` | `cmd/flags.go:metricsServerFlags` | 462 | 3 | 2 | PASS |
| 494 | production | `cmd` | `cmd/flags.go:timeoutFlags` | 154 | 3 | 2 | PASS |
| 495 | production | `cmd` | `cmd/paths.go:pathsDeleteCmd` | 42 | 2 | 3 | PASS |
| 496 | test | `interchaintest` | `interchaintest/feegrant_test.go:genMnemonic` | 46 | 3 | 2 | PASS |
| 497 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).AddKey` | 92 | 3 | 2 | PASS |
| 498 | production | `relayer` | `relayer/chain.go:ValidateClientPaths` | 44 | 3 | 2 | PASS |
| 499 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).initializeConnectionState` | 289 | 3 | 2 | PASS |
| 500 | production | `cosmos` | `relayer/chains/cosmos/fee_market.go:(*CosmosProvider).DynamicFee` | 16 | 3 | 2 | PASS |
| 501 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GrantBasicAllowance` | 499 | 3 | 2 | PASS |
| 502 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).GrantBasicAllowanceWithExpiration` | 514 | 3 | 2 | PASS |
| 503 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).KeystoreCreated` | 54 | 3 | 2 | PASS |
| 504 | production | `cosmos` | `relayer/chains/cosmos/keys.go:CreateMnemonic` | 213 | 3 | 2 | PASS |
| 505 | test | `cosmos_test` | `relayer/chains/cosmos/keys_test.go:testProviderWithKeystore` | 13 | 3 | 2 | PASS |
| 506 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(CosmosProviderConfig).NewProvider` | 95 | 3 | 2 | PASS |
| 507 | production | `cosmos` | `relayer/chains/cosmos/provider.go:NewRPCClient` | 481 | 3 | 2 | PASS |
| 508 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryClientState` | 470 | 3 | 2 | PASS |
| 509 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryConsensusState` | 607 | 3 | 2 | PASS |
| 510 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryLatestHeight` | 1204 | 3 | 2 | PASS |
| 511 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryNextSeqAck` | 1125 | 3 | 2 | PASS |
| 512 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryNextSeqRecv` | 1102 | 3 | 2 | PASS |
| 513 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryPacketAcknowledgement` | 1169 | 3 | 2 | PASS |
| 514 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryPacketCommitment` | 1148 | 3 | 2 | PASS |
| 515 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryTx` | 118 | 3 | 2 | PASS |
| 516 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgPacketAck).Msg` | 164 | 3 | 2 | PASS |
| 517 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgRecvPacket).Msg` | 109 | 3 | 2 | PASS |
| 518 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgTimeout).Msg` | 43 | 3 | 2 | PASS |
| 519 | production | `stride` | `relayer/chains/cosmos/stride/messages.go:(MsgSubmitQueryResponse).ValidateBasic` | 29 | 3 | 2 | PASS |
| 520 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).ConnectionHandshakeProof` | 1027 | 3 | 2 | PASS |
| 521 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgConnectionOpenAck` | 1091 | 3 | 2 | PASS |
| 522 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgConnectionOpenTry` | 1054 | 3 | 2 | PASS |
| 523 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgSubmitMisbehaviour` | 1373 | 3 | 2 | PASS |
| 524 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgTransfer` | 796 | 3 | 2 | PASS |
| 525 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgUpdateClient` | 757 | 3 | 2 | PASS |
| 526 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).PacketAcknowledgement` | 892 | 3 | 2 | PASS |
| 527 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).PacketCommitment` | 851 | 3 | 2 | PASS |
| 528 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).QueryIBCHeader` | 1476 | 3 | 2 | PASS |
| 529 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).SubmitTxAwaitResponse` | 222 | 3 | 2 | PASS |
| 530 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).mkTxResult` | 515 | 3 | 2 | PASS |
| 531 | production | `cosmos` | `relayer/chains/cosmos/tx.go:ensureSequenceGuard` | 89 | 3 | 2 | PASS |
| 532 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).KeystoreCreated` | 50 | 3 | 2 | PASS |
| 533 | production | `penumbra` | `relayer/chains/penumbra/keys.go:CreateMnemonic` | 201 | 3 | 2 | PASS |
| 534 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).handlePacketMessage` | 26 | 3 | 2 | PASS |
| 535 | production | `penumbra` | `relayer/chains/penumbra/msg.go:typedPenumbraMsg` | 35 | 3 | 2 | PASS |
| 536 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).clientState` | 130 | 3 | 2 | PASS |
| 537 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).initializeConnectionState` | 223 | 3 | 2 | PASS |
| 538 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(latestClientState).update` | 85 | 3 | 2 | PASS |
| 539 | production | `penumbra` | `relayer/chains/penumbra/provider.go:newRPCClient` | 437 | 3 | 2 | PASS |
| 540 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryClientState` | 245 | 3 | 2 | PASS |
| 541 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryConnections` | 473 | 3 | 2 | PASS |
| 542 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryConsensusState` | 382 | 3 | 2 | PASS |
| 543 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryLatestHeight` | 789 | 3 | 2 | PASS |
| 544 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryNextSeqAck` | 710 | 3 | 2 | PASS |
| 545 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryNextSeqRecv` | 687 | 3 | 2 | PASS |
| 546 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryPacketAcknowledgement` | 754 | 3 | 2 | PASS |
| 547 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryPacketCommitment` | 733 | 3 | 2 | PASS |
| 548 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryTx` | 39 | 3 | 2 | PASS |
| 549 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgPacketAck).Msg` | 189 | 3 | 2 | PASS |
| 550 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).Msg` | 136 | 3 | 2 | PASS |
| 551 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgTimeout).Msg` | 57 | 3 | 2 | PASS |
| 552 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelCloseConfirm` | 881 | 3 | 2 | PASS |
| 553 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelOpenInit` | 697 | 3 | 2 | PASS |
| 554 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ConnectionHandshakeProof` | 1347 | 3 | 2 | PASS |
| 555 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ConnectionOpenInit` | 508 | 3 | 2 | PASS |
| 556 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).IBCHeaderAtHeight` | 1881 | 3 | 2 | PASS |
| 557 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgConnectionOpenAck` | 1407 | 3 | 2 | PASS |
| 558 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgConnectionOpenTry` | 1370 | 3 | 2 | PASS |
| 559 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgSubmitMisbehaviour` | 926 | 3 | 2 | PASS |
| 560 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgTransfer` | 998 | 3 | 2 | PASS |
| 561 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).QueryIBCHeader` | 2058 | 3 | 2 | PASS |
| 562 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).mkTxResult` | 2266 | 3 | 2 | PASS |
| 563 | production | `penumbra` | `relayer/chains/penumbra/tx.go:castClientStateToTMType` | 1985 | 3 | 2 | PASS |
| 564 | production | `relayer` | `relayer/client.go:ClientInfoFromClientState` | 578 | 3 | 2 | PASS |
| 565 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:GetEIP712BytesForMsg` | 31 | 3 | 2 | PASS |
| 566 | production | `ethermint` | `relayer/codecs/ethermint/eip712.go:validateCodecInit` | 245 | 3 | 2 | PASS |
| 567 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:unpackAny` | 346 | 3 | 2 | PASS |
| 568 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).Sign` | 116 | 3 | 2 | PASS |
| 569 | production | `relayer` | `relayer/naive-strategy.go:(*RelaySequences).Empty` | 377 | 3 | 2 | PASS |
| 570 | production | `relayer` | `relayer/path.go:(*Path).End` | 164 | 3 | 2 | PASS |
| 571 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessor).Run` | 93 | 3 | 2 | PASS |
| 572 | production | `processor` | `relayer/processor/message_processor.go:isLocalhostClient` | 120 | 3 | 2 | PASS |
| 573 | production | `processor` | `relayer/processor/path_end.go:(PathEnd).shouldRelayChannelSingle` | 61 | 3 | 2 | PASS |
| 574 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).IsRelayedChannel` | 236 | 3 | 2 | PASS |
| 575 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).IsRelevantChannel` | 263 | 3 | 2 | PASS |
| 576 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).IsRelevantClient` | 245 | 3 | 2 | PASS |
| 577 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).IsRelevantConnection` | 254 | 3 | 2 | PASS |
| 578 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).OnConnectionMessage` | 181 | 3 | 2 | PASS |
| 579 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).RelevantClientID` | 170 | 3 | 2 | PASS |
| 580 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).shouldFlush` | 144 | 3 | 2 | PASS |
| 581 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).packetMessagesToSend` | 1129 | 3 | 2 | PASS |
| 582 | production | `processor` | `relayer/processor/types.go:(ChannelPacketMessagesCache).Cache` | 440 | 3 | 2 | PASS |
| 583 | production | `processor` | `relayer/processor/types.go:(ChannelPacketMessagesCache).Retain` | 490 | 3 | 2 | PASS |
| 584 | production | `processor` | `relayer/processor/types.go:(ChannelPacketStateCache).State` | 387 | 3 | 2 | PASS |
| 585 | test | `processor_test` | `relayer/processor/types_test.go:TestIBCHeaderCachePrune` | 18 | 3 | 2 | PASS |
| 586 | production | `provider` | `relayer/provider/provider.go:(TendermintIBCHeader).TMHeader` | 556 | 3 | 2 | PASS |
| 587 | production | `relayer` | `relayer/relayMsgs.go:(*RelayMsgs).Send` | 153 | 3 | 2 | PASS |
| 588 | production | `relayer` | `relayer/strategies.go:queryChannelsOnConnection` | 311 | 3 | 2 | PASS |
| 589 | production | `relayer` | `relayer/strategies.go:relayerStartEventProcessor` | 154 | 3 | 2 | PASS |
| 590 | production | `cmd` | `cmd/chains.go:isValidURL` | 573 | 3 | 1 | PASS |
| 591 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).AccountFromKeyOrAddress` | 222 | 3 | 1 | PASS |
| 592 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProviderConfig).SignMode` | 1831 | 3 | 1 | PASS |
| 593 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgTimeout).FetchCommitResponse` | 44 | 3 | 1 | PASS |
| 594 | production | `relayer` | `relayer/pathEnd.go:OrderFromString` | 18 | 3 | 1 | PASS |
| 595 | production | `relayer` | `relayer/pathEnd.go:StringFromOrder` | 30 | 3 | 1 | PASS |
| 596 | production | `processor` | `relayer/processor/types.go:PacketInfoChannelKey` | 659 | 3 | 1 | PASS |
| 597 | production | `processor` | `relayer/processor/types_internal.go:orderFromString` | 679 | 3 | 1 | PASS |
| 598 | production | `relayer` | `relayer/strategies.go:(*Chain).chainProcessor` | 138 | 3 | 1 | PASS |
| 599 | production | `cmd` | `cmd/chains.go:cmdChainsUseRpcAddr` | 50 | 2 | 2 | PASS |
| 600 | test | `cmd_test` | `cmd/chains_test.go:TestChainsAdd_URL` | 52 | 2 | 2 | PASS |
| 601 | production | `cmd` | `cmd/root.go:withUsage` | 231 | 2 | 2 | PASS |
| 602 | test | `interchaintest_test` | `interchaintest/relayer_override_test.go:TestClientOverrideFlag` | 25 | 2 | 2 | PASS |
| 603 | production | `cosmos` | `relayer/chains/cosmos/msg.go:CosmosMsg` | 26 | 2 | 2 | PASS |
| 604 | production | `penumbra` | `relayer/chains/penumbra/msg.go:PenumbraMsg` | 25 | 2 | 2 | PASS |
| 605 | production | `relayer` | `relayer/path.go:(Paths).Get` | 34 | 2 | 2 | PASS |
| 606 | production | `relayer` | `relayer/query.go:SPrintClientExpiration` | 302 | 2 | 2 | PASS |
| 607 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).ABCIInfo` | 41 | 2 | 1 | PASS |
| 608 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).ABCIQuery` | 58 | 2 | 1 | PASS |
| 609 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).ABCIQueryWithOptions` | 71 | 2 | 1 | PASS |
| 610 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).Block` | 223 | 2 | 1 | PASS |
| 611 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BlockByHash` | 235 | 2 | 1 | PASS |
| 612 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BroadcastTxAsync` | 122 | 2 | 1 | PASS |
| 613 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BroadcastTxCommit` | 90 | 2 | 1 | PASS |
| 614 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).BroadcastTxSync` | 137 | 2 | 1 | PASS |
| 615 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).Status` | 180 | 2 | 1 | PASS |
| 616 | production | `cclient` | `cclient/cmbft_client_wrapper.go:(CometRPCClient).Tx` | 339 | 2 | 1 | PASS |
| 617 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertProofOps` | 395 | 2 | 1 | PASS |
| 618 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertResultABCIQuery` | 518 | 2 | 1 | PASS |
| 619 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).DoBroadcastTxAsync` | 105 | 2 | 1 | PASS |
| 620 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).DoBroadcastTxSync` | 120 | 2 | 1 | PASS |
| 621 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetABCIQuery` | 42 | 2 | 1 | PASS |
| 622 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetABCIQueryWithOptions` | 135 | 2 | 1 | PASS |
| 623 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetBlockResults` | 29 | 2 | 1 | PASS |
| 624 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetBlockSearch` | 74 | 2 | 1 | PASS |
| 625 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetBlockTime` | 17 | 2 | 1 | PASS |
| 626 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetCommit` | 83 | 2 | 1 | PASS |
| 627 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetStatus` | 144 | 2 | 1 | PASS |
| 628 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetTx` | 54 | 2 | 1 | PASS |
| 629 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetTxSearch` | 63 | 2 | 1 | PASS |
| 630 | production | `cclient` | `cclient/cmbft_consensus.go:(CometRPCClient).GetValidators` | 93 | 2 | 1 | PASS |
| 631 | production | `cmd` | `cmd/appstate.go:(*appState).useBackupRpcAddrs` | 323 | 2 | 1 | PASS |
| 632 | production | `cmd` | `cmd/appstate.go:(*appState).useRpcAddr` | 310 | 2 | 1 | PASS |
| 633 | production | `cmd` | `cmd/config.go:(*Config).Wrapped` | 273 | 2 | 1 | PASS |
| 634 | production | `cmd` | `cmd/config.go:(*Config).memo` | 302 | 2 | 1 | PASS |
| 635 | production | `cmd` | `cmd/config.go:(*ProviderConfigWrapper).UnmarshalJSON` | 376 | 2 | 1 | PASS |
| 636 | production | `cmd` | `cmd/config.go:(Config).MustYAML` | 469 | 2 | 1 | PASS |
| 637 | production | `cmd` | `cmd/flags.go:OverwriteConfigFlag` | 542 | 2 | 1 | PASS |
| 638 | production | `cmd` | `cmd/flags.go:addOutputFlag` | 551 | 2 | 1 | PASS |
| 639 | production | `cmd` | `cmd/flags.go:clientUnbondingPeriodFlag` | 369 | 2 | 1 | PASS |
| 640 | production | `cmd` | `cmd/flags.go:dstPortFlag` | 416 | 2 | 1 | PASS |
| 641 | production | `cmd` | `cmd/flags.go:fileFlag` | 174 | 2 | 1 | PASS |
| 642 | production | `cmd` | `cmd/flags.go:flushIntervalFlag` | 511 | 2 | 1 | PASS |
| 643 | production | `cmd` | `cmd/flags.go:forceAddFlag` | 190 | 2 | 1 | PASS |
| 644 | production | `cmd` | `cmd/flags.go:getTimeout` | 243 | 2 | 1 | PASS |
| 645 | production | `cmd` | `cmd/flags.go:heightFlag` | 82 | 2 | 1 | PASS |
| 646 | production | `cmd` | `cmd/flags.go:ibcDenomFlags` | 74 | 2 | 1 | PASS |
| 647 | production | `cmd` | `cmd/flags.go:initBlockFlag` | 496 | 2 | 1 | PASS |
| 648 | production | `cmd` | `cmd/flags.go:jsonFlag` | 166 | 2 | 1 | PASS |
| 649 | production | `cmd` | `cmd/flags.go:keyNameFlag` | 534 | 2 | 1 | PASS |
| 650 | production | `cmd` | `cmd/flags.go:memoFlag` | 526 | 2 | 1 | PASS |
| 651 | production | `cmd` | `cmd/flags.go:orderFlag` | 392 | 2 | 1 | PASS |
| 652 | production | `cmd` | `cmd/flags.go:overrideFlag` | 384 | 2 | 1 | PASS |
| 653 | production | `cmd` | `cmd/flags.go:pathFlag` | 146 | 2 | 1 | PASS |
| 654 | production | `cmd` | `cmd/flags.go:processorFlag` | 488 | 2 | 1 | PASS |
| 655 | production | `cmd` | `cmd/flags.go:retryFlag` | 304 | 2 | 1 | PASS |
| 656 | production | `cmd` | `cmd/flags.go:skipConfirm` | 130 | 2 | 1 | PASS |
| 657 | production | `cmd` | `cmd/flags.go:srcPortFlag` | 408 | 2 | 1 | PASS |
| 658 | production | `cmd` | `cmd/flags.go:strategyFlag` | 259 | 2 | 1 | PASS |
| 659 | production | `cmd` | `cmd/flags.go:testnetFlag` | 182 | 2 | 1 | PASS |
| 660 | production | `cmd` | `cmd/flags.go:timeoutFlag` | 235 | 2 | 1 | PASS |
| 661 | production | `cmd` | `cmd/flags.go:updateTimeFlags` | 312 | 2 | 1 | PASS |
| 662 | production | `cmd` | `cmd/flags.go:urlFlag` | 251 | 2 | 1 | PASS |
| 663 | production | `cmd` | `cmd/flags.go:versionFlag` | 400 | 2 | 1 | PASS |
| 664 | production | `cmd` | `cmd/flags.go:yamlFlag` | 122 | 2 | 1 | PASS |
| 665 | production | `cmd` | `cmd/paths.go:checkmark` | 120 | 2 | 1 | PASS |
| 666 | test | `cmd_test` | `cmd/start_test.go:requireDisabledDebugServer` | 339 | 2 | 1 | PASS |
| 667 | test | `cmd_test` | `cmd/start_test.go:requireDisabledMetricsServer` | 319 | 2 | 1 | PASS |
| 668 | test | `cmd_test` | `cmd/start_test.go:requireMessages` | 359 | 2 | 1 | PASS |
| 669 | test | `cmd_test` | `cmd/start_test.go:requireRunningDebugServer` | 347 | 2 | 1 | PASS |
| 670 | test | `cmd_test` | `cmd/start_test.go:requireRunningMetricsServer` | 327 | 2 | 1 | PASS |
| 671 | test | `cregistry` | `cregistry/chain_info_test.go:TestGetAllRPCEndpoints` | 10 | 2 | 1 | PASS |
| 672 | test | `interchaintest` | `interchaintest/docker.go:uniqueRelayerImageName` | 33 | 2 | 1 | PASS |
| 673 | test | `interchaintest` | `interchaintest/feegrant_test.go:Feegrant` | 1041 | 2 | 1 | PASS |
| 674 | test | `interchaintest` | `interchaintest/feegrant_test.go:TxWithRetry` | 541 | 2 | 1 | PASS |
| 675 | test | `interchaintest` | `interchaintest/feegrant_test.go:buildUserUnfunded` | 1015 | 2 | 1 | PASS |
| 676 | test | `interchaintest` | `interchaintest/feegrant_test.go:randLowerCaseLetterString` | 1033 | 2 | 1 | PASS |
| 677 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).CreateChannel` | 217 | 2 | 1 | PASS |
| 678 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).CreateClients` | 240 | 2 | 1 | PASS |
| 679 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).CreateConnections` | 232 | 2 | 1 | PASS |
| 680 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).Exec` | 304 | 2 | 1 | PASS |
| 681 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).GeneratePath` | 114 | 2 | 1 | PASS |
| 682 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).LinkPath` | 175 | 2 | 1 | PASS |
| 683 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).RestoreKey` | 106 | 2 | 1 | PASS |
| 684 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).StopRelayer` | 276 | 2 | 1 | PASS |
| 685 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).UpdateClients` | 248 | 2 | 1 | PASS |
| 686 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).UpdatePath` | 122 | 2 | 1 | PASS |
| 687 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).start` | 289 | 2 | 1 | PASS |
| 688 | test | `interchaintest` | `interchaintest/relayer.go:NewRelayer` | 38 | 2 | 1 | PASS |
| 689 | test | `interchaintest_test` | `interchaintest/tendermint_v0.37_boundary_test.go:TestScenarioTendermint37Boundary` | 16 | 2 | 1 | PASS |
| 690 | test | `relayertest` | `internal/relayertest/system.go:(*System).MustRunWithInput` | 95 | 2 | 1 | PASS |
| 691 | test | `relayertest` | `internal/relayertest/system.go:(*System).MustRunWithLogger` | 109 | 2 | 1 | PASS |
| 692 | production | `relayer` | `relayer/chain.go:(*Chain).CreateTestKey` | 140 | 2 | 1 | PASS |
| 693 | production | `relayer` | `relayer/chain.go:(*Chain).GetTimeout` | 149 | 2 | 1 | PASS |
| 694 | production | `relayer` | `relayer/chain.go:(Chains).MustGet` | 118 | 2 | 1 | PASS |
| 695 | production | `cosmos` | `relayer/chains/cosmos/account.go:(*CosmosProvider).GetAccountNumberSequence` | 67 | 2 | 1 | PASS |
| 696 | production | `cosmos` | `relayer/chains/cosmos/codec.go:MakeCodecConfig` | 95 | 2 | 1 | PASS |
| 697 | production | `cosmos` | `relayer/chains/cosmos/fee_market.go:parseTokenDenom` | 65 | 2 | 1 | PASS |
| 698 | production | `cosmos` | `relayer/chains/cosmos/grpc_query.go:GetHeightFromMetadata` | 215 | 2 | 1 | PASS |
| 699 | production | `cosmos` | `relayer/chains/cosmos/grpc_query.go:GetProveFromMetadata` | 223 | 2 | 1 | PASS |
| 700 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).AddKey` | 65 | 2 | 1 | PASS |
| 701 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).CreateKeystore` | 44 | 2 | 1 | PASS |
| 702 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).GetKeyAddress` | 204 | 2 | 1 | PASS |
| 703 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).GetKeyAddressForKey` | 236 | 2 | 1 | PASS |
| 704 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).KeyExists` | 187 | 2 | 1 | PASS |
| 705 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).RestoreKey` | 82 | 2 | 1 | PASS |
| 706 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/privkey.go:(*PrivKey).Equals` | 27 | 2 | 1 | PASS |
| 707 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/privkey.go:(*PrivKey).PubKey` | 18 | 2 | 1 | PASS |
| 708 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/pubkey.go:(*PubKey).Equals` | 12 | 2 | 1 | PASS |
| 709 | production | `cosmos` | `relayer/chains/cosmos/log.go:msgTypesField` | 137 | 2 | 1 | PASS |
| 710 | production | `cosmos` | `relayer/chains/cosmos/msg.go:(CosmosMessage).MarshalLogObject` | 57 | 2 | 1 | PASS |
| 711 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).BlockTime` | 457 | 2 | 1 | PASS |
| 712 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).MustEncodeAccAddr` | 212 | 2 | 1 | PASS |
| 713 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).Sprint` | 263 | 2 | 1 | PASS |
| 714 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).setLightProvider` | 419 | 2 | 1 | PASS |
| 715 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).updateNextAccountSequence` | 469 | 2 | 1 | PASS |
| 716 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(CosmosProviderConfig).Validate` | 83 | 2 | 1 | PASS |
| 717 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryBalance` | 284 | 2 | 1 | PASS |
| 718 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryChannelClient` | 860 | 2 | 1 | PASS |
| 719 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryChannelsPaginated` | 923 | 2 | 1 | PASS |
| 720 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryDenomHash` | 1263 | 2 | 1 | PASS |
| 721 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryDenomTrace` | 1224 | 2 | 1 | PASS |
| 722 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryPacketReceipt` | 1189 | 2 | 1 | PASS |
| 723 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryStakingParams` | 1278 | 2 | 1 | PASS |
| 724 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryStatus` | 1215 | 2 | 1 | PASS |
| 725 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryUnreceivedAcknowledgements` | 1088 | 2 | 1 | PASS |
| 726 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).QueryUnreceivedPackets` | 1002 | 2 | 1 | PASS |
| 727 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).ChannelProof` | 1181 | 2 | 1 | PASS |
| 728 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).ConnectionProof` | 1123 | 2 | 1 | PASS |
| 729 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgAcknowledgement` | 911 | 2 | 1 | PASS |
| 730 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgChannelCloseConfirm` | 1284 | 2 | 1 | PASS |
| 731 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgChannelCloseInit` | 1268 | 2 | 1 | PASS |
| 732 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgChannelOpenAck` | 1230 | 2 | 1 | PASS |
| 733 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgChannelOpenConfirm` | 1250 | 2 | 1 | PASS |
| 734 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgChannelOpenInit` | 1156 | 2 | 1 | PASS |
| 735 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgChannelOpenTry` | 1198 | 2 | 1 | PASS |
| 736 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgConnectionOpenConfirm` | 1139 | 2 | 1 | PASS |
| 737 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgConnectionOpenInit` | 1005 | 2 | 1 | PASS |
| 738 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgRecvPacket` | 872 | 2 | 1 | PASS |
| 739 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgSubmitQueryResponse` | 1354 | 2 | 1 | PASS |
| 740 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgTimeout` | 969 | 2 | 1 | PASS |
| 741 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgTimeoutOnClose` | 987 | 2 | 1 | PASS |
| 742 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgUpgradeClient` | 777 | 2 | 1 | PASS |
| 743 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).NextSeqRecv` | 952 | 2 | 1 | PASS |
| 744 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).PacketReceipt` | 932 | 2 | 1 | PASS |
| 745 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).QueryICQWithProof` | 1333 | 2 | 1 | PASS |
| 746 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).TxFactory` | 1813 | 2 | 1 | PASS |
| 747 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).sdkError` | 366 | 2 | 1 | PASS |
| 748 | test | `cosmos` | `relayer/chains/cosmos/tx_test.go:(mockTxConfig).NewTxBuilder` | 129 | 2 | 1 | PASS |
| 749 | test | `cosmos` | `relayer/chains/cosmos/tx_test.go:TestSetWithExtensionOptions` | 88 | 2 | 1 | PASS |
| 750 | test | `mock_test` | `relayer/chains/mock/mock_chain_processor_test.go:TestMockChainAndPathProcessors` | 21 | 2 | 1 | PASS |
| 751 | production | `chains` | `relayer/chains/parsing.go:(*ChannelInfo).ParseAttrs` | 320 | 2 | 1 | PASS |
| 752 | production | `chains` | `relayer/chains/parsing.go:(*ClientInfo).ParseAttrs` | 145 | 2 | 1 | PASS |
| 753 | production | `chains` | `relayer/chains/parsing.go:(*ConnectionInfo).ParseAttrs` | 358 | 2 | 1 | PASS |
| 754 | production | `chains` | `relayer/chains/parsing.go:(*PacketInfo).ParseAttrs` | 210 | 2 | 1 | PASS |
| 755 | production | `penumbra` | `relayer/chains/penumbra/grpc_query.go:GetHeightFromMetadata` | 203 | 2 | 1 | PASS |
| 756 | production | `penumbra` | `relayer/chains/penumbra/grpc_query.go:GetProveFromMetadata` | 211 | 2 | 1 | PASS |
| 757 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).AddKey` | 61 | 2 | 1 | PASS |
| 758 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).CreateKeystore` | 40 | 2 | 1 | PASS |
| 759 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).DeleteKey` | 167 | 2 | 1 | PASS |
| 760 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).GetKeyAddress` | 192 | 2 | 1 | PASS |
| 761 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).KeyExists` | 175 | 2 | 1 | PASS |
| 762 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).RestoreKey` | 78 | 2 | 1 | PASS |
| 763 | production | `penumbra` | `relayer/chains/penumbra/log.go:msgTypesField` | 127 | 2 | 1 | PASS |
| 764 | production | `penumbra` | `relayer/chains/penumbra/msg.go:(PenumbraMessage).MarshalLogObject` | 71 | 2 | 1 | PASS |
| 765 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).BlockTime` | 410 | 2 | 1 | PASS |
| 766 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).Sprint` | 224 | 2 | 1 | PASS |
| 767 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).setLightProvider` | 372 | 2 | 1 | PASS |
| 768 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(PenumbraProviderConfig).Validate` | 69 | 2 | 1 | PASS |
| 769 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryBalance` | 116 | 2 | 1 | PASS |
| 770 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryBalanceWithAddress` | 127 | 2 | 1 | PASS |
| 771 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryChannelClient` | 590 | 2 | 1 | PASS |
| 772 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryChannels` | 617 | 2 | 1 | PASS |
| 773 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryClients` | 408 | 2 | 1 | PASS |
| 774 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryConnectionChannels` | 603 | 2 | 1 | PASS |
| 775 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryDenomTrace` | 833 | 2 | 1 | PASS |
| 776 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryDenomTraces` | 846 | 2 | 1 | PASS |
| 777 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryPacketAcknowledgements` | 645 | 2 | 1 | PASS |
| 778 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryPacketCommitments` | 630 | 2 | 1 | PASS |
| 779 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryPacketReceipt` | 774 | 2 | 1 | PASS |
| 780 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryStakingParams` | 861 | 2 | 1 | PASS |
| 781 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryStatus` | 1006 | 2 | 1 | PASS |
| 782 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryUnreceivedAcknowledgements` | 673 | 2 | 1 | PASS |
| 783 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryUnreceivedPackets` | 659 | 2 | 1 | PASS |
| 784 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelCloseInit` | 861 | 2 | 1 | PASS |
| 785 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ChannelProof` | 1513 | 2 | 1 | PASS |
| 786 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).ConnectionProof` | 1459 | 2 | 1 | PASS |
| 787 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).GetIBCUpdateHeader` | 1870 | 2 | 1 | PASS |
| 788 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgAcknowledgement` | 1258 | 2 | 1 | PASS |
| 789 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgChannelCloseConfirm` | 1612 | 2 | 1 | PASS |
| 790 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgChannelCloseInit` | 1596 | 2 | 1 | PASS |
| 791 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgChannelOpenAck` | 1558 | 2 | 1 | PASS |
| 792 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgChannelOpenConfirm` | 1578 | 2 | 1 | PASS |
| 793 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgChannelOpenInit` | 1488 | 2 | 1 | PASS |
| 794 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgChannelOpenTry` | 1526 | 2 | 1 | PASS |
| 795 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgConnectionOpenConfirm` | 1471 | 2 | 1 | PASS |
| 796 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgConnectionOpenInit` | 1325 | 2 | 1 | PASS |
| 797 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgRecvPacket` | 1229 | 2 | 1 | PASS |
| 798 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgTimeout` | 1289 | 2 | 1 | PASS |
| 799 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgTimeoutOnClose` | 1307 | 2 | 1 | PASS |
| 800 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgUpgradeClient` | 908 | 2 | 1 | PASS |
| 801 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).NextSeqRecv` | 1442 | 2 | 1 | PASS |
| 802 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).PacketAcknowledgement` | 1246 | 2 | 1 | PASS |
| 803 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).PacketCommitment` | 1217 | 2 | 1 | PASS |
| 804 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).PacketReceipt` | 1276 | 2 | 1 | PASS |
| 805 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).queryTMClientState` | 1975 | 2 | 1 | PASS |
| 806 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).sdkError` | 2133 | 2 | 1 | PASS |
| 807 | production | `penumbra` | `relayer/chains/penumbra/tx.go:mustGetHeight` | 941 | 2 | 1 | PASS |
| 808 | production | `relayer` | `relayer/channel.go:(*Chain).CloseChannel` | 107 | 2 | 1 | PASS |
| 809 | production | `relayer` | `relayer/client.go:MustGetHeight` | 499 | 2 | 1 | PASS |
| 810 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:extractMsgTypes` | 85 | 2 | 1 | PASS |
| 811 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(*PrivKey).UnmarshalAmino` | 92 | 2 | 1 | PASS |
| 812 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(*PubKey).UnmarshalAmino` | 183 | 2 | 1 | PASS |
| 813 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).Equals` | 77 | 2 | 1 | PASS |
| 814 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).PubKey` | 65 | 2 | 1 | PASS |
| 815 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).Address` | 145 | 2 | 1 | PASS |
| 816 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).Equals` | 173 | 2 | 1 | PASS |
| 817 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).VerifySignature` | 210 | 2 | 1 | PASS |
| 818 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).verifySignatureAsEIP712` | 217 | 2 | 1 | PASS |
| 819 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).verifySignatureECDSA` | 227 | 2 | 1 | PASS |
| 820 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:GenerateKey` | 44 | 2 | 1 | PASS |
| 821 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PrivKey).Equals` | 71 | 2 | 1 | PASS |
| 822 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PrivKey).UnmarshalAmino` | 86 | 2 | 1 | PASS |
| 823 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PubKey).Equals` | 159 | 2 | 1 | PASS |
| 824 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PubKey).UnmarshalAmino` | 169 | 2 | 1 | PASS |
| 825 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PrivKey).ToECDSA` | 116 | 2 | 1 | PASS |
| 826 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PubKey).Address` | 134 | 2 | 1 | PASS |
| 827 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PubKey).VerifySignature` | 193 | 2 | 1 | PASS |
| 828 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:GenerateKey` | 45 | 2 | 1 | PASS |
| 829 | production | `injective` | `relayer/codecs/injective/params.go:(Params).Validate` | 65 | 2 | 1 | PASS |
| 830 | production | `injective` | `relayer/codecs/injective/params.go:validateBool` | 82 | 2 | 1 | PASS |
| 831 | production | `injective` | `relayer/codecs/injective/params.go:validateEVMDenom` | 73 | 2 | 1 | PASS |
| 832 | production | `relayer` | `relayer/connection.go:(*Chain).CreateOpenConnections` | 15 | 2 | 1 | PASS |
| 833 | production | `relayer` | `relayer/ics24.go:(*Chain).SetPath` | 29 | 2 | 1 | PASS |
| 834 | production | `relayer` | `relayer/path.go:(*Path).MustYAML` | 62 | 2 | 1 | PASS |
| 835 | production | `relayer` | `relayer/path.go:(Paths).Add` | 53 | 2 | 1 | PASS |
| 836 | production | `relayer` | `relayer/path.go:(Paths).MustGet` | 44 | 2 | 1 | PASS |
| 837 | production | `relayer` | `relayer/path.go:(Paths).MustYAML` | 25 | 2 | 1 | PASS |
| 838 | production | `relayer` | `relayer/path.go:checkmark` | 309 | 2 | 1 | PASS |
| 839 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).assembleMessage` | 232 | 2 | 1 | PASS |
| 840 | production | `processor` | `relayer/processor/message_processor.go:(*messageProcessor).sendClientUpdate` | 373 | 2 | 1 | PASS |
| 841 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).ProcessBacklogIfReady` | 274 | 2 | 1 | PASS |
| 842 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).SetMessageLifecycle` | 136 | 2 | 1 | PASS |
| 843 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).handleFlush` | 299 | 2 | 1 | PASS |
| 844 | production | `processor` | `relayer/processor/path_processor.go:NewPathProcessor` | 98 | 2 | 1 | PASS |
| 845 | production | `processor` | `relayer/processor/types.go:(ChannelMessageCache).Merge` | 581 | 2 | 1 | PASS |
| 846 | production | `processor` | `relayer/processor/types.go:(ChannelMessagesCache).Retain` | 563 | 2 | 1 | PASS |
| 847 | production | `processor` | `relayer/processor/types.go:(ChannelPacketStateCache).Prune` | 402 | 2 | 1 | PASS |
| 848 | production | `processor` | `relayer/processor/types.go:(ClientICQMessageCache).Merge` | 609 | 2 | 1 | PASS |
| 849 | production | `processor` | `relayer/processor/types.go:(ClientICQMessagesCache).Retain` | 588 | 2 | 1 | PASS |
| 850 | production | `processor` | `relayer/processor/types.go:(ConnectionMessageCache).Merge` | 546 | 2 | 1 | PASS |
| 851 | production | `processor` | `relayer/processor/types.go:(ConnectionMessagesCache).Retain` | 528 | 2 | 1 | PASS |
| 852 | production | `processor` | `relayer/processor/types.go:(IBCHeaderCache).Merge` | 635 | 2 | 1 | PASS |
| 853 | production | `processor` | `relayer/processor/types.go:(PacketSequenceCache).Merge` | 511 | 2 | 1 | PASS |
| 854 | production | `processor` | `relayer/processor/types_internal.go:(*clientICQProcessingCache).deleteMessages` | 524 | 2 | 1 | PASS |
| 855 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessage).assemble` | 298 | 2 | 1 | PASS |
| 856 | production | `provider` | `relayer/provider/provider.go:(RelayerEvent).MarshalLogObject` | 188 | 2 | 1 | PASS |
| 857 | production | `provider` | `relayer/provider/provider.go:(loggableEvents).MarshalLogArray` | 197 | 2 | 1 | PASS |
| 858 | production | `relayer` | `relayer/relayMsgs.go:(SendMsgsResult).SuccessfullySent` | 116 | 2 | 1 | PASS |
| 859 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).RegisterInterfaces` | 30 | 1 | 1 | PASS |
| 860 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).RegisterLegacyAminoCodec` | 26 | 1 | 1 | PASS |
| 861 | production | `cclient` | `cclient/cmbft_client_wrapper.go:NewCometRPCClient` | 37 | 1 | 0 | PASS |
| 862 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertBlockID` | 477 | 1 | 0 | PASS |
| 863 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertHeader` | 453 | 1 | 0 | PASS |
| 864 | production | `cclient` | `cclient/cmbft_client_wrapper.go:convertResultTx` | 539 | 1 | 0 | PASS |
| 865 | production | `cclient` | `cclient/consensus.go:(ABCIQueryResponse).ValueCleaned` | 68 | 1 | 0 | PASS |
| 866 | production | `cmd` | `cmd/appstate.go:(*appState).configPath` | 50 | 1 | 0 | PASS |
| 867 | production | `cmd` | `cmd/chains.go:chainsAddDirCmd` | 401 | 1 | 0 | PASS |
| 868 | production | `cmd` | `cmd/chains.go:chainsCmd` | 27 | 1 | 0 | PASS |
| 869 | production | `cmd` | `cmd/chains.go:cmdChainsConfigure` | 204 | 1 | 0 | PASS |
| 870 | test | `cmd_test` | `cmd/chains_test.go:TestChainsAdd_Delete` | 93 | 1 | 0 | PASS |
| 871 | test | `cmd_test` | `cmd/chains_test.go:TestChainsAdd_File` | 30 | 1 | 0 | PASS |
| 872 | test | `cmd_test` | `cmd/chains_test.go:TestChainsList_Empty` | 16 | 1 | 0 | PASS |
| 873 | production | `cmd` | `cmd/config.go:(*Config).DeleteChain` | 584 | 1 | 0 | PASS |
| 874 | production | `cmd` | `cmd/config.go:DefaultConfig` | 481 | 1 | 0 | PASS |
| 875 | production | `cmd` | `cmd/config.go:configCmd` | 42 | 1 | 0 | PASS |
| 876 | production | `cmd` | `cmd/config.go:defaultConfigYAML` | 477 | 1 | 0 | PASS |
| 877 | production | `cmd` | `cmd/config.go:newDefaultGlobalConfig` | 503 | 1 | 0 | PASS |
| 878 | production | `cmd` | `cmd/errors.go:errChainNotFound` | 16 | 1 | 0 | PASS |
| 879 | production | `cmd` | `cmd/errors.go:errKeyDoesntExist` | 12 | 1 | 0 | PASS |
| 880 | production | `cmd` | `cmd/errors.go:errKeyExists` | 8 | 1 | 0 | PASS |
| 881 | production | `cmd` | `cmd/errors.go:invalidRpcAddr` | 20 | 1 | 0 | PASS |
| 882 | production | `cmd` | `cmd/feegrant.go:feegrantConfigureBaseCmd` | 15 | 1 | 0 | PASS |
| 883 | production | `cmd` | `cmd/flags.go:chainsAddFlags` | 138 | 1 | 0 | PASS |
| 884 | production | `cmd` | `cmd/flags.go:channelParameterFlags` | 365 | 1 | 0 | PASS |
| 885 | test | `cmd` | `cmd/flags_test.go:TestFlagEqualityAgainstSDK` | 13 | 1 | 0 | PASS |
| 886 | production | `cmd` | `cmd/keys.go:addressCmd` | 446 | 1 | 0 | PASS |
| 887 | production | `cmd` | `cmd/keys.go:keysCmd` | 40 | 1 | 0 | PASS |
| 888 | production | `cmd` | `cmd/keys.go:keysShowCmd` | 429 | 1 | 0 | PASS |
| 889 | production | `cmd` | `cmd/keys.go:keysUseCmd` | 60 | 1 | 0 | PASS |
| 890 | test | `cmd_test` | `cmd/keys_test.go:TestKeysDefaultCoinType` | 125 | 1 | 0 | PASS |
| 891 | test | `cmd_test` | `cmd/keys_test.go:TestKeysExport` | 79 | 1 | 0 | PASS |
| 892 | test | `cmd_test` | `cmd/keys_test.go:TestKeysList_Empty` | 17 | 1 | 0 | PASS |
| 893 | test | `cmd_test` | `cmd/keys_test.go:TestKeysRestoreAll_Delete` | 190 | 1 | 0 | PASS |
| 894 | test | `cmd_test` | `cmd/keys_test.go:TestKeysRestore_Delete` | 38 | 1 | 0 | PASS |
| 895 | production | `cmd` | `cmd/paths.go:pathsAddDirCmd` | 216 | 1 | 0 | PASS |
| 896 | production | `cmd` | `cmd/paths.go:pathsCmd` | 18 | 1 | 0 | PASS |
| 897 | production | `cmd` | `cmd/paths.go:printPath` | 115 | 1 | 0 | PASS |
| 898 | production | `cmd` | `cmd/query.go:feegrantQueryCmd` | 64 | 1 | 0 | PASS |
| 899 | production | `cmd` | `cmd/query.go:queryCmd` | 24 | 1 | 0 | PASS |
| 900 | production | `cmd` | `cmd/root.go:lineBreakCommand` | 225 | 1 | 0 | PASS |
| 901 | production | `cmd` | `cmd/root.go:readLine` | 218 | 1 | 0 | PASS |
| 902 | test | `cmd_test` | `cmd/start_test.go:TestMissingDebugListenAddr` | 306 | 1 | 0 | PASS |
| 903 | test | `cmd_test` | `cmd/start_test.go:TestMissingMetricsListenAddr` | 147 | 1 | 0 | PASS |
| 904 | test | `cmd_test` | `cmd/start_test.go:requireMessage` | 365 | 1 | 0 | PASS |
| 905 | test | `cmd_test` | `cmd/start_test.go:setupLogger` | 369 | 1 | 0 | PASS |
| 906 | test | `cmd_test` | `cmd/start_test.go:setupRelayer` | 375 | 1 | 0 | PASS |
| 907 | test | `cmd_test` | `cmd/start_test.go:updateConfig` | 391 | 1 | 0 | PASS |
| 908 | production | `cmd` | `cmd/tx.go:relayAcksCmd` | 1017 | 1 | 0 | PASS |
| 909 | production | `cmd` | `cmd/tx.go:relayMsgsCmd` | 995 | 1 | 0 | PASS |
| 910 | production | `cmd` | `cmd/tx.go:transactionCmd` | 25 | 1 | 0 | PASS |
| 911 | production | `cregistry` | `cregistry/chain_info.go:NewChainInfo` | 101 | 1 | 0 | PASS |
| 912 | test | `cregistry` | `cregistry/chain_info_test.go:ChainInfoWithRPCEndpoint` | 68 | 1 | 0 | PASS |
| 913 | production | `cregistry` | `cregistry/chain_registry.go:DefaultChainRegistry` | 17 | 1 | 0 | PASS |
| 914 | production | `cregistry` | `cregistry/cosmos_github_registry.go:(CosmosGithubRegistry).SourceLink` | 88 | 1 | 0 | PASS |
| 915 | production | `cregistry` | `cregistry/cosmos_github_registry.go:NewCosmosGithubRegistry` | 23 | 1 | 0 | PASS |
| 916 | test | `interchaintest` | `interchaintest/acc_cache_test.go:TestAccCacheBugfix` | 15 | 1 | 0 | PASS |
| 917 | test | `interchaintest` | `interchaintest/docker.go:BuildRelayerImage` | 40 | 1 | 0 | PASS |
| 918 | test | `interchaintest` | `interchaintest/docker.go:destroyRelayerImage` | 67 | 1 | 0 | PASS |
| 919 | test | `interchaintest` | `interchaintest/feegrant_test.go:assertTransactionIsValid` | 1073 | 1 | 0 | PASS |
| 920 | test | `interchaintest_test` | `interchaintest/ibc_test.go:TestRelayerDockerEventProcessor` | 64 | 1 | 0 | PASS |
| 921 | test | `interchaintest_test` | `interchaintest/ibc_test.go:TestRelayerDockerLegacyProcessor` | 82 | 1 | 0 | PASS |
| 922 | test | `interchaintest_test` | `interchaintest/ibc_test.go:TestRelayerEventProcessor` | 100 | 1 | 0 | PASS |
| 923 | test | `interchaintest_test` | `interchaintest/ibc_test.go:TestRelayerInProcess` | 56 | 1 | 0 | PASS |
| 924 | test | `interchaintest_test` | `interchaintest/ibc_test.go:TestRelayerLegacyProcessor` | 112 | 1 | 0 | PASS |
| 925 | test | `interchaintest_test` | `interchaintest/ibc_test.go:interchaintestConformance` | 24 | 1 | 0 | PASS |
| 926 | test | `interchaintest_test` | `interchaintest/interchain_accounts_test.go:parseInterchainAccountField` | 340 | 1 | 0 | PASS |
| 927 | test | `interchaintest_test` | `interchaintest/localhost_client_test.go:DefaultEncoding` | 44 | 1 | 0 | PASS |
| 928 | test | `interchaintest_test` | `interchaintest/misbehaviour_test.go:assertTransactionIsValid` | 232 | 1 | 0 | PASS |
| 929 | test | `interchaintest_test` | `interchaintest/misbehaviour_test.go:defaultEncoding` | 339 | 1 | 0 | PASS |
| 930 | test | `interchaintest_test` | `interchaintest/misbehaviour_test.go:queryHeaderAtHeight` | 243 | 1 | 0 | PASS |
| 931 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).AddChainConfiguration` | 67 | 1 | 0 | PASS |
| 932 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).PauseRelayer` | 364 | 1 | 0 | PASS |
| 933 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).ResumeRelayer` | 369 | 1 | 0 | PASS |
| 934 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).SetClientContractHash` | 359 | 1 | 0 | PASS |
| 935 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).Sys` | 59 | 1 | 0 | PASS |
| 936 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).UseDockerNetwork` | 302 | 1 | 0 | PASS |
| 937 | test | `interchaintest` | `interchaintest/relayer.go:(*Relayer).log` | 63 | 1 | 0 | PASS |
| 938 | test | `interchaintest` | `interchaintest/relayer_factory.go:(RelayerFactory).Build` | 31 | 1 | 0 | PASS |
| 939 | test | `interchaintest` | `interchaintest/relayer_factory.go:(RelayerFactory).Capabilities` | 36 | 1 | 0 | PASS |
| 940 | test | `interchaintest` | `interchaintest/relayer_factory.go:(RelayerFactory).Name` | 41 | 1 | 0 | PASS |
| 941 | test | `interchaintest` | `interchaintest/relayer_factory.go:NewRelayerFactory` | 24 | 1 | 0 | PASS |
| 942 | test | `stride_test` | `interchaintest/stride/setup_test.go:StrideEncoding` | 112 | 1 | 0 | PASS |
| 943 | production | `relaydebug` | `internal/relaydebug/debugserver.go:StartDebugServer` | 18 | 1 | 0 | PASS |
| 944 | production | `relayermetrics` | `internal/relayermetrics/metricsserver.go:StartMetricsServer` | 19 | 1 | 0 | PASS |
| 945 | test | `relayertest` | `internal/relayertest/system.go:(*System).MustAddChain` | 123 | 1 | 0 | PASS |
| 946 | test | `relayertest` | `internal/relayertest/system.go:(*System).MustGetConfig` | 143 | 1 | 0 | PASS |
| 947 | test | `relayertest` | `internal/relayertest/system.go:(*System).MustRun` | 88 | 1 | 0 | PASS |
| 948 | test | `relayertest` | `internal/relayertest/system.go:(*System).Run` | 51 | 1 | 0 | PASS |
| 949 | test | `relayertest` | `internal/relayertest/system.go:(*System).RunC` | 56 | 1 | 0 | PASS |
| 950 | test | `relayertest` | `internal/relayertest/system.go:(*System).RunWithInput` | 61 | 1 | 0 | PASS |
| 951 | test | `relayertest` | `internal/relayertest/system.go:(*System).RunWithInputC` | 68 | 1 | 0 | PASS |
| 952 | test | `relayertest` | `internal/relayertest/system.go:(*System).WriteConfig` | 155 | 1 | 0 | PASS |
| 953 | test | `relayertest` | `internal/relayertest/system.go:NewSystem` | 32 | 1 | 0 | PASS |
| 954 | production | `main` | `main.go:init` | 12 | 1 | 0 | PASS |
| 955 | production | `main` | `main.go:main` | 8 | 1 | 0 | PASS |
| 956 | production | `relayer` | `relayer/chain.go:(*Chain).ChainID` | 80 | 1 | 0 | PASS |
| 957 | production | `relayer` | `relayer/chain.go:(*Chain).ClientID` | 88 | 1 | 0 | PASS |
| 958 | production | `relayer` | `relayer/chain.go:(*Chain).ConnectionID` | 84 | 1 | 0 | PASS |
| 959 | production | `relayer` | `relayer/chain.go:(*Chain).GetSelfVersion` | 93 | 1 | 0 | PASS |
| 960 | production | `relayer` | `relayer/chain.go:(*Chain).GetTrustingPeriod` | 98 | 1 | 0 | PASS |
| 961 | production | `relayer` | `relayer/chain.go:(*Chain).String` | 102 | 1 | 0 | PASS |
| 962 | production | `relayer` | `relayer/chain.go:NewChain` | 72 | 1 | 0 | PASS |
| 963 | production | `cosmos` | `relayer/chains/cosmos/account.go:(*CosmosProvider).EnsureExists` | 60 | 1 | 0 | PASS |
| 964 | production | `cosmos` | `relayer/chains/cosmos/account.go:(*CosmosProvider).GetAccount` | 20 | 1 | 0 | PASS |
| 965 | production | `cosmos` | `relayer/chains/cosmos/bech32_hack.go:(*CosmosProvider).SetSDKContext` | 16 | 1 | 0 | PASS |
| 966 | production | `cosmos` | `relayer/chains/cosmos/bech32_hack.go:SetSDKConfigContext` | 21 | 1 | 0 | PASS |
| 967 | production | `cosmos` | `relayer/chains/cosmos/broadcast.go:(_err).Error` | 13 | 1 | 0 | PASS |
| 968 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).CurrentBlockHeight` | 556 | 1 | 0 | PASS |
| 969 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).Provider` | 121 | 1 | 0 | PASS |
| 970 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).SetPathProcessors` | 127 | 1 | 0 | PASS |
| 971 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).latestHeightWithRetry` | 133 | 1 | 0 | PASS |
| 972 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:(*CosmosChainProcessor).nodeStatusWithRetry` | 152 | 1 | 0 | PASS |
| 973 | production | `cosmos` | `relayer/chains/cosmos/cosmos_chain_processor.go:NewCosmosChainProcessor` | 59 | 1 | 0 | PASS |
| 974 | test | `cosmos` | `relayer/chains/cosmos/fee_market_test.go:TestParseDenom` | 57 | 1 | 0 | PASS |
| 975 | test | `cosmos` | `relayer/chains/cosmos/fee_market_test.go:TestQueryBaseFee` | 42 | 1 | 0 | PASS |
| 976 | production | `cosmos` | `relayer/chains/cosmos/feegrant.go:(*CosmosProvider).ConfigureFeegrants` | 136 | 1 | 0 | PASS |
| 977 | production | `cosmos` | `relayer/chains/cosmos/grpc_query.go:(*CosmosProvider).NewStream` | 89 | 1 | 0 | PASS |
| 978 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).DecodeBech32AccAddr` | 232 | 1 | 0 | PASS |
| 979 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).DeleteKey` | 182 | 1 | 0 | PASS |
| 980 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).EncodeBech32AccAddr` | 228 | 1 | 0 | PASS |
| 981 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).ExportPrivKeyArmor` | 199 | 1 | 0 | PASS |
| 982 | production | `cosmos` | `relayer/chains/cosmos/keys.go:(*CosmosProvider).UseKey` | 75 | 1 | 0 | PASS |
| 983 | production | `cosmos` | `relayer/chains/cosmos/keys.go:KeyringAlgoOptions` | 36 | 1 | 0 | PASS |
| 984 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/algo.go:(sr25519Algo).Generate` | 39 | 1 | 0 | PASS |
| 985 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/algo.go:(sr25519Algo).Name` | 16 | 1 | 0 | PASS |
| 986 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/privkey.go:(*PrivKey).ProtoMessage` | 35 | 1 | 0 | PASS |
| 987 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/privkey.go:(*PrivKey).Reset` | 37 | 1 | 0 | PASS |
| 988 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/privkey.go:(*PrivKey).String` | 41 | 1 | 0 | PASS |
| 989 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/privkey.go:GenPrivKey` | 45 | 1 | 0 | PASS |
| 990 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/pubkey.go:(*PubKey).Address` | 20 | 1 | 0 | PASS |
| 991 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/pubkey.go:(PubKey).Bytes` | 24 | 1 | 0 | PASS |
| 992 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/pubkey.go:(PubKey).String` | 28 | 1 | 0 | PASS |
| 993 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/pubkey.go:(PubKey).Type` | 32 | 1 | 0 | PASS |
| 994 | production | `sr25519` | `relayer/chains/cosmos/keys/sr25519/pubkey.go:(PubKey).VerifySignature` | 36 | 1 | 0 | PASS |
| 995 | test | `cosmos_test` | `relayer/chains/cosmos/keys_test.go:TestKeyRestore` | 35 | 1 | 0 | PASS |
| 996 | test | `cosmos_test` | `relayer/chains/cosmos/keys_test.go:TestKeyRestoreEth` | 53 | 1 | 0 | PASS |
| 997 | test | `cosmos_test` | `relayer/chains/cosmos/keys_test.go:TestKeyRestoreInj` | 71 | 1 | 0 | PASS |
| 998 | test | `cosmos_test` | `relayer/chains/cosmos/keys_test.go:TestKeyRestoreSr25519` | 89 | 1 | 0 | PASS |
| 999 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).handleClientICQMessage` | 141 | 1 | 0 | PASS |
| 1000 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).handleClientMessage` | 136 | 1 | 0 | PASS |
| 1001 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).logChannelMessage` | 177 | 1 | 0 | PASS |
| 1002 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).logChannelOpenMessage` | 187 | 1 | 0 | PASS |
| 1003 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).logClientICQMessage` | 206 | 1 | 0 | PASS |
| 1004 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).logConnectionMessage` | 197 | 1 | 0 | PASS |
| 1005 | production | `cosmos` | `relayer/chains/cosmos/message_handlers.go:(*CosmosChainProcessor).logObservedIBCMessage` | 150 | 1 | 0 | PASS |
| 1006 | test | `cosmos` | `relayer/chains/cosmos/message_handlers_test.go:TestChannelStateCache` | 98 | 1 | 0 | PASS |
| 1007 | test | `cosmos` | `relayer/chains/cosmos/message_handlers_test.go:TestConnectionStateCache` | 14 | 1 | 0 | PASS |
| 1008 | production | `module` | `relayer/chains/cosmos/module/app_module.go:(AppModuleBasic).GetQueryCmd` | 44 | 1 | 0 | PASS |
| 1009 | production | `module` | `relayer/chains/cosmos/module/app_module.go:(AppModuleBasic).GetTxCmd` | 39 | 1 | 0 | PASS |
| 1010 | production | `module` | `relayer/chains/cosmos/module/app_module.go:(AppModuleBasic).Name` | 19 | 1 | 0 | PASS |
| 1011 | production | `module` | `relayer/chains/cosmos/module/app_module.go:(AppModuleBasic).RegisterGRPCGatewayRoutes` | 34 | 1 | 0 | PASS |
| 1012 | production | `module` | `relayer/chains/cosmos/module/app_module.go:(AppModuleBasic).RegisterInterfaces` | 27 | 1 | 0 | PASS |
| 1013 | production | `module` | `relayer/chains/cosmos/module/app_module.go:(AppModuleBasic).RegisterLegacyAminoCodec` | 24 | 1 | 0 | PASS |
| 1014 | production | `cosmos` | `relayer/chains/cosmos/msg.go:(CosmosMessage).MsgBytes` | 52 | 1 | 0 | PASS |
| 1015 | production | `cosmos` | `relayer/chains/cosmos/msg.go:(CosmosMessage).Type` | 48 | 1 | 0 | PASS |
| 1016 | production | `cosmos` | `relayer/chains/cosmos/msg.go:NewCosmosMessage` | 19 | 1 | 0 | PASS |
| 1017 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).ChainId` | 167 | 1 | 0 | PASS |
| 1018 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).ChainName` | 171 | 1 | 0 | PASS |
| 1019 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).CommitmentPrefix` | 188 | 1 | 0 | PASS |
| 1020 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).Key` | 179 | 1 | 0 | PASS |
| 1021 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).ProviderConfig` | 163 | 1 | 0 | PASS |
| 1022 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).SetBackupRpcAddrs` | 280 | 1 | 0 | PASS |
| 1023 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).SetMetrics` | 465 | 1 | 0 | PASS |
| 1024 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).SetRpcAddr` | 273 | 1 | 0 | PASS |
| 1025 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).Timeout` | 183 | 1 | 0 | PASS |
| 1026 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(*CosmosProvider).Type` | 175 | 1 | 0 | PASS |
| 1027 | production | `cosmos` | `relayer/chains/cosmos/provider.go:(CosmosProviderConfig).BroadcastMode` | 90 | 1 | 0 | PASS |
| 1028 | production | `cosmos` | `relayer/chains/cosmos/provider.go:keysDir` | 476 | 1 | 0 | PASS |
| 1029 | production | `cosmos` | `relayer/chains/cosmos/query.go:(*CosmosProvider).GetQueryContext` | 275 | 1 | 0 | PASS |
| 1030 | production | `cosmos` | `relayer/chains/cosmos/query.go:DefaultPageRequest` | 1286 | 1 | 0 | PASS |
| 1031 | production | `cosmos` | `relayer/chains/cosmos/query.go:sendPacketQuery` | 1015 | 1 | 0 | PASS |
| 1032 | production | `cosmos` | `relayer/chains/cosmos/query.go:writeAcknowledgementQuery` | 1023 | 1 | 0 | PASS |
| 1033 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgPacketAck).Data` | 150 | 1 | 0 | PASS |
| 1034 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgPacketAck).Seq` | 153 | 1 | 0 | PASS |
| 1035 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgPacketAck).Timeout` | 156 | 1 | 0 | PASS |
| 1036 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgPacketAck).TimeoutStamp` | 160 | 1 | 0 | PASS |
| 1037 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgRecvPacket).Data` | 93 | 1 | 0 | PASS |
| 1038 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgRecvPacket).Seq` | 97 | 1 | 0 | PASS |
| 1039 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgRecvPacket).Timeout` | 101 | 1 | 0 | PASS |
| 1040 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgRecvPacket).TimeoutStamp` | 105 | 1 | 0 | PASS |
| 1041 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgRecvPacket).timeoutPacket` | 82 | 1 | 0 | PASS |
| 1042 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgTimeout).Data` | 27 | 1 | 0 | PASS |
| 1043 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgTimeout).Seq` | 31 | 1 | 0 | PASS |
| 1044 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgTimeout).Timeout` | 35 | 1 | 0 | PASS |
| 1045 | production | `cosmos` | `relayer/chains/cosmos/relayer_packets.go:(relayMsgTimeout).TimeoutStamp` | 39 | 1 | 0 | PASS |
| 1046 | production | `stride` | `relayer/chains/cosmos/stride/codec.go:RegisterCometTypes` | 30 | 1 | 0 | PASS |
| 1047 | production | `stride` | `relayer/chains/cosmos/stride/codec.go:RegisterInterfaces` | 24 | 1 | 0 | PASS |
| 1048 | production | `stride` | `relayer/chains/cosmos/stride/codec.go:RegisterLegacyAminoCodec` | 20 | 1 | 0 | PASS |
| 1049 | production | `stride` | `relayer/chains/cosmos/stride/codec.go:init` | 38 | 1 | 0 | PASS |
| 1050 | production | `stride` | `relayer/chains/cosmos/stride/messages.go:(MsgSubmitQueryResponse).GetSignBytes` | 44 | 1 | 0 | PASS |
| 1051 | production | `stride` | `relayer/chains/cosmos/stride/messages.go:(MsgSubmitQueryResponse).GetSigners` | 49 | 1 | 0 | PASS |
| 1052 | production | `stride` | `relayer/chains/cosmos/stride/messages.go:(MsgSubmitQueryResponse).Route` | 23 | 1 | 0 | PASS |
| 1053 | production | `stride` | `relayer/chains/cosmos/stride/messages.go:(MsgSubmitQueryResponse).Type` | 26 | 1 | 0 | PASS |
| 1054 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).DefaultGenesis` | 34 | 1 | 0 | PASS |
| 1055 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).GetQueryCmd` | 48 | 1 | 0 | PASS |
| 1056 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).GetTxCmd` | 44 | 1 | 0 | PASS |
| 1057 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).Name` | 22 | 1 | 0 | PASS |
| 1058 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).RegisterGRPCGatewayRoutes` | 42 | 1 | 0 | PASS |
| 1059 | production | `stride` | `relayer/chains/cosmos/stride/module.go:(AppModuleBasic).ValidateGenesis` | 38 | 1 | 0 | PASS |
| 1060 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).MsgRegisterCounterpartyPayee` | 1648 | 1 | 0 | PASS |
| 1061 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).NewClientState` | 1595 | 1 | 0 | PASS |
| 1062 | production | `cosmos` | `relayer/chains/cosmos/tx.go:(*CosmosProvider).SendMessage` | 82 | 1 | 0 | PASS |
| 1063 | test | `cosmos` | `relayer/chains/cosmos/tx_test.go:(*mockTxBuilder).SetExtensionOptions` | 143 | 1 | 0 | PASS |
| 1064 | test | `cosmos` | `relayer/chains/cosmos/tx_test.go:makeMockTxConfig` | 153 | 1 | 0 | PASS |
| 1065 | test | `cosmos` | `relayer/chains/cosmos/tx_test.go:makeTxConfig` | 147 | 1 | 0 | PASS |
| 1066 | production | `mock` | `relayer/chains/mock/message_handlers.go:handleMsgAcknowledgement` | 80 | 1 | 0 | PASS |
| 1067 | production | `mock` | `relayer/chains/mock/message_handlers.go:handleMsgRecvPacket` | 54 | 1 | 0 | PASS |
| 1068 | production | `mock` | `relayer/chains/mock/message_handlers.go:handleMsgTransfer` | 27 | 1 | 0 | PASS |
| 1069 | production | `mock` | `relayer/chains/mock/mock_chain_processor.go:(*MockChainProcessor).Provider` | 67 | 1 | 0 | PASS |
| 1070 | production | `mock` | `relayer/chains/mock/mock_chain_processor.go:(*MockChainProcessor).SetPathProcessors` | 62 | 1 | 0 | PASS |
| 1071 | production | `mock` | `relayer/chains/mock/mock_chain_processor.go:NewMockChainProcessor` | 43 | 1 | 0 | PASS |
| 1072 | production | `chains` | `relayer/chains/parsing.go:(*ChannelInfo).MarshalLogObject` | 312 | 1 | 0 | PASS |
| 1073 | production | `chains` | `relayer/chains/parsing.go:(*ClientICQInfo).MarshalLogObject` | 387 | 1 | 0 | PASS |
| 1074 | production | `chains` | `relayer/chains/parsing.go:(*ClientInfo).MarshalLogObject` | 138 | 1 | 0 | PASS |
| 1075 | production | `chains` | `relayer/chains/parsing.go:(*ConnectionInfo).MarshalLogObject` | 350 | 1 | 0 | PASS |
| 1076 | production | `chains` | `relayer/chains/parsing.go:(*PacketInfo).MarshalLogObject` | 200 | 1 | 0 | PASS |
| 1077 | production | `chains` | `relayer/chains/parsing.go:(ClientInfo).ClientState` | 129 | 1 | 0 | PASS |
| 1078 | production | `chains` | `relayer/chains/parsing.go:NewClientInfo` | 119 | 1 | 0 | PASS |
| 1079 | production | `penumbra` | `relayer/chains/penumbra/codec.go:makeCodecConfig` | 88 | 1 | 0 | PASS |
| 1080 | production | `penumbra` | `relayer/chains/penumbra/grpc_query.go:(*PenumbraProvider).NewStream` | 90 | 1 | 0 | PASS |
| 1081 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).EncodeBech32AccAddr` | 216 | 1 | 0 | PASS |
| 1082 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).ExportPrivKeyArmor` | 187 | 1 | 0 | PASS |
| 1083 | production | `penumbra` | `relayer/chains/penumbra/keys.go:(*PenumbraProvider).UseKey` | 71 | 1 | 0 | PASS |
| 1084 | production | `penumbra` | `relayer/chains/penumbra/keys.go:KeyringAlgoOptions` | 32 | 1 | 0 | PASS |
| 1085 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).handleClientMessage` | 120 | 1 | 0 | PASS |
| 1086 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).logChannelMessage` | 152 | 1 | 0 | PASS |
| 1087 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).logConnectionMessage` | 162 | 1 | 0 | PASS |
| 1088 | production | `penumbra` | `relayer/chains/penumbra/message_handlers.go:(*PenumbraChainProcessor).logObservedIBCMessage` | 125 | 1 | 0 | PASS |
| 1089 | production | `penumbra` | `relayer/chains/penumbra/msg.go:(PenumbraMessage).MsgBytes` | 66 | 1 | 0 | PASS |
| 1090 | production | `penumbra` | `relayer/chains/penumbra/msg.go:(PenumbraMessage).Type` | 62 | 1 | 0 | PASS |
| 1091 | production | `penumbra` | `relayer/chains/penumbra/msg.go:NewPenumbraMessage` | 19 | 1 | 0 | PASS |
| 1092 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).Provider` | 99 | 1 | 0 | PASS |
| 1093 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).SetPathProcessors` | 105 | 1 | 0 | PASS |
| 1094 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:(*PenumbraChainProcessor).latestHeightWithRetry` | 111 | 1 | 0 | PASS |
| 1095 | production | `penumbra` | `relayer/chains/penumbra/penumbra_chain_processor.go:NewPenumbraChainProcessor` | 53 | 1 | 0 | PASS |
| 1096 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).ChainId` | 157 | 1 | 0 | PASS |
| 1097 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).ChainName` | 161 | 1 | 0 | PASS |
| 1098 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).CommitmentPrefix` | 177 | 1 | 0 | PASS |
| 1099 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).Key` | 169 | 1 | 0 | PASS |
| 1100 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).ProviderConfig` | 153 | 1 | 0 | PASS |
| 1101 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).SetBackupRpcAddrs` | 240 | 1 | 0 | PASS |
| 1102 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).SetRpcAddr` | 234 | 1 | 0 | PASS |
| 1103 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).Timeout` | 173 | 1 | 0 | PASS |
| 1104 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).TrustingPeriod` | 201 | 1 | 0 | PASS |
| 1105 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(*PenumbraProvider).Type` | 165 | 1 | 0 | PASS |
| 1106 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(PenumbraIBCHeader).ConsensusState` | 127 | 1 | 0 | PASS |
| 1107 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(PenumbraIBCHeader).Height` | 123 | 1 | 0 | PASS |
| 1108 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(PenumbraIBCHeader).NextValidatorsHash` | 135 | 1 | 0 | PASS |
| 1109 | production | `penumbra` | `relayer/chains/penumbra/provider.go:(PenumbraProviderConfig).BroadcastMode` | 76 | 1 | 0 | PASS |
| 1110 | production | `penumbra` | `relayer/chains/penumbra/provider.go:keysDir` | 432 | 1 | 0 | PASS |
| 1111 | production | `penumbra` | `relayer/chains/penumbra/provider.go:toPenumbraPacket` | 418 | 1 | 0 | PASS |
| 1112 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryConnectionsUsingClient` | 486 | 1 | 0 | PASS |
| 1113 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryDenomHash` | 857 | 1 | 0 | PASS |
| 1114 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryICQWithProof` | 1014 | 1 | 0 | PASS |
| 1115 | production | `penumbra` | `relayer/chains/penumbra/query.go:(*PenumbraProvider).QueryUnbondingPeriod` | 140 | 1 | 0 | PASS |
| 1116 | production | `penumbra` | `relayer/chains/penumbra/query.go:DefaultPageRequest` | 869 | 1 | 0 | PASS |
| 1117 | production | `penumbra` | `relayer/chains/penumbra/query.go:sendPacketQuery` | 939 | 1 | 0 | PASS |
| 1118 | production | `penumbra` | `relayer/chains/penumbra/query.go:writeAcknowledgementQuery` | 972 | 1 | 0 | PASS |
| 1119 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgPacketAck).Data` | 175 | 1 | 0 | PASS |
| 1120 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgPacketAck).Seq` | 178 | 1 | 0 | PASS |
| 1121 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgPacketAck).Timeout` | 181 | 1 | 0 | PASS |
| 1122 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgPacketAck).TimeoutStamp` | 185 | 1 | 0 | PASS |
| 1123 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).Data` | 105 | 1 | 0 | PASS |
| 1124 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).Seq` | 109 | 1 | 0 | PASS |
| 1125 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).Timeout` | 113 | 1 | 0 | PASS |
| 1126 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).TimeoutStamp` | 117 | 1 | 0 | PASS |
| 1127 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgRecvPacket).timeoutPacket` | 94 | 1 | 0 | PASS |
| 1128 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgTimeout).Data` | 28 | 1 | 0 | PASS |
| 1129 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgTimeout).Seq` | 32 | 1 | 0 | PASS |
| 1130 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgTimeout).Timeout` | 36 | 1 | 0 | PASS |
| 1131 | production | `penumbra` | `relayer/chains/penumbra/relayer_packets.go:(relayMsgTimeout).TimeoutStamp` | 40 | 1 | 0 | PASS |
| 1132 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgRegisterCounterpartyPayee` | 2293 | 1 | 0 | PASS |
| 1133 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).MsgSubmitQueryResponse` | 2281 | 1 | 0 | PASS |
| 1134 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).NewClientState` | 2027 | 1 | 0 | PASS |
| 1135 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).SendMessage` | 93 | 1 | 0 | PASS |
| 1136 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).SendMessagesToMempool` | 2286 | 1 | 0 | PASS |
| 1137 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(*PenumbraProvider).SubmitMisbehavior` | 466 | 1 | 0 | PASS |
| 1138 | production | `penumbra` | `relayer/chains/penumbra/tx.go:(_err).Error` | 83 | 1 | 0 | PASS |
| 1139 | production | `penumbra` | `relayer/chains/penumbra/tx.go:ackPacketQuery` | 1772 | 1 | 0 | PASS |
| 1140 | production | `penumbra` | `relayer/chains/penumbra/tx.go:rcvPacketQuery` | 1767 | 1 | 0 | PASS |
| 1141 | production | `ethermint` | `relayer/codecs/ethermint/algorithm.go:(ethSecp256k1Algo).Generate` | 99 | 1 | 0 | PASS |
| 1142 | production | `ethermint` | `relayer/codecs/ethermint/algorithm.go:(ethSecp256k1Algo).Name` | 51 | 1 | 0 | PASS |
| 1143 | production | `ethermint` | `relayer/codecs/ethermint/algorithm.go:EthSecp256k1Option` | 34 | 1 | 0 | PASS |
| 1144 | production | `ethermint` | `relayer/codecs/ethermint/codec.go:RegisterInterfaces` | 13 | 1 | 0 | PASS |
| 1145 | production | `ethermint` | `relayer/codecs/ethermint/encoding.go:jsonNameFromTag` | 339 | 1 | 0 | PASS |
| 1146 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(*PrivKey).UnmarshalAminoJSON` | 109 | 1 | 0 | PASS |
| 1147 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(*PubKey).UnmarshalAminoJSON` | 200 | 1 | 0 | PASS |
| 1148 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).Bytes` | 56 | 1 | 0 | PASS |
| 1149 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).MarshalAmino` | 87 | 1 | 0 | PASS |
| 1150 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).MarshalAminoJSON` | 102 | 1 | 0 | PASS |
| 1151 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).ToECDSA` | 131 | 1 | 0 | PASS |
| 1152 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PrivKey).Type` | 82 | 1 | 0 | PASS |
| 1153 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).Bytes` | 155 | 1 | 0 | PASS |
| 1154 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).MarshalAmino` | 178 | 1 | 0 | PASS |
| 1155 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).MarshalAminoJSON` | 193 | 1 | 0 | PASS |
| 1156 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).String` | 163 | 1 | 0 | PASS |
| 1157 | production | `ethermint` | `relayer/codecs/ethermint/ethsecp256k1.go:(PubKey).Type` | 168 | 1 | 0 | PASS |
| 1158 | production | `injective` | `relayer/codecs/injective/algorithm.go:(ethSecp256k1Algo).Generate` | 93 | 1 | 0 | PASS |
| 1159 | production | `injective` | `relayer/codecs/injective/algorithm.go:(ethSecp256k1Algo).Name` | 50 | 1 | 0 | PASS |
| 1160 | production | `injective` | `relayer/codecs/injective/algorithm.go:EthSecp256k1Option` | 32 | 1 | 0 | PASS |
| 1161 | production | `injective` | `relayer/codecs/injective/codec.go:RegisterInterfaces` | 20 | 1 | 0 | PASS |
| 1162 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PrivKey).Bytes` | 57 | 1 | 0 | PASS |
| 1163 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PrivKey).Type` | 76 | 1 | 0 | PASS |
| 1164 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PrivKey).UnmarshalAminoJSON` | 103 | 1 | 0 | PASS |
| 1165 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PubKey).String` | 149 | 1 | 0 | PASS |
| 1166 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PubKey).Type` | 154 | 1 | 0 | PASS |
| 1167 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(*PubKey).UnmarshalAminoJSON` | 186 | 1 | 0 | PASS |
| 1168 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PrivKey).MarshalAmino` | 81 | 1 | 0 | PASS |
| 1169 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PrivKey).MarshalAminoJSON` | 96 | 1 | 0 | PASS |
| 1170 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PrivKey).PubKey` | 62 | 1 | 0 | PASS |
| 1171 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PrivKey).Sign` | 110 | 1 | 0 | PASS |
| 1172 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PubKey).Bytes` | 144 | 1 | 0 | PASS |
| 1173 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PubKey).MarshalAmino` | 164 | 1 | 0 | PASS |
| 1174 | production | `injective` | `relayer/codecs/injective/ethsecp256k1.go:(PubKey).MarshalAminoJSON` | 179 | 1 | 0 | PASS |
| 1175 | production | `injective` | `relayer/codecs/injective/params.go:(*Params).ParamSetPairs` | 55 | 1 | 0 | PASS |
| 1176 | production | `injective` | `relayer/codecs/injective/params.go:(Params).String` | 49 | 1 | 0 | PASS |
| 1177 | production | `injective` | `relayer/codecs/injective/params.go:DefaultParams` | 39 | 1 | 0 | PASS |
| 1178 | production | `injective` | `relayer/codecs/injective/params.go:NewParams` | 29 | 1 | 0 | PASS |
| 1179 | production | `injective` | `relayer/codecs/injective/params.go:ParamKeyTable` | 24 | 1 | 0 | PASS |
| 1180 | production | `relayer` | `relayer/ics24.go:(*Chain).AddPath` | 41 | 1 | 0 | PASS |
| 1181 | production | `relayer` | `relayer/ics24.go:(*Chain).ErrCantSetPath` | 67 | 1 | 0 | PASS |
| 1182 | production | `relayer` | `relayer/ics24.go:(*Chain).ErrPathNotSet` | 62 | 1 | 0 | PASS |
| 1183 | production | `relayer` | `relayer/ics24.go:(*Chain).PathSet` | 24 | 1 | 0 | PASS |
| 1184 | production | `relayer` | `relayer/ics24.go:(*PathEnd).Vclient` | 10 | 1 | 0 | PASS |
| 1185 | production | `relayer` | `relayer/ics24.go:(*PathEnd).Vconn` | 15 | 1 | 0 | PASS |
| 1186 | production | `relayer` | `relayer/ics24.go:(PathEnd).String` | 19 | 1 | 0 | PASS |
| 1187 | production | `relayer` | `relayer/log-chain.go:(*Chain).LogFailedTx` | 55 | 1 | 0 | PASS |
| 1188 | production | `relayer` | `relayer/log-chain.go:(*Chain).LogRetryGetIBCUpdateHeader` | 74 | 1 | 0 | PASS |
| 1189 | production | `relayer` | `relayer/log-chain.go:(*Chain).errQueryUnrelayedPacketAcks` | 70 | 1 | 0 | PASS |
| 1190 | production | `relayer` | `relayer/log-chain.go:(*Chain).logPacketsRelayed` | 59 | 1 | 0 | PASS |
| 1191 | production | `relayer` | `relayer/path.go:(*Path).String` | 174 | 1 | 0 | PASS |
| 1192 | production | `relayer` | `relayer/path.go:(*PathWithStatus).PrintString` | 293 | 1 | 0 | PASS |
| 1193 | production | `relayer` | `relayer/path.go:GenPath` | 180 | 1 | 0 | PASS |
| 1194 | test | `relayer` | `relayer/pathEnd_test.go:TestOrderFromString` | 10 | 1 | 0 | PASS |
| 1195 | test | `relayer` | `relayer/pathEnd_test.go:TestStringFromOrder` | 27 | 1 | 0 | PASS |
| 1196 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessorBuilder).WithInitialBlockHistory` | 48 | 1 | 0 | PASS |
| 1197 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessorBuilder).WithMessageLifecycle` | 61 | 1 | 0 | PASS |
| 1198 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessorBuilder).WithPathProcessors` | 54 | 1 | 0 | PASS |
| 1199 | production | `processor` | `relayer/processor/event_processor.go:(EventProcessorBuilder).WithStuckPacket` | 67 | 1 | 0 | PASS |
| 1200 | production | `processor` | `relayer/processor/event_processor.go:NewEventProcessor` | 28 | 1 | 0 | PASS |
| 1201 | production | `processor` | `relayer/processor/message_processor.go:newMessageProcessor` | 77 | 1 | 0 | PASS |
| 1202 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).AddPacketsObserved` | 25 | 1 | 0 | PASS |
| 1203 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).IncBlockQueryFailure` | 53 | 1 | 0 | PASS |
| 1204 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).IncPacketsRelayed` | 29 | 1 | 0 | PASS |
| 1205 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).IncTxFailure` | 57 | 1 | 0 | PASS |
| 1206 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetClientExpiration` | 45 | 1 | 0 | PASS |
| 1207 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetClientTrustingPeriod` | 49 | 1 | 0 | PASS |
| 1208 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetFeesSpent` | 41 | 1 | 0 | PASS |
| 1209 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetLatestHeight` | 33 | 1 | 0 | PASS |
| 1210 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetUnrelayedAcks` | 65 | 1 | 0 | PASS |
| 1211 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetUnrelayedPackets` | 61 | 1 | 0 | PASS |
| 1212 | production | `processor` | `relayer/processor/metrics.go:(*PrometheusMetrics).SetWalletBalance` | 37 | 1 | 0 | PASS |
| 1213 | production | `processor` | `relayer/processor/metrics.go:NewPrometheusMetrics` | 69 | 1 | 0 | PASS |
| 1214 | production | `processor` | `relayer/processor/path_end.go:NewPathEnd` | 23 | 1 | 0 | PASS |
| 1215 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).localhostSentinelProofChannel` | 1026 | 1 | 0 | PASS |
| 1216 | production | `processor` | `relayer/processor/path_end_runtime.go:(*pathEndRuntime).localhostSentinelProofPacket` | 1015 | 1 | 0 | PASS |
| 1217 | production | `processor` | `relayer/processor/path_end_runtime.go:newPathEndRuntime` | 65 | 1 | 0 | PASS |
| 1218 | test | `processor` | `relayer/processor/path_end_test.go:TestAllowChannelFilter` | 72 | 1 | 0 | PASS |
| 1219 | test | `processor` | `relayer/processor/path_end_test.go:TestAllowChannelFilterWithSpecificPort` | 216 | 1 | 0 | PASS |
| 1220 | test | `processor` | `relayer/processor/path_end_test.go:TestDenyChannelFilter` | 144 | 1 | 0 | PASS |
| 1221 | test | `processor` | `relayer/processor/path_end_test.go:TestDenyChannelWithSpecificPort` | 256 | 1 | 0 | PASS |
| 1222 | test | `processor` | `relayer/processor/path_end_test.go:TestNoChannelFilter` | 21 | 1 | 0 | PASS |
| 1223 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).PathEnd1Messages` | 155 | 1 | 0 | PASS |
| 1224 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).PathEnd2Messages` | 160 | 1 | 0 | PASS |
| 1225 | production | `processor` | `relayer/processor/path_processor.go:(*PathProcessor).disablePeriodicFlush` | 132 | 1 | 0 | PASS |
| 1226 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).channelMessagesToSend` | 1081 | 1 | 0 | PASS |
| 1227 | production | `processor` | `relayer/processor/path_processor_internal.go:(*PathProcessor).connectionMessagesToSend` | 1110 | 1 | 0 | PASS |
| 1228 | production | `processor` | `relayer/processor/types.go:(*ChannelCloseLifecycle).messageLifecycler` | 85 | 1 | 0 | PASS |
| 1229 | production | `processor` | `relayer/processor/types.go:(*ChannelMessageLifecycle).messageLifecycler` | 73 | 1 | 0 | PASS |
| 1230 | production | `processor` | `relayer/processor/types.go:(*ConnectionMessageLifecycle).messageLifecycler` | 57 | 1 | 0 | PASS |
| 1231 | production | `processor` | `relayer/processor/types.go:(*FlushLifecycle).messageLifecycler` | 25 | 1 | 0 | PASS |
| 1232 | production | `processor` | `relayer/processor/types.go:(*PacketMessageLifecycle).messageLifecycler` | 41 | 1 | 0 | PASS |
| 1233 | production | `processor` | `relayer/processor/types.go:(ChannelKey).Counterparty` | 177 | 1 | 0 | PASS |
| 1234 | production | `processor` | `relayer/processor/types.go:(ChannelKey).MarshalLogObject` | 209 | 1 | 0 | PASS |
| 1235 | production | `processor` | `relayer/processor/types.go:(ChannelKey).MsgInitKey` | 188 | 1 | 0 | PASS |
| 1236 | production | `processor` | `relayer/processor/types.go:(ChannelKey).PreInitKey` | 200 | 1 | 0 | PASS |
| 1237 | production | `processor` | `relayer/processor/types.go:(ConnectionKey).Counterparty` | 226 | 1 | 0 | PASS |
| 1238 | production | `processor` | `relayer/processor/types.go:(ConnectionKey).MarshalLogObject` | 258 | 1 | 0 | PASS |
| 1239 | production | `processor` | `relayer/processor/types.go:(ConnectionKey).MsgInitKey` | 237 | 1 | 0 | PASS |
| 1240 | production | `processor` | `relayer/processor/types.go:(ConnectionKey).PreInitKey` | 249 | 1 | 0 | PASS |
| 1241 | production | `processor` | `relayer/processor/types.go:(IBCHeaderCache).Clone` | 628 | 1 | 0 | PASS |
| 1242 | production | `processor` | `relayer/processor/types.go:(IBCMessagesCache).Clone` | 101 | 1 | 0 | PASS |
| 1243 | production | `processor` | `relayer/processor/types.go:ChannelInfoChannelKey` | 670 | 1 | 0 | PASS |
| 1244 | production | `processor` | `relayer/processor/types.go:ConnectionInfoConnectionKey` | 680 | 1 | 0 | PASS |
| 1245 | production | `processor` | `relayer/processor/types.go:NewIBCMessagesCache` | 116 | 1 | 0 | PASS |
| 1246 | production | `processor` | `relayer/processor/types_internal.go:(*channelKeySendCache).get` | 423 | 1 | 0 | PASS |
| 1247 | production | `processor` | `relayer/processor/types_internal.go:(*channelKeySendCache).set` | 429 | 1 | 0 | PASS |
| 1248 | production | `processor` | `relayer/processor/types_internal.go:(*clientICQProcessingCache).get` | 508 | 1 | 0 | PASS |
| 1249 | production | `processor` | `relayer/processor/types_internal.go:(*clientICQProcessingCache).set` | 514 | 1 | 0 | PASS |
| 1250 | production | `processor` | `relayer/processor/types_internal.go:(*connectionKeySendCache).get` | 466 | 1 | 0 | PASS |
| 1251 | production | `processor` | `relayer/processor/types_internal.go:(*connectionKeySendCache).set` | 472 | 1 | 0 | PASS |
| 1252 | production | `processor` | `relayer/processor/types_internal.go:(*packetMessageSendCache).get` | 380 | 1 | 0 | PASS |
| 1253 | production | `processor` | `relayer/processor/types_internal.go:(*packetMessageSendCache).set` | 386 | 1 | 0 | PASS |
| 1254 | production | `processor` | `relayer/processor/types_internal.go:(*processingMessage).isProcessing` | 345 | 1 | 0 | PASS |
| 1255 | production | `processor` | `relayer/processor/types_internal.go:(*processingMessage).setFinishedProcessing` | 359 | 1 | 0 | PASS |
| 1256 | production | `processor` | `relayer/processor/types_internal.go:(*processingMessage).setProcessing` | 351 | 1 | 0 | PASS |
| 1257 | production | `processor` | `relayer/processor/types_internal.go:(channelIBCMessage).MarshalLogObject` | 201 | 1 | 0 | PASS |
| 1258 | production | `processor` | `relayer/processor/types_internal.go:(channelIBCMessage).msgType` | 194 | 1 | 0 | PASS |
| 1259 | production | `processor` | `relayer/processor/types_internal.go:(channelIBCMessage).tracker` | 187 | 1 | 0 | PASS |
| 1260 | production | `processor` | `relayer/processor/types_internal.go:(channelMessageToTrack).MarshalLogObject` | 657 | 1 | 0 | PASS |
| 1261 | production | `processor` | `relayer/processor/types_internal.go:(channelMessageToTrack).assembledMsg` | 649 | 1 | 0 | PASS |
| 1262 | production | `processor` | `relayer/processor/types_internal.go:(channelMessageToTrack).msgType` | 653 | 1 | 0 | PASS |
| 1263 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessage).MarshalLogObject` | 328 | 1 | 0 | PASS |
| 1264 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessage).msgType` | 321 | 1 | 0 | PASS |
| 1265 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessage).tracker` | 314 | 1 | 0 | PASS |
| 1266 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessageToTrack).MarshalLogObject` | 674 | 1 | 0 | PASS |
| 1267 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessageToTrack).assembledMsg` | 666 | 1 | 0 | PASS |
| 1268 | production | `processor` | `relayer/processor/types_internal.go:(clientICQMessageToTrack).msgType` | 670 | 1 | 0 | PASS |
| 1269 | production | `processor` | `relayer/processor/types_internal.go:(connectionIBCMessage).MarshalLogObject` | 275 | 1 | 0 | PASS |
| 1270 | production | `processor` | `relayer/processor/types_internal.go:(connectionIBCMessage).msgType` | 268 | 1 | 0 | PASS |
| 1271 | production | `processor` | `relayer/processor/types_internal.go:(connectionIBCMessage).tracker` | 261 | 1 | 0 | PASS |
| 1272 | production | `processor` | `relayer/processor/types_internal.go:(connectionMessageToTrack).MarshalLogObject` | 640 | 1 | 0 | PASS |
| 1273 | production | `processor` | `relayer/processor/types_internal.go:(connectionMessageToTrack).assembledMsg` | 632 | 1 | 0 | PASS |
| 1274 | production | `processor` | `relayer/processor/types_internal.go:(connectionMessageToTrack).msgType` | 636 | 1 | 0 | PASS |
| 1275 | production | `processor` | `relayer/processor/types_internal.go:(packetIBCMessage).MarshalLogObject` | 110 | 1 | 0 | PASS |
| 1276 | production | `processor` | `relayer/processor/types_internal.go:(packetIBCMessage).channelKey` | 130 | 1 | 0 | PASS |
| 1277 | production | `processor` | `relayer/processor/types_internal.go:(packetIBCMessage).msgType` | 103 | 1 | 0 | PASS |
| 1278 | production | `processor` | `relayer/processor/types_internal.go:(packetIBCMessage).tracker` | 96 | 1 | 0 | PASS |
| 1279 | production | `processor` | `relayer/processor/types_internal.go:(packetMessageToTrack).MarshalLogObject` | 623 | 1 | 0 | PASS |
| 1280 | production | `processor` | `relayer/processor/types_internal.go:(packetMessageToTrack).assembledMsg` | 615 | 1 | 0 | PASS |
| 1281 | production | `processor` | `relayer/processor/types_internal.go:(packetMessageToTrack).msgType` | 619 | 1 | 0 | PASS |
| 1282 | production | `processor` | `relayer/processor/types_internal.go:newChannelKeySendCache` | 417 | 1 | 0 | PASS |
| 1283 | production | `processor` | `relayer/processor/types_internal.go:newClientICQProcessingCache` | 502 | 1 | 0 | PASS |
| 1284 | production | `processor` | `relayer/processor/types_internal.go:newConnectionKeySendCache` | 460 | 1 | 0 | PASS |
| 1285 | production | `processor` | `relayer/processor/types_internal.go:newPacketMessageSendCache` | 374 | 1 | 0 | PASS |
| 1286 | production | `processor` | `relayer/processor/types_internal.go:packetInfoChannelKey` | 590 | 1 | 0 | PASS |
| 1287 | test | `processor_test` | `relayer/processor/types_test.go:(mockIBCHeader).ConsensusState` | 15 | 1 | 0 | PASS |
| 1288 | test | `processor_test` | `relayer/processor/types_test.go:(mockIBCHeader).Height` | 14 | 1 | 0 | PASS |
| 1289 | test | `processor_test` | `relayer/processor/types_test.go:(mockIBCHeader).NextValidatorsHash` | 16 | 1 | 0 | PASS |
| 1290 | production | `provider` | `relayer/provider/matcher.go:isMatchingTendermintClient` | 149 | 1 | 0 | PASS |
| 1291 | production | `provider` | `relayer/provider/matcher.go:isMatchingTendermintConsensusState` | 162 | 1 | 0 | PASS |
| 1292 | production | `provider` | `relayer/provider/matcher.go:tmClientCodec` | 18 | 1 | 0 | PASS |
| 1293 | production | `provider` | `relayer/provider/provider.go:(*TimeoutHeightError).Error` | 497 | 1 | 0 | PASS |
| 1294 | production | `provider` | `relayer/provider/provider.go:(*TimeoutOnCloseError).Error` | 525 | 1 | 0 | PASS |
| 1295 | production | `provider` | `relayer/provider/provider.go:(*TimeoutTimestampError).Error` | 513 | 1 | 0 | PASS |
| 1296 | production | `provider` | `relayer/provider/provider.go:(PacketInfo).Packet` | 98 | 1 | 0 | PASS |
| 1297 | production | `provider` | `relayer/provider/provider.go:(RelayerTxResponse).MarshalLogObject` | 205 | 1 | 0 | PASS |
| 1298 | production | `provider` | `relayer/provider/provider.go:(TendermintIBCHeader).ConsensusState` | 544 | 1 | 0 | PASS |
| 1299 | production | `provider` | `relayer/provider/provider.go:(TendermintIBCHeader).Height` | 540 | 1 | 0 | PASS |
| 1300 | production | `provider` | `relayer/provider/provider.go:(TendermintIBCHeader).NextValidatorsHash` | 552 | 1 | 0 | PASS |
| 1301 | production | `provider` | `relayer/provider/provider.go:NewTimeoutHeightError` | 501 | 1 | 0 | PASS |
| 1302 | production | `provider` | `relayer/provider/provider.go:NewTimeoutOnCloseError` | 529 | 1 | 0 | PASS |
| 1303 | production | `provider` | `relayer/provider/provider.go:NewTimeoutTimestampError` | 517 | 1 | 0 | PASS |
| 1304 | production | `relayer` | `relayer/query.go:QueryIBCHeaders` | 217 | 1 | 0 | PASS |
| 1305 | production | `relayer` | `relayer/query.go:QueryIBCUpdateHeaders` | 185 | 1 | 0 | PASS |
| 1306 | production | `relayer` | `relayer/query.go:QueryLatestHeights` | 22 | 1 | 0 | PASS |
| 1307 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_LastUpdateHeight` | 124 | 1 | 0 | PASS |
| 1308 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintChainId` | 14 | 1 | 0 | PASS |
| 1309 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintClientId` | 26 | 1 | 0 | PASS |
| 1310 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintExpired_WhenTimeIsInPast` | 40 | 1 | 0 | PASS |
| 1311 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintGood_WhenTimeIsInFuture` | 68 | 1 | 0 | PASS |
| 1312 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintRFC822FormattedTime_WhenTimeIsInFuture` | 82 | 1 | 0 | PASS |
| 1313 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintRFC822FormattedTime_WhenTimeIsInPast` | 54 | 1 | 0 | PASS |
| 1314 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_PrintRemainingTime_WhenTimeIsInFuture` | 96 | 1 | 0 | PASS |
| 1315 | test | `relayer` | `relayer/query_test.go:TestSPrintClientExpiration_TrustingPeriod` | 110 | 1 | 0 | PASS |
| 1316 | test | `relayer` | `relayer/query_test.go:mockChain` | 136 | 1 | 0 | PASS |
| 1317 | test | `relayer` | `relayer/query_test.go:mockClientStateInfo` | 151 | 1 | 0 | PASS |
| 1318 | production | `relayer` | `relayer/relayMsgs.go:(SendMsgsResult).Error` | 128 | 1 | 0 | PASS |
| 1319 | production | `relayer` | `relayer/relayMsgs.go:AsRelayMsgSender` | 93 | 1 | 0 | PASS |
| 1320 | test | `relayer_test` | `relayer/relaymsgs_test.go:(fakeRelayerMessage).MsgBytes` | 55 | 1 | 0 | PASS |
| 1321 | test | `relayer_test` | `relayer/relaymsgs_test.go:(fakeRelayerMessage).Type` | 51 | 1 | 0 | PASS |
| 1322 | test | `relayer_test` | `relayer/relaymsgs_test.go:TestRelayMsgs_IsMaxTx` | 14 | 1 | 0 | PASS |
| 1323 | test | `relayer_test` | `relayer/relaymsgs_test.go:TestRelayMsgs_Send_Success` | 59 | 1 | 0 | PASS |
| 1324 | test | `relayer` | `relayer/strategies_test.go:TestApplyChannelFilterAllowRule` | 10 | 1 | 0 | PASS |
| 1325 | test | `relayer` | `relayer/strategies_test.go:TestApplyChannelFilterDenyRule` | 33 | 1 | 0 | PASS |
| 1326 | test | `relayer` | `relayer/strategies_test.go:TestApplyChannelFilterNoRule` | 56 | 1 | 0 | PASS |
| 1327 | test | `relayer` | `relayer/strategies_test.go:TestValidateChannelFilterRule` | 76 | 1 | 0 | PASS |

## Criterio de conclusao futuro

O trabalho de complexidade so termina quando uma nova execucao no SHA de entrega registrar `0` para ciclo >=10, cognitiva >=10 e uniao; `go build ./...` passar; unit tests passarem sem depender de endpoint live; testes focados das fronteiras alteradas passarem; e o CI executar o mesmo gate com pins compativeis com Go 1.21 (ou com a versao Go declarada depois do upgrade intencional).
