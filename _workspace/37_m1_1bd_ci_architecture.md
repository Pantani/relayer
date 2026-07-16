# M1.1b-d — arquitetura de CI e separação black-box/white-box

Data: 2026-07-15  
Papel: `integration-qa`  
Escopo desta auditoria: `.github/workflows/interchaintest.yml`, `Makefile`,
`go.work`, `Dockerfile`, `local.Dockerfile` e `interchaintest/`. Nenhum arquivo
de produção foi alterado por esta análise.

> **Atualização oficial:** a recomendação original abaixo foi feita contra
> `cosmos/interchaintest/v10`. Ela foi superada pelo adendo **“Interchaintest
> /v11 oficial”** ao final deste documento. Com o pin oficial `/v11` no SHA
> `1a8c536`, a rota primária é restaurar `./interchaintest` no `go.work`; não
> criar um módulo black-box v10 e não manter o white-box propositalmente
> vermelho. A seção original permanece apenas como registro do diagnóstico do
> grafo v10.

## Veredito

O menor corte seguro é criar um módulo Go **black-box isolado** que nunca
importe `github.com/cosmos/relayer/v2` e consuma uma imagem do `rly` construída
uma única vez pelo módulo raiz. O módulo atual deve continuar explicitamente
classificado como **white-box** enquanto importa `cmd`, `relayer`, `processor`,
providers e o adapter in-process.

O módulo black-box **não pode ser incluído em `go.work`**: o workspace unificaria
o grafo oficial do interchaintest v10 (SDK 0.53/Store v1/IBC v10) com o grafo do
relayer (SDK 0.54/Store v2/IBC v11) e recriaria o conflito que se pretende
isolar. A fronteira correta é processo/Docker, não MVS.

O corte não autoriza esconder o gate white-box. Durante a transição devem
existir dois checks requeridos:

1. `interchaintest-blackbox`, que pode ficar verde já com o binário atual;
2. `interchaintest-whitebox-contract`, que permanece vermelho até um fork do
   framework compatível com SDK 0.54 ou até os últimos testes internos serem
   portados para testes do módulo raiz.

Não usar `continue-on-error`, `|| true`, exclusão silenciosa de cenários ou a
remoção do job white-box.

## Evidência do estado atual

### Falha 1 — workspace

O `go.work` contém somente:

```text
go 1.25.9
use .
```

Todos os targets atuais fazem `cd interchaintest && go test ...` sem desligar o
workspace. A execução real falha antes de compilar:

```text
go test -run '^$' ./...
FAIL: pattern ./...: directory prefix . does not contain modules listed in
      go.work or their selected dependencies
```

A mesma falha foi reexecutada e reproduzida sem alteração de estado.

Logo, `events`, `legacy`, `multiple-paths`, `misbehaviour`, `fee-middleware`,
`fee-grant` e `prepare-scenario-matrix` estão quebrados na primeira camada.

### Falha 2 — grafo isolado

Reexecução com o workspace desligado:

```text
GOWORK=off go test -mod=mod -run '^$' ./...
FAIL cosmossdk.io/x/upgrade/types
```

Erro de fronteira:

```text
cosmossdk.io/store/types.CommitMultiStore
does not implement
github.com/cosmos/cosmos-sdk/store/v2/types.CommitMultiStore
```

O teste isolado também foi reexecutado e reproduziu o mesmo erro nos pacotes
raiz e `stride`.

O grafo confirma a causa:

```text
github.com/cosmos/interchaintest/v10@v10.0.1
  -> github.com/cosmos/cosmos-sdk@v0.53.4
  -> github.com/cometbft/cometbft@v0.38.19
  -> github.com/cosmos/ibc-go/v10@v10.3.0
  -> cosmossdk.io/store@v1.1.2
  -> cosmossdk.io/x/upgrade@v0.2.0

github.com/cosmos/relayer/v2
  -> github.com/cosmos/cosmos-sdk@v0.54.3
  -> github.com/cometbft/cometbft@v0.39.3
  -> github.com/cosmos/ibc-go/v11@v11.2.0
  -> github.com/cosmos/cosmos-sdk/store/v2@v2.0.0
```

A troca Strangelove v8 -> Cosmos interchaintest v10 limpa a procedência do
framework, mas não o torna compatível com a família SDK 0.54.

### Cobertura real do workflow atual

