# M0.2 — CLI/config

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
Base do lote: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Escopo realizado

- `cmd/tx.go`: os builders `createClientsCmd`, `createClientCmd`, `createConnectionCmd` e `linkCmd` agora registram a mesma UX Cobra e delegam a execucao para funcoes nomeadas pequenas;
- `cmd/tx_handshake.go`: separa, sem reordenar efeitos observaveis, leitura de flags, resolucao de path/chains, verificacao de chaves, consultas com retry, execucao do handshake Classic e persistencia dos IDs criados;
- `cmd/config.go`: `Config.ValidatePathEnd` usa retornos antecipados e delega a sequencia client/connection a um helper;
- `cmd/tx_handshake_test.go`: caracteriza `Use`, aliases, flags, parsing das opcoes, orientacao src/dst do client criado e erros exatos de chave ausente;
- `cmd/config_path_validation_test.go`: caracteriza chain ausente, identificadores vazios, validacao client+connection, connection sem client e propagacao de erro do provider.

Nao houve mudanca de nomes/defaults de flags, aliases, mensagens de erro, ordem de criacao/persistencia, politicas de retry ou chamadas de provider.

## Limite IBC v2

Os tipos privados extraidos representam explicitamente apenas o fluxo IBC Classic existente. Nenhuma flag, schema de config ou abstracao de handshake IBC v2 foi inventada neste lote. IBC v2 deve receber um contrato de comando separado depois das decisoes de UX/config; estender os inputs Classic criados aqui fica deliberadamente fora do escopo.

Tambem ficaram fora:

- `flushCmd` e demais fluxos de packet/event processor, pois dependem do modelo de payload/identificadores IBC v2;
- FeeGrant;
- `relayer/codecs/ethermint/**`;
- testes interchain/Docker, que exigem infraestrutura externa e pertencem ao fan-in de QA.

## Complexidade

Ferramentas pinadas:

- `github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0`;
- `github.com/uudashr/gocognit/cmd/gocognit@v1.2.1`.

| Funcao | Ciclomatica antes | Cognitiva antes | Ciclomatica depois | Cognitiva depois |
|---|---:|---:|---:|---:|
| `createClientsCmd` | 14 | 26 | 1 | 0 |
| `createClientCmd` | 23 | 45 | 1 | 0 |
| `createConnectionCmd` | 21 | 40 | 1 | 0 |
| `linkCmd` | 26 | 50 | 1 | 0 |
| `(*Config).ValidatePathEnd` | 12 | 15 | 7 | 6 |
| `(*Config).validatePathEndIdentifiers` | n/a | n/a | 3 | 2 |

Observacao: `_workspace/08_m0_2_snapshot.md` registrou `ValidatePathEnd` com cognitiva 10; a reexecucao direta com `gocognit@v1.2.1 -test` antes da edicao retornou 15, valor usado nesta comparacao.

O maior helper novo e `readSingleClientOptions`, com `7/6`. Nenhuma funcao criada ou tocada nesta fatia atingiu 10 em qualquer metrica. Os testes novos tambem ficaram abaixo de 10; o maior foi `TestHandshakeCommandContracts`, com `3/4`.

Comandos de medicao:

```sh
GOTOOLCHAIN=go1.25.9 go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 9 cmd/tx_handshake.go cmd/tx_handshake_test.go cmd/config_path_validation_test.go
GOTOOLCHAIN=go1.25.9 go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 9 -test cmd/tx_handshake.go cmd/tx_handshake_test.go cmd/config_path_validation_test.go
GOTOOLCHAIN=go1.25.9 go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 0 cmd/tx.go cmd/config.go
GOTOOLCHAIN=go1.25.9 go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 0 -test cmd/tx.go cmd/config.go
```

## Verificacao

| Check | Resultado |
|---|---|
| `gofmt -w cmd/tx.go cmd/tx_handshake.go cmd/tx_handshake_test.go cmd/config.go cmd/config_path_validation_test.go` | PASS |
| `GOTOOLCHAIN=go1.25.9 go test -mod=readonly ./cmd` | PASS |
| `GOTOOLCHAIN=go1.25.9 go build -mod=readonly ./...` | PASS |
| `make lint` | PASS — `0 issues`, todos os modulos verificados |
| metricas focadas `-over 9` | PASS — nenhuma saida/violacao nos arquivos novos; funcoes tocadas em `tx.go` e `config.go` abaixo de 10 |

O teste unitario completo do repositorio e os testes interchain/Docker ficam para o fan-in, evitando atribuir a esta fatia resultados afetados pelas edicoes concorrentes de FeeGrant e Ethermint.

## Estado de entrega

Working tree preservado, sem commit, push ou troca de branch.
