# Baseline de manutenção do Go relayer

Data: 2026-07-15  
Base auditada: `bef2e868f157659b403fe1303ee121fb69fec9e6`

Este documento fixa o ponto de partida do programa de modernização. Ele não declara o repositório pronto: torna o estado atual reproduzível, define os gates e divide a implementação em lotes revisáveis.

## Progresso: M0.1 a M1.1b-d

Implementado na branch `Pantani/cx/m0-baseline`:

| Entrega | Estado |
|---|---|
| Toolchain | Go 1.25.9 em `go.mod`, `go.work`, `interchaintest/go.mod`, Actions e Docker |
| Dynamic fee | Resposta Osmosis decodificada como protobuf; teste live removido e substituído por fixture offline de 17 bytes |
| Caracterização Classic | Golden tests de packet, channel, connection, client e evento não-IBC |
| Lint | Config v2, golangci-lint v2.12.2 pinado, formatters e job de CI em `main` |
| Build e testes | `go build -mod=readonly ./...` e `go test -mod=readonly ./...` passam com Go 1.25.9 |
| Complexidade tocada | Todas as funções criadas ou modificadas neste fluxo ficam `<10/<10` |
| Complexidade global | Continua vermelha: 98 ciclomaticas, 152 cognitivas, união 158; máximo 48/169 |

O lint verde inicial contém `bodyclose`, `govet`, `ineffassign` e `nolintlint`, além de `gofmt`/`goimports`. A migração também revelou dívida preexistente em `staticcheck`, `unused`, `gosec` e linters de estilo; ela deve ser reduzida em lotes próprios, sem exclusões `nolint` em massa.

O M0.2 também foi concluído na mesma branch, em três fatias paralelas:

| Entrega | Estado |
|---|---|
| FeeGrant | Dois testes monolíticos `40/169` e `39/166` reduzidos para `1/0`; arquivo máximo `8/7`; quatro cenários Docker passaram com `-race` |
| CLI/config | Builders Classic extraídos, contratos Cobra caracterizados e `ValidatePathEnd` reduzido para `7/6` |
| Ethermint | Traversal, tipos, SignDoc EIP-712 e derivação HD decompostos; todo o escopo manuscrito máximo `8/7` |
| Build e testes | Raiz e interchaintest compilam; unit tests, race Ethermint, lint e diff-check passam |
| Complexidade global | `98/152/158` passou para `86/139/145` em ciclo/cognitiva/união; máximo `48/99` |

O lifecycle global dos codecs EIP-712 continua sem initializer e foi registrado como gap preexistente, não como suporte concluído.

O M0.3 concluiu a fundação protocol-neutral sem alterar os pins ou ligar o v2 ao runtime:

| Entrega | Estado |
|---|---|
| Core neutro | Protocol/capabilities, chaves comparáveis e envelopes de packet, evento ordenado, proof e mensagem sem dependências Cosmos |
| Adaptadores | Round-trip Classic sem perda e DTO v2 contract-only alinhado a ibc-go v11.2.0, sem importar `/v11` |
| Config | Classic implícito preserva YAML legado; v2 explícito usa client pair/prefix e rejeita connection/channel filter |
| Segurança de dispatch | `StartRelayer` e `ChainsFromPath` rejeitam v2 antes do runtime Classic, RPC ou goroutines |
| Build e testes | 247 testes raiz, 143 testes focados com `-race`, build raiz/interchaintest, lint e module verify passam |
| Complexidade global | `86/139/145` passou para `86/138/144`; máximo permanece `48/99`; core novo máximo `8/8` |

M1.1a, M1.1b e M1.1b-d avançaram a base até o grafo atual e restauraram o harness de integração:

| Entrega | Estado |
|---|---|
| Eventos v2 | Ingestão ABCI ordenada, correlação por `msg_index`/`message.action`, classificação Classic/v2 e decode limitado de packet/ack em sidecar |
| Grafo raiz | Cosmos SDK 0.54.3, CometBFT 0.39.3, IBC-Go v11.2.0, Store v2 e Log v2; testes raiz, lint e cross-build Linux passam |
| Framework de integração | `github.com/cosmos/interchaintest/v11@v11.0.0-20260507171724-1a8c536981a8`, sem fork; módulo compila readonly isolado e no workspace |
| Contrato CI | `interchaintest-contract` verifica os dois módulos, dependências carregadas e testes unitários do adapter antes das lanes Docker |
| Compatibilidade Classic | Setup Gaia v14.1.0 + Osmosis v22.0.0 passou 9 subtestes reais; localhost stateful legado é decodificado apenas para leitura |
| Limite conhecido | Ainda não há chain runtime v11.2.0 nem relay v2 operacional; a imagem oficial `simd:v11.2.0` não está publicada |
| Complexidade global | 83 violações ciclomáticas, 134 cognitivas, união 138; máximos 48/99; código novo/tocado `<10/<10` |