Os seis jobs fixos executam o adapter local in-process; eles não validam uma
imagem pré-compilada. Os testes Docker de conformance, backup RPC, múltiplos
canais e override não têm jobs dedicados. Apenas dois testes Docker podem
entrar indiretamente na matriz `TestScenario*`, mas a preparação da matriz
falha ao compilar o pacote inteiro.

Portanto, no estado atual, não há um gate black-box executável.

### Build Docker divergente

O `Dockerfile` de release já fixa `golang:1.25.9-alpine3.22`, mas
`BuildRelayerImage` usa `local.Dockerfile`, ainda baseado em
`golang:1-alpine3.17`. Assim, mesmo quando um teste Docker é executado, o
toolchain não é determinado pelo mesmo arquivo usado no release. O corte deve
fixar `local.Dockerfile` em Go 1.25.9 e registrar o build-info do binário
extraído da imagem.

## Matriz dos jobs atuais

| job/target | teste selecionado | fronteira atual | binário pré-compilado hoje? | decisão |
|---|---|---|---:|---|
| `events` / `interchaintest-events` | `TestRelayerEventProcessor` | `NewRelayerFactory`, `cmd`, `relayertest`, `processor` | não | manter white-box |
| `legacy` / `interchaintest-legacy` | `TestRelayerLegacyProcessor` | adapter local in-process | não | manter white-box |
| `multiple-paths` | `TestRelayerMultiplePathsSingleProcess` | `NewRelayer` local | não | converter para black-box depois do corte inicial |
| `misbehaviour` | `TestRelayerMisbehaviourDetection` | factory local; ataque é feito pela chain | não | conversão black-box pequena |
| `fee-middleware` | `TestRelayerFeeMiddleware` | factory local | não | conversão black-box pequena |
| `fee-grant` | `TestRelayerFeeGrant` | `processor.PathProcMessageCollector`, provider e consensus client internos | não | white-box profundo |
| `prepare-scenario-matrix` | todo o pacote | precisa compilar white-box + black-box juntos | não | substituir por matriz estática ou módulo black-box isolado |
| `scenarios` | todos `TestScenario*` | mistura factories locais e Docker | parcial | separar por módulo antes de gerar a matriz |

## Matriz dos testes

### Grupo A — podem rodar com imagem pré-compilada após troca mecânica do helper

Esses testes já usam `interchaintest.NewBuiltinRelayerFactory` e executam o
`rly` via Docker. A única dependência local é a construção repetida da imagem,
exceto onde anotado.

| teste | arquivo | alteração mínima |
|---|---|---|
| `TestRelayerDockerEventProcessor` | `ibc_test.go` | mover para black-box e trocar `BuildRelayerImage` por `imageFromEnv` |
| `TestRelayerDockerLegacyProcessor` | `ibc_test.go` | idem |
| `TestBackupRpcs` | `backup_rpc_test.go` | trocar helper e usar tipos IBC v10 do harness |
| `TestMultipleChannelsOneConnection` | `multi_channel_test.go` | trocar helper e usar tipos IBC v10 do harness |
| `TestScenarioClientThresholdUpdate` | `client_threshold_test.go` | trocar helper; usar tipos IBC v10 somente no harness |
| `TestScenarioClientTrustingPeriodUpdate` | `client_threshold_test.go` | idem |
| `TestClientOverrideFlag` | `relayer_override_test.go` | trocar helper e substituir `cmd.Config` por struct JSON local mínima |

O módulo black-box pode usar ibc-go v10 para os tipos do **harness**, pois esse
é o grafo do interchaintest v10. O software sob teste continua sendo o binário
IBC v11; isto deve ser provado pelo build-info e por chains IBC v11 em runtime.
Não importar v11 no harness v10 apenas para aparentar uma versão mais nova.

### Grupo B — conversíveis para black-box usando apenas CLI pública

Estes cenários usam a factory local, mas não dependem necessariamente de
estado interno do processo. Devem ser portados em um segundo passo, preservando
nome e assertion funcional:

| teste | acoplamento a remover |
|---|---|
| `TestMemoAndReceiverLimit` | acesso concreto a `Relayer.Sys()` e `cmd.Config`; escrever/consultar configuração pela CLI ou struct JSON local |
| `TestRelayerMultiplePathsSingleProcess` | `NewRelayer`; substituir pela factory Docker única |
| `TestLocalhost_TokenTransfers` | factory local; atualizar chain fixture v8 para IBC v11 |
| `TestLocalhost_InterchainAccounts` | factory local; atualizar chain fixture v8 beta |
| `TestRelayerFeeMiddleware` | factory local |
| `TestScenarioTendermint37Boundary` | factory local |
| `TestScenarioPathFilterAllow` | factory local e constantes `processor`; usar valores públicos de config |
| `TestScenarioPathFilterDeny` | idem |
| `TestRelayerMisbehaviourDetection` | factory local; assertions já são on-chain |
| `TestScenarioInterchainAccounts` | factory local |
| `TestScenarioICAChannelClose` | factory local |
| `TestScenarioStrideICAandICQ` | factory local e codec `rlystride`; consultar/decode via chain CLI ou JSON bruto |

