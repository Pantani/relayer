# Snapshot de entrada

- Data da auditoria: 2026-07-15 (America/Sao_Paulo)
- Repositorio: `git@github.com:Pantani/relayer.git`
- Branch default remota: `main`
- Estado do checkout: `detached HEAD`
- HEAD/base auditada: `bef2e868f157659b403fe1303ee121fb69fec9e6`
- Ultimo commit: `2025-04-03T21:49:41+03:00`, `test: update keys_test.go (#1552)`
- Tag descritiva: `v2.6.0-6-gbef2e86`
- Runtime local: `go1.26.5 darwin/arm64`
- Contrato do modulo: Go 1.21
- Dependencias centrais atuais: Cosmos SDK `v0.50.11`, ibc-go `v8.2.0`, CometBFT `v0.38.12`
- PRs abertos em `Pantani/relayer`: 0 no momento do snapshot
- Branches remotas observadas antes do inventario detalhado: 79 alem da branch local `main`

## Contrato do trabalho

1. Complexidade ciclomatica por funcao manuscrita menor que 10 (maximo 9).
2. Complexidade cognitiva no mesmo escopo menor que 10 (maximo 9).
3. Codigo gerado fica fora do gate apenas quando o primeiro bloco do arquivo possui o marcador canonico `Code generated ... DO NOT EDIT.`.
4. Branches e PRs sao inventariados sem merge, checkout ou exclusao.
5. O roadmap usa releases oficiais verificadas em 2026-07-15 e separa prereleases.
6. IBC Classic e IBC v2 sao tratados como protocolos distintos.

## Ferramentas de medicao pinadas

- `github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0`
- `github.com/uudashr/gocognit/cmd/gocognit@v1.1.4` (ultima versao compativel com o contrato Go 1.21; v1.2.1 requer Go 1.24)

Comandos baseline:

```sh
bash ./scripts/check-complexity.sh
```

## Snapshot incremental M1.1b

- Data: 2026-07-15 (America/Sao_Paulo)
- Branch de trabalho: `Pantani/cx/m0-baseline`
- Base preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`
- Estado: lotes M0.1-M1.1a ainda nao commitados nem publicados; o M1.1b parte
  desse mesmo diff e nao pode descartar mudancas anteriores.
- Contrato Go atual: `go 1.25.9` no modulo raiz e no workspace.
- Runtime local: `go1.26.5 darwin/arm64`.
- Grafo selecionado antes do M1.1b: Cosmos SDK `v0.50.11`, ibc-go `v8.2.0`,
  CometBFT `v0.38.12`, gogoproto `v1.7.0`, Store `v1.1.1` e Log `v1.4.1`.
- Superficie inicial: 71 arquivos Go importam `/v8`, 7 importam os modulos
  `cosmossdk.io/x` removidos, 2 importam Store v1 diretamente e 4 importam
  `cometbft/crypto/sr25519`.
- Alvo pinado para o lote: ibc-go `v11.2.0`, Cosmos SDK `v0.54.3` e CometBFT
  `v0.39.3`, mantendo IBC Classic compilavel e com regressao verde.
- Context7 nao estava exposto pela sessao; fontes oficiais e o source das tags
  foram usados como fallback rastreavel.

## M1.1b-d — integration harness compatibility (2026-07-15)

- base/HEAD: `bef2e868f157659b403fe1303ee121fb69fec9e6`
- branch: `Pantani/cx/m0-baseline`
- entrada: continuar o próximo lote após a migração raiz SDK `0.54.3`,
  CometBFT `0.39.3` e ibc-go `v11.2.0`
- estado inicial: working tree mantém os lotes M0.1–M1.1b não commitados;
  preservar todas as mudanças locais e não trocar/resetar a branch
- bloqueio de entrada: `GOWORK=off go test -mod=readonly -run '^$' ./...`
  em `interchaintest/` falha pela fronteira Store v1 de
  `cosmossdk.io/x/upgrade@v0.2.0` contra Store v2 do SDK `0.54`
- objetivo do sublote: escolher e implementar uma fronteira de integração
  coerente sem remover silenciosamente cenários Classic nem alegar E2E v11
  que não foi executado
- contrato incremental: toda função Go nova ou com corpo alterado deve ficar
  com complexidade ciclomática e cognitiva máximas de 9

### Saída M1.1b-d

- framework oficial `/v11` integrado sem fork no pseudo-version
  `v11.0.0-20260507171724-1a8c536981a8`;
- compile/build readonly aprovados no workspace e isoladamente;
- setup Classic Gaia v14.1.0 + Osmosis v22.0.0 aprovado em 9 subtestes;
- runtime chain v11.2 e relay v2 permanecem explicitamente não verificados;
- complexidade global ao fim: `83/134/138`, máximos `48/99`.