## Estado confirmado no snapshot inicial

| Área | Resultado |
|---|---|
| Checkout | `detached HEAD`, no mesmo SHA de `main` e `origin/main` |
| Fork | `Pantani/relayer`, zero PRs abertos |
| Upstream | `cosmos/relayer` arquivado, 35 PRs ainda abertos |
| Branches do fork | 79 heads remotas: 5 integradas, 14 ativas por heurística temporal, 59 históricas e 1 sem merge-base |
| Dependências atuais | Go 1.21, Cosmos SDK v0.50.11, ibc-go v8.2.0 e CometBFT v0.38.12 |
| Build | `go build ./...` passa |
| Testes | Histórico do snapshot: 96 passam e 1 falha porque `TestQueryBaseFee` dependia de RPC público; corrigido no M0.1 |
| Lint | `.golangci.yml` é anterior ao schema v2; golangci-lint v2.12.2 encerra antes da análise |
| Complexidade | 1.327 funções manuscritas; 158 violam ciclomatica ou cognitiva `<10`; máximo 48/169 |

## Gate de complexidade

`make complexity` executa versões pinadas de `gocyclo` e `gocognit`, inclui testes e exclui somente arquivos com marcador canônico `Code generated ... DO NOT EDIT.`. O limite é máximo 9 por função nas duas métricas.

O alvo está deliberadamente vermelho no snapshot. Não há baseline permissivo, `nolint` ou aumento do limite. Ele deve entrar como gate obrigatório de CI quando o inventário chegar a zero; até lá, cada função nova ou alterada precisa satisfazer o gate e o relatório global acompanha a redução.

No lote M1.1b-e, os parsers duplicados de identificadores de eventos foram
consolidados sem alterar os erros públicos ou a semântica de atributo vazio. O
inventário global passou para 83 violações ciclomáticas, 130 cognitivas e 134
funções na união, com máximos preservados em 48/99. As quatro funções tocadas e
os novos testes ficam abaixo de 10 nas duas métricas.

## Lotes de entrega

1. **M0.1 — baseline determinístico: concluído:** dynamic fee offline, lint pinado, caracterização Classic e Go 1.25.9.
2. **M0.2 — bordas simples: concluído:** testes FeeGrant, builders/validadores Classic e codecs Ethermint foram decompostos sem mudar protocolo.
3. **M0.3 — modelo neutro: concluído:** envelopes/capabilities, adaptadores Classic/v2, config discriminada e guardas de runtime.
4. **M1.1 — em andamento:** ingestão de eventos, grafo v11 e harness foram concluídos; faltam chain runtime v11.2, chave por client/sequence, state machine e timeout timestamp-only.
5. **M1.2 — provas e mensagens v2:** counterparty/config, updates de client, queries, Recv/Ack/Timeout e resultado NOOP/SUCCESS/FAILURE.
6. **M1.3 — execução confiável:** retry classificado, confirmação, checkpoint, restart e coexistência Classic/v2.
7. **M2 — stack atual:** ICS-20 v2, GMP, callbacks/rate-limit v2, PFM Classic, light clients, observabilidade, fault injection e fuzz.
8. **M3/M4 — novos destinos:** Eureka Cosmos-EVM; outros ecossistemas somente depois de contratos estáveis.

Os hotspots do processor só devem ser decompostos depois de definido o modelo Classic/v2, pois hoje concentram a semântica de ordenação, cache, retry, ack e timeout. A previsão inicial é de 12 a 18 PRs pequenas nas oito trilhas, sempre com testes focados e scores `<10/<10` para código criado ou tocado.

## Regra de conclusão

O programa só termina quando build e testes determinísticos passam, todas as funções manuscritas têm ciclomatica e cognitiva no máximo 9, IBC Classic possui política explícita de compatibilidade e a matriz M1+M2 de IBC v2 está verde contra chains reais pinadas.