### Grupo C — white-box de verdade

| teste | razão | destino recomendado |
|---|---|---|
| `TestRelayerInProcess` | testa o adapter in-process por definição | fork SDK0.54 do interchaintest ou remoção somente após equivalência em testes raiz |
| `TestRelayerEventProcessor` | in-process/race do processor | módulo raiz com fixtures, ou fork |
| `TestRelayerLegacyProcessor` | in-process/race do processor | módulo raiz com fixtures, ou fork |
| `TestRelayerFeeGrant` | observa `PathProcMessageCollector`, provider e tx internos | manter white-box; separar a parte E2E em black-box |
| `TestRelayerFeeGrantExternal` | mesmo acoplamento interno | idem |
| `TestAccCacheBugfix` | testa diretamente comportamento do SDK, não o binário | mover para unit test do provider Cosmos no módulo raiz |

## Arquitetura proposta

```text
root go.mod: SDK0.54 / IBC11 / Store v2
        |
        +-- build uma vez --> imagem rly-under-test:${SHA}
        |                         |
        |                         +--> build-info obrigatório
        |                         +--> artifact docker save
        |
        +-- testes raiz/white-box sem interchaintest v10

interchaintest/blackbox/go.mod: interchaintest v10 / SDK0.53 / Store v1
        |
        +-- GOWORK=off
        +-- zero imports github.com/cosmos/relayer/v2
        +-- carrega artifact rly-under-test:${SHA}
        +-- chains SDK0.54/IBC11 em Docker

interchaintest/go.mod: contrato white-box legado
        |
        +-- GOWORK=off
        +-- check requerido e explicitamente vermelho até fork/port
```

### Estrutura mínima

```text
interchaintest/
  blackbox/
    go.mod
    go.sum
    image_test.go
    conformance_test.go
    backup_rpc_test.go
    multi_channel_test.go
    client_threshold_test.go
    relayer_override_test.go
  ... white-box existente ...
```

`image_test.go` deve apenas ler uma referência externa:

```go
func imageFromEnv(t *testing.T) (repository, tag string) {
	t.Helper()
	repository = os.Getenv("RLY_IMAGE_REPOSITORY")
	tag = os.Getenv("RLY_IMAGE_TAG")
	require.NotEmpty(t, repository)
	require.NotEmpty(t, tag)
	return repository, tag
}
```

O helper não deve executar `docker build`, importar o repositório raiz ou
usar `runtime.Caller` para arquivar toda a árvore.

## Diff proposto — não aplicado

O trecho abaixo descreve o menor patch de orquestração. Os movimentos de teste
do Grupo A são parte do mesmo change set.

```diff
diff --git a/go.work b/go.work
@@
 go 1.25.9
 use .
+// Deliberadamente não incluir interchaintest/blackbox: ele usa o grafo
+// SDK0.53/Store-v1 do framework, isolado por processo.

diff --git a/local.Dockerfile b/local.Dockerfile
@@
-FROM golang:1-alpine3.17 AS build-env
+FROM golang:1.25.9-alpine3.22 AS build-env

diff --git a/Makefile b/Makefile
@@
+RLY_IMAGE_REPOSITORY ?= relayer-under-test
+RLY_IMAGE_TAG ?= local
+ICT_ENV := GOWORK=off
+
+interchaintest-image:
+	docker build -f local.Dockerfile \
+	  -t $(RLY_IMAGE_REPOSITORY):$(RLY_IMAGE_TAG) .
+
+interchaintest-blackbox-compile:
+	cd interchaintest/blackbox && $(ICT_ENV) \
+	  go test -mod=readonly -run '^$$' ./...
+
+interchaintest-blackbox:
+	cd interchaintest/blackbox && $(ICT_ENV) \
+	  RLY_IMAGE_REPOSITORY=$(RLY_IMAGE_REPOSITORY) \
+	  RLY_IMAGE_TAG=$(RLY_IMAGE_TAG) \
+	  go test -mod=readonly -timeout 30m -race -v ./...
+
+interchaintest-whitebox-contract:
+	cd interchaintest && $(ICT_ENV) \
+	  go test -mod=readonly -run '^$$' ./...
@@
-	cd interchaintest && go test ...
+	cd interchaintest && $(ICT_ENV) go test ...
```

