# M1.1b-d — contrato de QA do interchaintest SDK 0.54

Data do snapshot: 2026-07-15 20:19:02 -03  
Branch: `Pantani/cx/m0-baseline`  
HEAD base: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Papel: Integration QA independente; nenhuma fonte de produção foi alterada.

## Parecer do snapshot de entrada

**REPROVADO antes da migração para interchaintest v11.** O módulo raiz compila,
mas isso é um falso verde para integração porque `go.work` declara somente `.`.
O módulo `interchaintest`, executado isoladamente, falha antes de listar qualquer
teste por misturar Store v1 do interchaintest v10 com Store v2 do SDK 0.54.
Sem `GOWORK=off`, os targets reais do Makefile falham ainda antes porque o
subdiretório não pertence ao workspace ativo:

```text
pattern ./...: directory prefix . does not contain modules listed in go.work
or their selected dependencies
```

Desligar o workspace revela a segunda falha, de dependências:

A falha foi reproduzida duas vezes:

```text
cd interchaintest
GOWORK=off go test -mod=readonly -run '^$' ./...
GOWORK=off go test -count=1 -mod=readonly -run '^$' ./...

FAIL cosmossdk.io/x/upgrade@v0.2.0/types/storeloader.go
cosmossdk.io/store/types.CommitMultiStore não implementa
github.com/cosmos/cosmos-sdk/store/v2/types.CommitMultiStore
```

O comando usado pelo workflow para descobrir a matriz também falha:

```text
GOWORK=off go test -mod=readonly -list '^TestScenario' ./...
FAIL antes de produzir a lista
```

