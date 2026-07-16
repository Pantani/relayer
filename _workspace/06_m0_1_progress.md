# M0.1 — baseline determinístico e toolchain

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
Base: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Entregas

### Dynamic fee Osmosis

- Substituído o parsing de bytes por remoção de caracteres de controle por uma resposta protobuf tipada local.
- O `LegacyDec` é reconstruído diretamente da representação inteira escalada usada pelo custom type Osmosis, sem conversão intermediária para `float64`.
- Erros de transporte, resposta nil, código ABCI, protobuf inválido, campo ausente, decimal inválido e denom inválido têm erros distintos.
- O teste não inicializa provider nem acessa RPC público. A fixture usa campo protobuf de 17 bytes, reproduzindo a mudança que quebrou o parser anterior.

Contrato primário conferido em `osmosis-labs/osmosis`: `QueryEipBaseFeeResponse.base_fee` é o campo protobuf 1 e usa o custom type decimal do SDK.

### IBC Classic

Adicionados testes de caracterização para:

- `send_packet`, incluindo sequence, ports/channels, order, height/timestamp e data hex;
- `channel_open_ack`;
- `connection_open_try`;
- `update_client`, incluindo consensus height e header;
- descarte de evento não-IBC.

Esses testes congelam o contrato Classic antes da introdução dos adaptadores v2.

### Toolchain e lint

- Go 1.25.9 pinado em `.go-version`, módulo principal, workspace e módulo interchaintest.
- Actions usam `actions/setup-go@v6` com `.go-version`; o workflow de build voltou a observar a branch default `main`.
- Docker usa a imagem verificada `golang:1.25.9-alpine3.22`.
- O tag `goreleaser/goreleaser-cross:v1.25.9` foi verificado antes de atualizar o Makefile.
- `.golangci.yml` migrou para schema v2.
- `make lint` instala `golangci-lint v2.12.2` num caminho cacheado que inclui a versão e passa no repositório.

O baseline verde habilita `bodyclose`, `govet`, `ineffassign` e `nolintlint`, além de `gofmt`/`goimports`. A auditoria de migração também encontrou dívida anterior nos conjuntos mais amplos:

- `staticcheck`: 50 ocorrências;
- `errcheck`: 50 ocorrências;
- `unused`: 18 ocorrências;
- `govet inline`: 6 ocorrências do analyzer novo, desabilitado explicitamente;
- `ineffassign`: 1 ocorrência, corrigida em `BuildSimTx`;
- `gosec` e linters de estilo: dívida ampla, não ativada como CI vermelho neste lote.

Esses linters serão habilitados progressivamente depois de corrigir suas ocorrências; não foram adicionadas diretivas `nolint` ou exclusões por arquivo.

## Verificação

```text
GOTOOLCHAIN=go1.25.9 go build -mod=readonly ./...       PASS
GOTOOLCHAIN=go1.25.9 go test -mod=readonly ./...        PASS
go test ./relayer/chains/cosmos ./relayer/chains       PASS
interchaintest go test -run '^$' ./...                  PASS (compilacao)
interchaintest go build ./...                           PASS
make lint                                               PASS, 0 issues
golangci-lint config verify                             PASS
Docker golang:1.25.9-alpine3.22 manifest                PASS
goreleaser-cross:v1.25.9 manifest                       PASS
complexidade das funções criadas/modificadas            PASS, <10/<10
make complexity global                                  FAIL esperado
```

Os cenarios interchain completos, que dependem de Docker e redes locais, nao foram executados neste lote; apenas a compilacao integral do modulo foi validada.

Rebaseline após os novos testes:

```text
arquivos manuscritos=136
funções=1.337
ciclomatica >=10=98
cognitiva >=10=152
união=158
máximos=48/169
```

## Próximo lote

M0.2 começa pelas bordas que não dependem da arquitetura IBC v2: decomposição dos dois testes `TestRelayerFeeGrant`, builders/validadores de CLI/config e codecs Ethermint, sempre com caracterização antes da refatoração e `<10/<10` por função tocada.