O comentário dentro de `go.work` pode ser movido para o README caso a política
do projeto prefira manter o arquivo mínimo; a decisão de isolamento deve ficar
documentada em algum lugar versionado.

Workflow proposto:

```diff
diff --git a/.github/workflows/interchaintest.yml b/.github/workflows/interchaintest.yml
@@
+  build-rly-image:
+    runs-on: ubuntu-latest
+    outputs:
+      image-tag: ${{ steps.meta.outputs.tag }}
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v6
+        with:
+          go-version-file: .go-version
+      - id: meta
+        run: echo "tag=${GITHUB_SHA}" >> "$GITHUB_OUTPUT"
+      - name: Build relayer image once
+        run: docker build -f local.Dockerfile \
+             -t "relayer-under-test:${GITHUB_SHA}" .
+      - name: Prove binary dependency family
+        run: |
+          set -euo pipefail
+          cid=$(docker create "relayer-under-test:${GITHUB_SHA}" /bin/rly version)
+          docker cp "${cid}:/bin/rly" ./rly-under-test
+          docker rm "${cid}"
+          go version -m ./rly-under-test | tee rly-build-info.txt
+          grep -Eq 'github.com/cosmos/cosmos-sdk[[:space:]]+v0\.54\.3' rly-build-info.txt
+          grep -Eq 'github.com/cometbft/cometbft[[:space:]]+v0\.39\.3' rly-build-info.txt
+          grep -Eq 'github.com/cosmos/ibc-go/v11[[:space:]]+v11\.2\.0' rly-build-info.txt
+          grep -Eq 'github.com/cosmos/cosmos-sdk/store/v2[[:space:]]+v2\.0\.0' rly-build-info.txt
+          if grep -Eq 'github.com/cosmos/ibc-go/v(8|10)[[:space:]]' rly-build-info.txt; then exit 1; fi
+      - name: Export image
+        run: docker save "relayer-under-test:${GITHUB_SHA}" | gzip > rly-image.tar.gz
+      - uses: actions/upload-artifact@v4
+        with:
+          name: rly-image
+          path: |
+            rly-image.tar.gz
+            rly-build-info.txt
+
+  blackbox-compile:
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v6
+        with:
+          go-version-file: .go-version
+      - run: cd interchaintest/blackbox && \
+             GOWORK=off go test -mod=readonly -run '^$' ./...
+      - name: Assert no local relayer import
+        run: |
+          set -euo pipefail
+          cd interchaintest/blackbox
+          deps=$(GOWORK=off go list -deps ./...)
+          if grep -F 'github.com/cosmos/relayer/v2/' <<<"${deps}"; then exit 1; fi
+
+  blackbox:
+    needs: [build-rly-image, blackbox-compile]
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v6
+        with:
+          go-version-file: .go-version
+      - uses: actions/download-artifact@v4
+        with:
+          name: rly-image
+      - run: gunzip -c rly-image.tar.gz | docker load
+      - run: cd interchaintest/blackbox && \
+             GOWORK=off \
+             RLY_IMAGE_REPOSITORY=relayer-under-test \
+             RLY_IMAGE_TAG=${GITHUB_SHA} \
+             go test -mod=readonly -timeout 30m -race -v ./...
+
+  whitebox-contract:
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v6
+        with:
+          go-version-file: .go-version
+      - run: cd interchaintest && \
+             GOWORK=off go test -mod=readonly -run '^$' ./...
```

`whitebox-contract` é intencionalmente um check requerido. O patch de
separação não deve ser anunciado como lote concluído enquanto esse check
continuar vermelho.

Para matriz dinâmica, se ela for mantida, usar `set -euo pipefail` e gerar a
lista somente dentro de `interchaintest/blackbox`. O pipeline atual com vários
`grep -v` não ativa `pipefail` e mistura mensagens de inicialização com nomes
de testes. Uma matriz estática inicial para os sete testes do Grupo A é menor
e mais auditável.

## Prova runtime de SDK 0.54 / IBC v11

O build local atual já prova a família do **binário**:

```text
./build/rly version --json
cosmos-sdk: v0.54.3

go version -m ./build/rly
github.com/cosmos/cosmos-sdk              v0.54.3
github.com/cometbft/cometbft              v0.39.3
github.com/cosmos/ibc-go/v11              v11.2.0
github.com/cosmos/cosmos-sdk/store/v2     v2.0.0
```