Há, entretanto, uma rota upstream sem fork. O commit
[`1a8c536981a8`](https://github.com/cosmos/interchaintest/commit/1a8c536981a88e3d7684f3ebc2430ef424f3ee8c)
é o módulo `github.com/cosmos/interchaintest/v11` e declara Go 1.25.9,
Cosmos SDK 0.54.0, CometBFT 0.39.0, Store v2.0.0, IBC-Go v11.0.0 e Log v2.
O pin reproduzível para este lote é:

```text
github.com/cosmos/interchaintest/v11
v11.0.0-20260507171724-1a8c536981a8
```

Um ensaio temporário do produtor, com esse pin e as adaptações de API, compilou
os dois pacotes de `interchaintest` sem carregar pacotes Store/Log v1. Essa
evidência deve ser repetida no working tree; não é aceite por si só. Como ainda
não há tag estável v11 do framework, o pseudo-version é um risco controlado,
não motivo para manter v10 nem para criar fork agora.

## Matriz de aceitação do sublote

| ID | Superfície | Gate obrigatório | Critério de aprovação | Estado do snapshot |
|---|---|---|---|---|
| A1 | Módulo raiz | `go test -mod=readonly -run '^$' ./...` e `go build -mod=readonly ./...` | Ambos passam sem atualizar `go.mod`/`go.sum` | **Compilação aprovada**; build deve ser repetido após a integração |
| A2 | Framework | Pin exato `/v11@v11.0.0-20260507171724-1a8c536981a8` | Nenhum import `/v10`; SHA e motivo do pseudo-version documentados | **Pendente no working tree**; ensaio temporário aprovado |
| A3 | Grafo isolado | Em `interchaintest`: `GOWORK=off go mod verify`, `go list -deps ./...`, `go mod why -m`, `go test -mod=readonly -run '^$' ./...`, `go build -mod=readonly ./...` | Todos passam; nenhum pacote Store/Log v1, IBC v8/v10 ou SDK pré-0.54 é carregado | **Reprovado com v10**; ensaio temporário v11 aprovado |
| A4 | Workspace | Restaurar `go.work` com `.` e `./interchaintest`; executar gates de raiz e integração com workspace ativo | Os dois módulos são visíveis e compilam; remover módulo do workspace não é aceite | **Reprovado**: `go.work` contém apenas `.` |
| A5 | CI unitária | Job de build executa teste race, build e ratchet de complexidade | Nenhum gate pode ser omitido por exclusão do `go.work` | **Reprovado**: não há complexity gate na CI |
| A6 | CI integração | Antes de criar a matriz, verificar módulo, compilar e listar cenários com `-mod=readonly` | Falha de compile termina o job; matriz vazia é erro explícito | **Reprovado**: comando atual falha com v10 |
| A7 | Imagem alvo | Construir `simd` do tag ibc-go v11.2.0, commit `cfc072e53eee42b2ab804cd4344ba610016f793c`, usando o Dockerfile oficial e `IBC_GO_VERSION=v11.2.0` | Log registra source SHA e digest da imagem; app executado declara SDK 0.54/IBC v11 | **Não verificado** |
| A8 | Classic v11 | Cenário com duas chains v11.2.0 executa Send -> Recv -> Ack nos processadores events e legacy | Mensagens on-chain e saldos são confirmados, não apenas ausência de erro | **Ausente** |
| A9 | Timeout v11 | Cenário produz packet expirado e confirma `MsgTimeout`/refund | Timeout height/timestamp e saldo final são assertados | **Ausente** |
| A10 | Localhost v11 | Executar token transfer localhost na chain v11.2.0 | Fluxo stateless funciona sem consulta a client state removido; duas transferências e denom final são confirmados | **Reprovado por imagem v8 atual** |
| A11 | Misbehaviour v11 | Cenário confirma update/misbehaviour pelo wire v11 | `MsgUpdateClient` aparece on-chain e o client é congelado/atualizado conforme o caso | **Não verificado em runtime v11** |
| A12 | Compatibilidade antiga | Manter ao menos um smoke Classic contra chain anterior suportada | O cenário é rotulado `legacy`; não pode ser confundido com cobertura v11 | **Parcial**, imagens antigas existem, mas não há matriz declarativa |
| A13 | IBC v2 | Teste de integração negativo configura path v2 e tenta iniciar relay | Erro `unsupported` ocorre antes de query, goroutine, Docker ou broadcast | **Somente unitário hoje**; relay v2 continua fora do escopo operacional |
| A14 | Flake mock | `go test -race -count=20 -run '^TestMockChainAndPathProcessors$' ./relayer/chains/mock` e uma suíte raiz paralela | Zero falha sem `-p 1`; condição temporal é sincronizada, não afrouxada | **Aberto**: 3 repetições focais passaram; lote anterior falhou sob suíte paralela |
| A15 | Complexidade incremental | Medir todo Go manuscrito e funções com corpo alterado | Funções novas/alteradas: cyclo <=9 e cognitivo <=9; totais não excedem 83/137/141 | **Ratchet manual aprovado**, enforcement CI ausente |
| A16 | Higiene | `git diff --check`, lint pinado, `go mod verify` nos dois módulos | Tudo passa e nenhuma geração incidental fica no diff | **Revalidar ao final** |

`A7` a `A11` exigem Docker. Se o daemon, registry ou runner não estiver
disponível, o resultado correto é **não verificado**, nunca aprovado.

O gate de grafo não deve exigir ausência literal de `cosmossdk.io/store` ou
`cosmossdk.io/log` em `go list -m all`: o MVS pode enumerar módulos transitivos
dormentes. No ensaio v11, `go mod why -m` informou que o módulo principal não
precisa deles e `go list -deps ./...` carregou somente Store/Log v2. O contrato
correto é ausência de pacotes v1 no grafo de compilação, somada ao compile
readonly.

## Versões de imagem e cobertura real

O inventário atual mistura imports Go v11 com chains antigas. Imports v11
provam compatibilidade de compilação do cliente; não provam interoperabilidade
com uma aplicação v11.

| Uso atual | Imagem/versão | Verificação live em 2026-07-15 | Conclusão |
|---|---|---|---|
| memo e localhost transfer | `ghcr.io/cosmos/ibc-go-simd:v8.0.0` | Existe; digest `sha256:e9fa64ba53912d92d118148711207f916ee74de2bcc9924eff2936151f33bd95` | Apenas compatibilidade antiga |
| localhost ICA | `ghcr.io/cosmos/ibc-go-simd:v8.0.0-beta.1` | Manifest retorna HTTP 404 | Teste quebrará no pull mesmo após compilar |
| candidato publicado mais novo | `ghcr.io/cosmos/ibc-go-simd:v11.1.0` | Existe; digest `sha256:c863331a3836474da4337beff12829b6f120bb1883f9c420e2becceca0fe1891` | Útil como smoke v11.1, mas não representa o grafo alvo v11.2/SDK 0.54 |
| alvo do projeto | `ghcr.io/cosmos/ibc-go-simd:v11.2.0` | Manifest retorna HTTP 404 | Construir do source tag oficial |

O tag oficial
[`ibc-go/v11.2.0`](https://github.com/cosmos/ibc-go/tree/v11.2.0)
aponta para `cfc072e53eee42b2ab804cd4344ba610016f793c`. Seu
[`Dockerfile`](https://github.com/cosmos/ibc-go/blob/v11.2.0/Dockerfile)
constrói `simd`; o
[`go.mod`](https://github.com/cosmos/ibc-go/blob/v11.2.0/go.mod)
declara SDK 0.54.0, CometBFT 0.39.0 e Store v2. Assim, falta publicação de
imagem, não fonte reproduzível.

O lote deve substituir a tag inexistente imediatamente. Imagens por tag devem
ter digest registrado no log de CI; para v11.2.0, a entrada imutável é o commit
do tag e o digest da imagem construída.

## Cenários v11 mínimos

O conjunto mínimo para fechar M1.1b-d é:

1. `classic-v11-events`: duas chains construídas do source v11.2.0, criação de
   client/connection/channel e packet Send -> Recv -> Ack com saldos.
2. `classic-v11-legacy`: o mesmo contrato com processor legacy.
3. `timeout-v11`: packet deliberadamente expirado, confirmação de timeout e
   refund.
4. `localhost-v11`: duas transferências stateless na mesma chain v11.2.0.
5. `misbehaviour-v11`: evidência de `MsgUpdateClient` e estado final do client.
6. `v2-unsupported-boundary`: path v2 falha antes de qualquer side effect. Esse
   teste impede que a migração do grafo seja anunciada como relay v2 completo.

ICS-29 foi removido do upstream v11 e a compatibilidade local é voltada a
chains antigas. Portanto o teste fee middleware atual deve ficar em uma lane
`legacy-compat`, separado da lane `v11`, para não produzir uma conclusão falsa.

## Flake temporal do mock

O teste focal passou em três repetições sob race:

```text
go test -mod=readonly -race -count=3 -timeout=90s \
  -run '^TestMockChainAndPathProcessors$' ./relayer/chains/mock
PASS: 3
```

Isso não invalida as falhas paralelas registradas no M1.1b. O estado é
**inconclusivo/aberto** até o stress de 20 repetições e a suíte paralela
passarem. `-p 1`, aumento cego do limite da fila ou remoção da asserção apenas
mascaram o problema e não são critérios de aceite.

## Complexidade incremental

O gate global foi reexecutado com as ferramentas pinadas e permaneceu idêntico
ao M1.1b:

```text
cyclomatic violations: 83  max 48
cognitive violations:  137 max 99
union:                  141
```

Isso aprova o ratchet do snapshot, mas `make complexity` continua vermelho e
nenhum workflow o executa. Para este sublote, além de não aumentar esses três
totais, toda função manuscrita nova ou cujo corpo foi alterado deve ter score
máximo 9 nas duas métricas. Arquivo gerado só pode ser excluído pelo marcador
determinístico `// Code generated ... DO NOT EDIT.`.

## Falsos verdes que o líder não deve aceitar

1. `go test ./...` na raiz com `go.work` contendo apenas `.`.
2. `go test -list` ou matriz vazia tratados como cobertura de integração.
3. Imports `/v11` em testes executados somente contra imagens v8.
4. Compilação contra interchaintest v11 tratada como E2E v11.2.
5. `simd:v11.1.0` tratado como prova do grafo alvo SDK 0.54/v11.2.
6. Suíte verde somente com `-p 1` usada para encerrar o flake.
7. Contagem global estável sem verificar funções novas/alteradas.
8. `make complexity` existente, mas ausente da CI, tratado como enforcement.
9. Pseudo-version flutuante (`main` ou hash curto sem versão completa).
10. Dependência externa ou Docker indisponível registrado como `PASS`.

## Mudanças necessárias no harness

### Skill `relayer-qa`

Alterar a assertion de módulos. Não basta testar “módulos declarados pelo
`go.work`”, porque remover uma entrada produz falso verde. A skill deve:

1. descobrir todo `go.mod` versionado do repositório;
2. exigir que cada módulo esteja no inventário explícito do harness;
3. executar cada módulo isoladamente com `GOWORK=off -mod=readonly`;
4. executar também o workspace restaurado para detectar seleção cruzada;
5. classificar Docker/registry indisponível como `não verificado`.

### Agente `integration-qa`

Adicionar ao checklist obrigatório:

- inventário de imagens `repository:tag`, existência live e digest;
- comparação entre versão compilada do relayer e versão executada da chain;
- proibição explícita de inferir runtime v11/v2 por imports;
- separação de lanes `legacy-compat`, `v11` e `v2-negative`;
- validação de que a descoberta da matriz falha quando compile/list falha.

### Orquestrador e dependency agent

Registrar o pseudo-version do interchaintest como decisão temporária com SHA,
data e condição de saída: trocar por uma tag v11 estável depois que ela existir,
somente após repetir toda a matriz. Atualizações automáticas de pseudo-version
não devem ser aceitas sem rebaseline do grafo e E2E.

### CI de complexidade

Enquanto a dívida global existir, introduzir ratchet automatizado:

- totais máximos `83/137/141`;
- zero função nova ou com corpo alterado acima de 9;
- falha se uma exclusão gerada não tiver marcador auditável.

Quando a dívida chegar a zero, substituir o ratchet pelo `make complexity`
global estrito. A CI atual não possui nenhum desses gates.

## Condição de handoff

O sublote pode ser integrado somente quando `A1`–`A6`, `A14`–`A16` estiverem
aprovados e ao menos `A7`, `A8` e `A10` tiverem evidência Docker real. Se a
entrega for intencionalmente dividida, compile/workspace pode fechar uma etapa
intermediária, mas ela deve ser rotulada “framework migrated; runtime v11.2
not verified” e não encerra M1.1b-d.
