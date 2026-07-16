# M1.1b-d — opções para o framework `interchaintest`

Consulta: 2026-07-15 (America/Sao_Paulo)  
Relayer auditado: `bef2e868f157659b403fe1303ee121fb69fec9e6` mais a árvore de trabalho M1.1 em andamento  
Escopo deste documento: compilação e arquitetura do módulo aninhado `interchaintest`; execução real contra uma chain SDK 0.54/IBC-Go v11 é um gate separado.

## Decisão

Adotar agora o `cosmos/interchaintest` oficial em `main`, fixado pelo pseudo-version:

```text
github.com/cosmos/interchaintest/v11 v11.0.0-20260507171724-1a8c536981a8
```

Essa rota compilou em experimento isolado contra o relayer local com SDK 0.54.3, Store v2, CometBFT 0.39.3 e IBC-Go v11.2.0. Ela mantém conformance, cenários, reporter, chain factory, relayer Docker e relayer in-process. Não há justificativa técnica atual para fork ou para substituir a suíte por uma orquestração black-box própria.

## Estado oficial verificado

| Linha | Estado em 2026-07-15 | SDK | Store/log | IBC-Go | CometBFT |
|---|---|---:|---|---:|---:|
| tag `v10.0.1`, SHA `953cd349...` | última tag publicada; módulo `/v10` | 0.53.4 | Store v1, Log v1 | v10.3.0 | 0.38.19 |
| `main`, SHA `1a8c536...` de 2026-05-07 | ainda sem tag `/v11`; módulo `/v11` | 0.54.0 | Store v2, Log v2 | v11.0.0 | 0.39.0 |
| alvo deste relayer | versões já fixadas no módulo raiz | 0.54.3 | Store v2 2.0.0, Log v2 2.1.0 | v11.2.0 | 0.39.3 |

Fontes primárias:

- [`main` fixado em `1a8c536`](https://github.com/cosmos/interchaintest/commit/1a8c536981a88e3d7684f3ebc2430ef424f3ee8c)
- [`go.mod` do SHA de `main`](https://github.com/cosmos/interchaintest/blob/1a8c536981a88e3d7684f3ebc2430ef424f3ee8c/go.mod)
- [`go.mod` da tag `v10.0.1`](https://github.com/cosmos/interchaintest/blob/v10.0.1/go.mod)
- [tags oficiais](https://github.com/cosmos/interchaintest/tags)
- [adoção de Store v2 no upstream](https://github.com/cosmos/interchaintest/commit/08abe11c26f038f471ef8244ef760f55017141dc)
- [remoção dos imports de Store v1 no upstream](https://github.com/cosmos/interchaintest/commit/7ab504b2f91d642f85d6a80cb88b54ba1537f0b9)

O endpoint de “latest release” do GitHub retornou 404: o projeto publica tags, mas ainda não publicou uma release `/v11`. Por isso o SHA deve ser registrado e renovado de forma controlada, sem usar `@main` flutuante.

## Causa exata do bloqueio local

No início desta auditoria, o módulo aninhado misturava:

- `github.com/cosmos/interchaintest/v10 v10.0.1`;
- Cosmos SDK 0.54.3 e Store v2 selecionados pelo módulo raiz/local;
- imports locais já migrados para IBC-Go v11.

O erro reproduzido com:

```bash
cd interchaintest
GOWORK=off go test -mod=readonly -run '^$' ./...
```

foi:

```text
cosmossdk.io/x/upgrade@v0.2.0/types/storeloader.go:
cannot use cosmossdk.io/store/types.CommitMultiStore as
github.com/cosmos/cosmos-sdk/store/v2/types.CommitMultiStore
```

Os caminhos de importação foram comprovados com `go mod why -m`:

```text
local interchaintest
  -> github.com/cosmos/interchaintest/v10/chain/cosmos
     -> cosmossdk.io/x/upgrade
        -> cosmossdk.io/store/types
           -> cosmossdk.io/log

local interchaintest
  -> github.com/cosmos/interchaintest/v10/chain/cosmos
     -> github.com/cosmos/ibc-go/v10/modules/core/02-client/types
        -> cosmossdk.io/store/prefix

local interchaintest
  -> github.com/cosmos/interchaintest/v10/chain/cosmos
     -> github.com/cosmos/interchain-security/v7/x/ccv/provider
```

`chain/cosmos` era importado diretamente por 16 arquivos locais, incluindo o adaptador não-teste `relayer.go` e os cenários de fee middleware, ICA, misbehaviour, múltiplos canais, filtros, RPC backup, relay-many, Stride e boundary. Os demais pontos estruturais são:

| Import local | Uso | Consequência |
|---|---|---|
| `interchaintest/v10` | 17 testes/factory | carrega o framework e as factories built-in |
| `interchaintest/v10/chain/cosmos` | 16 arquivos | é o caminho que introduz app/SDK/IBC v10/upgrade v1 |
| `interchaintest/v10/conformance` | `ibc_test.go`, boundary | preserva a suíte canônica de conformance |
| `interchaintest/v10/relayer` e `/relayer/rly` | factory e testes Docker | integração com o binário relayer e capabilities |
| `interchaintest/v10/testutil` | 14 arquivos | polling, blocos, ack e helpers de integração |
| `cosmos-sdk/types/module/testutil` | localhost e Stride setup | configuração de codec/app para fixtures locais |
| `ibc-go/v11/testing` | misbehaviour | construção/assinatura de headers maliciosos |

Portanto, trocar apenas dependências diretas do `go.mod` não poderia resolver o problema: a implementação `/v10/chain/cosmos` compila código Store v1 no mesmo build em que `baseapp` já exige Store v2.

## Experimento isolado que compilou

Todos os arquivos foram copiados para `/tmp`; nenhum `go.mod` de produção foi usado como área de experimento.

Resumo reproduzível:

```bash
RELAYER_ROOT=$(pwd)
tmp=$(mktemp -d /tmp/relayer-itest-v11-exp.XXXXXX)
cp -R interchaintest "$tmp/module"
cd "$tmp/module"

# Troque mecanicamente os imports /v10 por /v11 e aplique os adapters
# enumerados na próxima seção.
go mod edit -droprequire=github.com/cosmos/interchaintest/v10
go mod edit -require=github.com/cosmos/interchaintest/v11@v11.0.0-20260507171724-1a8c536981a8
go mod edit -require=github.com/moby/go-archive@v0.1.0
go mod edit -replace=github.com/cosmos/relayer/v2="$RELAYER_ROOT"
go mod tidy
go test -mod=readonly -run '^$' ./...
go test -mod=readonly -list '^TestScenario' ./...
```

Resultado final observado:

```text
ok github.com/cosmos/relayer/v2/interchaintest [no tests to run]
ok github.com/cosmos/relayer/v2/interchaintest/stride [no tests to run]

TestScenarioClientThresholdUpdate
TestScenarioClientTrustingPeriodUpdate
TestScenarioICAChannelClose
TestScenarioInterchainAccounts
TestScenarioPathFilterAllow
TestScenarioPathFilterDeny
TestScenarioTendermint37Boundary
TestScenarioStrideICAandICQ
```

Também foi criado um `go.work` temporário contendo o módulo raiz e o módulo do experimento. Ambos compilaram com `-mod=readonly`; o padrão `./...` executado no diretório raiz continuou sem atravessar o boundary do módulo aninhado.

### Adaptações de API necessárias

Além da troca mecânica `/v10 -> /v11`, o protótipo precisou destas adaptações:

1. Fixar `github.com/moby/go-archive v0.1.0`. A v0.2.0 removeu o alias `archive.Compression` ainda usado por Docker 28.5.2.
2. Migrar `dockertypes.ImageRemoveOptions` para `api/types/image.RemoveOptions`.
3. Usar `github.com/moby/moby/client` na assinatura da `RelayerFactory`.
4. Renomear `ibc.DockerImage.UidGid` para `UIDGID`.
5. Adicionar `signingAlgorithm` a `Relayer.AddKey` e encaminhar `--signing-algorithm`.
6. Implementar `Relayer.ContainerImage()` para o relayer in-process.
7. Implementar `Relayer.CreateClient(...)` e encaminhar todas as opções de client.
8. Migrar `UpdatePath(ChannelFilter)` para `UpdatePath(PathUpdateOptions)`.
9. Tratar alturas como `int64` nas APIs `Height`, `PollForAck` e logs.
10. Migrar `transfer.DenomTrace` para `transfer.Denom`.
11. Para misbehaviour, obter `LatestHeight` do client Tendermint concreto e usar `cometbft/types.MockPV`; o mock `ibc-go/testing/mock.PV` não existe em v11.

Depois dessas mudanças, não houve outro erro de compilação.

### Observação sobre Store/Log v1 “dormente”

No experimento:

- `go mod why -m cosmossdk.io/store` e `go mod why -m cosmossdk.io/log` responderam que o módulo principal não precisa deles;
- `go list -deps ./...` carregou apenas `cosmossdk.io/log/v2` e `github.com/cosmos/cosmos-sdk/store/v2/...`;
- `go list -m all` ainda enumerou módulos v1 dormentes declarados por algum `go.mod` transitivo.

Logo, o gate correto é ausência de **pacotes v1 carregados** em `go list -deps` mais compilação readonly. Proibir qualquer linha v1 em `go list -m all` produziria falso positivo.

## Comparação das opções

| Opção | Cobertura | Esforço inicial | Manutenção | Risco | Veredito |
|---|---|---:|---:|---|---|
| Pin do upstream `/v11` por SHA | preserva toda a suíte, inclusive in-process/race | 1–2 dias incluindo QA | baixa/média até sair tag | pseudo-version sem release; pequenas APIs novas | **recomendada agora** |
| Fork temporário de `/v11` | preserva toda a suíte | 2–4 dias | alta: sync, security e tags próprios | divergência silenciosa do upstream | reservar como fallback |
| Black-box próprio com Docker/Compose | valida o binário e chains reais | 2–4 semanas | alta: lifecycle, funding, polling, reports e flakes próprios | perde conformance canônica e race in-process | não substituir a suíte |
| Manter `/v10` e forçar Store v2 com `replace` | nenhuma confiável | indefinido | muito alta | tipos incompatíveis; não é um problema de seleção de versão | rejeitada |

Um smoke test black-box mínimo pode ser acrescentado no futuro para instalação/distribuição, mas não substitui `interchaintest`: hoje os testes exercitam tanto o binário Docker quanto o relayer in-process, além de conformance, fee middleware, ICA, misbehaviour, filtros e concorrência.

## Implementação recomendada

1. Fixar exatamente o pseudo-version `/v11`; nunca usar `@main` em CI.
2. Aplicar as adaptações acima e manter o `replace github.com/cosmos/relayer/v2 => ../` relativo.
3. Colocar `./interchaintest` em `go.work` junto de `.`. Isso faz os targets atuais `cd interchaintest && go test ...` funcionarem e preserva o boundary dos unit tests raiz.
4. Fixar `moby/go-archive v0.1.0` com comentário sobre Docker 28.5.2, para evitar que `go mod tidy` selecione v0.2.0.
5. Executar em CI os dois modos abaixo: workspace e módulo isolado.
6. Abrir uma tarefa de renovação: substituir o pseudo-version pela primeira tag oficial `/v11` após repetir os mesmos gates.
7. Tratar runtime como marco separado: compilar contra IBC-Go v11 não comprova uma chain SDK 0.54/IBC v11. O aceite requer pelo menos um cenário contra uma imagem v11.2.0 construída de fonte ou publicada e verificada.

## Gates de aceite

```bash
# Módulo isolado: prova que o go.mod aninhado é autoconsistente.
cd interchaintest
GOWORK=off go mod verify
GOWORK=off go test -mod=readonly -run '^$' ./...
GOWORK=off go test -mod=readonly -list '^TestScenario' ./...

# Workspace: prova que os targets Make existentes não são mascarados por go.work.
go work sync
cd interchaintest
go test -mod=readonly -run '^$' ./...

# O build não pode carregar pacotes Store/Log v1.
if GOWORK=off go list -deps ./... | grep -Eq '^cosmossdk.io/store(/|$)|^cosmossdk.io/log$'; then
  exit 1
fi

# Gate de integração real, separado da compilação.
go test -race -v -run '<cenario SDK0.54 + IBC-Go v11.2.0>' .
```

## Condição para acionar o fork

Criar um fork somente se uma destas condições ocorrer:

- a primeira tag `/v11` remover uma API necessária e o upstream rejeitar/custar a aceitar a correção;
- uma correção necessária não puder ser expressa no consumidor por pin/adapter;
- o SHA fixado deixar de ser reproduzível ou apresentar vulnerabilidade sem correção upstream.

Até lá, o fork adicionaria governança e dívida sem resolver um bloqueio existente.