Isto ainda não prova interoperabilidade on-chain. O primeiro cenário
black-box deve usar duas chains `ibc-go-simd` da família nova e validar:

1. `version --long` das duas chains contém SDK 0.54, Store v2 e IBC v11;
2. o binário cria clients, connection e channel Classic;
3. uma transferência ICS-20 produz `send_packet`, `recv_packet` e
   acknowledgement;
4. o saldo e o denom resultante são verificados nas duas pontas;
5. o mesmo smoke roda com processor `events` e `legacy`.

Imagem oficial verificada ao vivo nesta auditoria:

```text
ghcr.io/cosmos/ibc-go-simd:v11.1.0
digest sha256:c863331a3836474da4337beff12829b6f120bb1883f9c420e2becceca0fe1891
github.com/cosmos/cosmos-sdk@v0.54.0
github.com/cosmos/cosmos-sdk/store/v2@v2.0.0
github.com/cosmos/ibc-go/v11@v11.0.0
github.com/cometbft/cometbft@v0.39.0
```

`v11.2.0` não possuía manifest publicado no registry consultado; portanto não
se deve inventar esse tag. Para provar a família, pin `v11.1.0` por digest. Se
o critério exigir chain exatamente v11.2.0, construa `simd` a partir do tag
oficial v11.2.0 em um job separado e registre seu build-info.

Os fixtures atuais `ibc-go-simd:v8.0.0` e `v8.0.0-beta.1`, além de Gaia 14 e
Osmosis 22, preservam compatibilidade histórica, mas não provam SDK 0.54/IBC
v11. Eles devem permanecer como matriz Classic antiga, não como evidência da
família nova.

Esta prova cobre Classic sobre uma chain IBC v11. Ela **não** prova IBC v2;
IBC v2 requer cenários e assertions próprios em lote posterior.

## Critérios de aceite

### Corte black-box

- [ ] `interchaintest/blackbox/go.mod` não contém `replace` ou `require` para
      `github.com/cosmos/relayer/v2`.
- [ ] `GOWORK=off go test -mod=readonly -run '^$' ./...` passa no módulo
      black-box.
- [ ] `GOWORK=off go list -deps ./...` não lista nenhum pacote do relayer.
- [ ] Os sete testes do Grupo A mantêm suas assertions e nomes rastreáveis.
- [ ] Nenhum teste chama `BuildRelayerImage`; todos consomem o mesmo artifact.
- [ ] A imagem é identificada pelo SHA do commit e construída uma única vez.
- [ ] `go version -m` prova SDK 0.54.3, Comet 0.39.3, IBC v11.2.0 e Store v2.
- [ ] O check rejeita IBC v10/v8 dentro do binário.
- [ ] A chain SDK0.54/IBC11 é pinada por digest e seu `version --long` é salvo.
- [ ] Handshake, ICS-20 e acknowledgement passam em chain v11 com events e
      legacy.

### Preservação de cobertura

- [ ] Existe inventário versionado `teste original -> black-box | root |
      white-box` sem linha órfã.
- [ ] O número de cenários removidos da suíte original é zero.
- [ ] O workflow não contém `continue-on-error`, `|| true` ou filtros por
      versão que pulem o caso SDK0.54/IBC11.
- [ ] `whitebox-contract` permanece required até ficar verde.
- [ ] O lote não é marcado como completo apenas porque black-box ficou verde.

### Fechamento white-box

Uma das alternativas precisa ser concluída:

1. fork temporário de `cosmos/interchaintest` alinhado a SDK 0.54, Comet 0.39,
   IBC v11, Store v2 e Log v2, com commit pinado; ou
2. conversão de todos os cenários do Grupo B para black-box e migração dos
   testes do Grupo C para o módulo raiz sem dependência de interchaintest.

Depois disso:

- [ ] `GOWORK=off go test -mod=readonly -run '^$' ./...` passa no módulo
      white-box remanescente ou o módulo deixa de existir sem perder testes.
- [ ] Todos os jobs do workflow estão verdes e requeridos.
- [ ] A skill QA pode afirmar que todos os módulos declarados e todos os
      módulos isolados inventariados possuem gate explícito e reproduzível.

## Assertions QA

| assertion | resultado atual | evidência | severidade | ação |
|---|---|---|---:|---|
| targets Make executam o nested module | reprovado | `go.work` exclui o módulo e Make não usa `GOWORK=off` | P0 | corrigir todos os targets |
| interchaintest isolado compila | reprovado | Store v1 `CommitMultiStore` vs Store v2 | P0 | split ou fork |
| existe gate black-box real | reprovado | workflow usa factories in-process; matrix não compila | P0 | módulo black-box + artifact |
| imagem do relayer tem proveniência SDK0.54/IBC11 | verificável, não gateado | build-info local confirma versões | P1 | tornar grep obrigatório em CI |
| chain runtime prova SDK0.54/IBC11 | não verificado pelo repo | fixtures atuais são antigas; imagem v11.1 disponível | P1 | smoke v11 pinado por digest |
| cenários antigos continuam visíveis | parcialmente aprovado | arquivos permanecem, mas nenhum job consegue listá-los | P1 | inventário e matriz por fronteira |
| gate white-box não é escondido | reprovado no estado atual | módulo saiu de `go.work`, mas workflow não compensa corretamente | P0 | check required com `GOWORK=off` |

## Conclusão operacional

O próximo patch deve implementar somente o corte do Grupo A e o artifact de
imagem, deixando `whitebox-contract` visivelmente requerido. Depois do primeiro
smoke SDK0.54/IBC11 verde, escolher entre fork e conversão dos Grupos B/C. Essa
sequência entrega sinal black-box útil sem transformar uma incompatibilidade de
grafo em falsa aprovação.

---

## Adendo — Interchaintest /v11 oficial

Data da reavaliação: 2026-07-15  
Status: **substitui o veredito e a conclusão operacional anteriores**

### Novo fato primário

O commit oficial
[`cosmos/interchaintest@1a8c536981a8`](https://github.com/cosmos/interchaintest/commit/1a8c536981a88e3d7684f3ebc2430ef424f3ee8c)
é um módulo `/v11` da mesma família do relayer:

```text
module:  github.com/cosmos/interchaintest/v11
version: v11.0.0-20260507171724-1a8c536981a8
Go:      1.25.9
SDK:     0.54.0
Comet:   0.39.0
Store:   store/v2 2.0.0
IBC:     ibc-go/v11 11.0.0
Log:     log/v2 2.1.0
```

Proveniência verificada com `go list -m -json` e `go mod download -json`:

```text
Origin URL:  https://github.com/cosmos/interchaintest
Origin Hash: 1a8c536981a88e3d7684f3ebc2430ef424f3ee8c
module sum:  h1:ScPmept2RRYJcGzaEMnjgAGyomyUq+x8R0pbjg9a6Io=
go.mod sum:  h1:r3hfwwOD54Q1CsSQLaiFtvJWyzw2bqew2mbmhjufmSA=
```

O módulo raiz do relayer eleva os patches por MVS para SDK `0.54.3`, Comet
`0.39.3` e IBC `v11.2.0`, mantendo Store v2 `2.0.0` e Log v2 `2.1.0`.
Portanto a incompatibilidade estrutural do interchaintest v10 deixou de
existir.

### Veredito revisado

Sim: `go.work` pode e deve voltar a declarar os dois módulos:

```go.work
go 1.25.9

use (
	.
	./interchaintest
)
```

A separação black-box por outro módulo Go não é mais requisito arquitetural.
Testes in-process e Docker podem permanecer no mesmo `interchaintest/`, usando
o mesmo grafo SDK0.54/IBC11. Uma imagem pré-compilada única continua sendo um
hardening/otimização válido, mas não é blocker para corrigir o CI.

O pseudo-version deve ser pinado por inteiro. Não usar `@main`, um hash curto
em arquivo versionado ou atualização automática sem compile contract.

### Prova independente

Uma cópia temporária do módulo foi adaptada para `/v11` e ligada ao módulo
raiz atual. Os seguintes checks passaram:

```text
GOWORK=off go mod verify                              PASS
GOWORK=off go test -mod=mod -run '^$' ./...          PASS (raiz + stride)
workspace go test -mod=readonly -run '^$' ./...      PASS (relayer)
workspace go test -mod=readonly -run '^$' ./...      PASS (interchaintest)
workspace go build -mod=readonly ./...                PASS (relayer)
workspace go build -mod=readonly ./...                PASS (interchaintest)
```

O grafo efetivamente compilado contém Store v2 e IBC v11. `go mod why`
confirmou que `cosmossdk.io/store`, `cosmossdk.io/log` e
`cosmossdk.io/x/upgrade` não são necessários pelo módulo principal.

`go list -m all` isoladamente não deve ser usado como gate negativo: ele pode
enumerar módulos Store/Log v1 dormentes trazidos por go.mod de dependências.
O gate correto combina `go list -deps -test`, `go mod why`, versão selecionada
e compilação real.

### Adaptações de API observadas

A troca `/v10 -> /v11` exige mudanças focais, não um fork:

- `ibc.Relayer.AddKey` recebe também `signingAlgorithm`;
- `UpdatePath` recebe `ibc.PathUpdateOptions`;
- o adapter local implementa `CreateClient` e `ContainerImage`;
- heights do framework passam a `int64` em logs e helpers;
- misbehaviour usa `ClientState.LatestHeight` e o mock PV do Comet 0.39;
- Docker/Moby v28 usa os tipos atuais de archive/image removal;
- os imports do framework passam integralmente a
  `github.com/cosmos/interchaintest/v11/...`;
- denom deve continuar usando a API atual do ibc-go v11
  (`ExtractDenomFromPath`/`Denom`), sem reintroduzir `DenomTrace` como modelo.

Essas mudanças devem continuar sujeitas ao limite de complexidade máximo 9.
Em particular, `UpdatePath` e o encoder de opções de client devem ser divididos
em helpers se o adapter ultrapassar o contrato.

### Revisão de `local.Dockerfile`

O build usado por `BuildRelayerImage` deve começar com:

```dockerfile
FROM golang:1.25.9-alpine3.22 AS build-env
```

Isso alinha o Docker local ao `.go-version` e ao `Dockerfile` de release. O
pin corrige o finding anterior de `golang:1-alpine3.17`. O job Docker ainda
deve extrair `/bin/rly` e registrar `go version -m` para provar SDK0.54.3,
Comet0.39.3, IBC11.2.0 e Store v2.

### Jobs revisados

| job | mudança com `/v11` |
|---|---|
| novo `module-contract` | obrigatório; compila raiz e interchaintest no workspace e repete o nested module com `GOWORK=off` |
| `events` | mantém `TestRelayerEventProcessor`; passa a depender de `module-contract` |
| `legacy` | mantém `TestRelayerLegacyProcessor`; depende de `module-contract` |
| `multiple-paths` | mantém o teste in-process; depende de `module-contract` |
| `misbehaviour` | mantém o job, agora com tipos Comet0.39/IBC11 |
| `fee-middleware` | mantém o job no mesmo módulo `/v11` |
| `fee-grant` | mantém o white-box profundo; não precisa fork |
| `prepare-scenario-matrix` | volta a ser viável; adicionar compile readonly e `pipefail` |
| `scenarios` | mantém a matriz, ancora e cita o nome do teste com segurança |
| novo `docker-events` | executa `TestRelayerDockerEventProcessor` e prova a imagem real |
| novo `docker-legacy` | executa `TestRelayerDockerLegacyProcessor` |
| novo `localhost` | executa os dois testes localhost, que não entram em `TestScenario*` |

Backup RPC, múltiplos canais e client override continuam fora dos jobs atuais.
Eles devem ser uma matriz Docker posterior; isto é lacuna de cobertura, não
bloqueio do grafo `/v11`.

### Diff de CI/go.work proposto — substitui o diff v10 anterior

```diff
diff --git a/go.work b/go.work
@@
 go 1.25.9
-use .
+use (
+  .
+  ./interchaintest
+)

diff --git a/Makefile b/Makefile
@@
 interchaintest:
-	cd interchaintest && go test -race -v -run TestRelayerInProcess .
+	cd interchaintest && go test -mod=readonly -race -v \
+	  -run '^TestRelayerInProcess$' .
@@
+interchaintest-compile:
+	go test -mod=readonly -run '^$$' ./...
+	cd interchaintest && go test -mod=readonly -run '^$$' ./...
+	cd interchaintest && GOWORK=off go mod verify
+	cd interchaintest && GOWORK=off go test -mod=readonly -run '^$$' ./...
+
+interchaintest-localhost:
+	cd interchaintest && go test -mod=readonly -race -v \
+	  -run '^TestLocalhost_' .
```

Aplicar `-mod=readonly`, timeout e regex ancorada da mesma forma a todos os
targets interchaintest existentes.

Contrato central do workflow:

```yaml
jobs:
  module-contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v6
        with:
          go-version-file: .go-version
      - name: Verify selected family
        run: |
          set -euo pipefail
          go test -mod=readonly -run '^$' ./...
          go build -mod=readonly ./...
          cd interchaintest
          go test -mod=readonly -run '^$' ./...
          go build -mod=readonly ./...
          GOWORK=off go mod verify
          GOWORK=off go test -mod=readonly -run '^$' ./...
          GOWORK=off go build -mod=readonly ./...
          selected=$(GOWORK=off go list -m \
            github.com/cosmos/interchaintest/v11 \
            github.com/cosmos/cosmos-sdk \
            github.com/cometbft/cometbft \
            github.com/cosmos/ibc-go/v11 \
            github.com/cosmos/cosmos-sdk/store/v2)
          grep -F 'v11.0.0-20260507171724-1a8c536981a8' <<<"${selected}"
          grep -F 'github.com/cosmos/cosmos-sdk v0.54.3' <<<"${selected}"
          grep -F 'github.com/cometbft/cometbft v0.39.3' <<<"${selected}"
          grep -F 'github.com/cosmos/ibc-go/v11 v11.2.0' <<<"${selected}"
          grep -F 'github.com/cosmos/cosmos-sdk/store/v2 v2.0.0' <<<"${selected}"
          deps=$(GOWORK=off go list -deps -test ./...)
          if grep -Eq 'cosmossdk.io/x/upgrade|github.com/cosmos/ibc-go/v10' \
            <<<"${deps}"; then exit 1; fi
```

Todos os jobs de teste devem declarar `needs: module-contract`.

Descoberta segura da matriz:

```yaml
- name: Generate scenario matrix
  id: set-matrix
  shell: bash
  run: |
    set -euo pipefail
    cd interchaintest
    go test -mod=readonly -run '^$' ./...
    output=$(go test -mod=readonly -list '^TestScenario' ./...)
    matrix=$(jq -Rsc \
      'split("\n") | map(select(startswith("TestScenario")))' \
      <<<"${output}")
    test "${matrix}" != '[]'
    echo "matrix=${matrix}" >> "${GITHUB_OUTPUT}"
```

Execução da matriz:

```yaml
- run: |
    cd interchaintest
    go test -mod=readonly -timeout 30m -race -v \
      -run '^${{ matrix.test }}$' ./...
```

### Gates de qualidade adicionais

- O workflow do interchaintest não chama hoje `make complexity`. Adicionar um
  ratchet requerido com baseline temporário `83 cyclomatic / 137 cognitive /
  union 141` e máximo 9 para qualquer função nova/alterada; o objetivo final
  continua zero violações globais.
- Não usar ausência de Store/Log v1 em `go list -m all` como condição. Exigir
  compile real, `go list -deps -test` sem `x/upgrade`/IBC10 e `go mod why` para
  investigar qualquer edge antigo enumerado.
- Pin pseudo-version, SHA de origem e checksums devem constar no relatório de
  dependências e em política de atualização manual/Dependabot controlada.
- Os fixtures runtime v11/SDK0.54 e IBC v2 são gates separados: compilar o
  framework `/v11` não prova comportamento IBC v2.

### Critério de aceite revisado

- [ ] `go.work` lista raiz e `./interchaintest`.
- [ ] o pin é exatamente
      `v11.0.0-20260507171724-1a8c536981a8`.
- [ ] não existem imports manuscritos `interchaintest/v10`.
- [ ] raiz e nested module passam compile/test vazio e build no workspace.
- [ ] nested module repete `go mod verify`, compile/test vazio e build com
      `GOWORK=off -mod=readonly`.
- [ ] `go list -deps -test` não seleciona `cosmossdk.io/x/upgrade` nem
      `ibc-go/v10`.
- [ ] todos os jobs atuais dependem do compile contract.
- [ ] matriz falha se discovery falhar ou produzir lista vazia.
- [ ] `local.Dockerfile` fixa Go `1.25.9-alpine3.22`.
- [ ] jobs Docker events/legacy e localhost existem.
- [ ] runtime SDK0.54/IBC11 é verificado; IBC v2 não é inferido desse smoke.
- [ ] complexidade incremental permanece com scores máximos 9.

### Conclusão revisada

A rota oficial `/v11` elimina a necessidade imediata de fork e torna o módulo
in-process novamente válido. A ação correta é migrar as APIs focais, restaurar
o workspace com os dois módulos e transformar a coerência em contrato duplo:
workspace mais execução isolada `GOWORK=off`. A proposta de módulo black-box
v10 permanece somente como alternativa histórica e não deve ser implementada
como arquitetura do lote M1.1b-d.
