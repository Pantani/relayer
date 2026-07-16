# M1.1b — contrato incremental de QA

Data do snapshot: 2026-07-15 19:10–19:15 BRT  
Branch: `Pantani/cx/m0-baseline`  
Base Git preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Escopo: migração coerente do grafo para SDK 0.54 / CometBFT 0.39 / ibc-go v11, preservando IBC Classic.  
Parecer deste documento: **contrato definido; implementação M1.1b ainda não aprovada**.

Este relatório foi produzido em modo read-only para código de produção. O único
arquivo criado por este agente é este artefato de QA. O worktree já continha as
mudanças não commitadas de M0–M1.1a e começou a receber edições M1.1b de outros
agentes durante a medição. Por isso, os resultados abaixo distinguem o snapshot
pré-edição do estado intermediário concorrente.

## 1. O que “passar M1.1b” significa

M1.1b não implementa ainda relay operacional IBC v2. Ele deve produzir uma
base compilável e comportamentalmente compatível com IBC Classic usando o
grafo novo. A aceitação tem três níveis que não podem ser confundidos:

| nível | o que prova | o que não prova |
|---|---|---|
| **compilação** | APIs, imports, codecs e tipos fecham em raiz, `interchaintest`, plataformas de release e imagem | que packet/proof/tx preservam semântica ou que duas chains interoperam |
| **comportamento** | fixtures/goldens e testes executam parser, config, query decode, builders, signing e tratamento de respostas | compatibilidade com uma chain real, mempool, keeper ou proof store |
| **interoperabilidade** | duas chains pinadas executam criação de client/connection/channel, packet, ack/timeout e confirmação on-chain | suporte IBC v2, salvo se o cenário usar explicitamente channel v2 |

Um build verde é necessário, mas não aprova sozinho a migração. Da mesma forma,
os interchaintests Classic existentes não provam IBC v2.

## 2. Snapshot de módulos e workspace antes das edições M1.1b

`go.work` declara exatamente dois módulos e Go 1.25.9:

```text
use (
    .
    ./interchaintest
)
```

| artefato | SHA-256 no snapshot |
|---|---|
| `go.mod` | `eb0e65d35f1530460c61f0ce0858765c6a4e784c08661e9dc40640a6336aa3fa` |
| `go.sum` | `a5832669e4028b88ccd00f8a743725be0a5e397c49769732b95aacd3653bc767` |
| `go.work` | `b36d9c1218bcfd45e2f5803de682d016b03708380d0a83f0e4e194c5dd6376d9` |
| `interchaintest/go.mod` | `811563792d8098796bcdd3e5e63ca524a99bc12c61626762254ce1487b548d43` |
| `interchaintest/go.sum` | `ed77a80414297f8c37c9710e2df76dda680c317e269dbce05bd817cf503fb466` |

Os dois módulos selecionavam a mesma pilha:

| módulo | versão pré-M1.1b |
|---|---:|
| `github.com/cosmos/cosmos-sdk` | `v0.50.11` |
| `github.com/cometbft/cometbft` | `v0.38.12` |
| `github.com/cosmos/ibc-go/v8` | `v8.2.0` |
| `github.com/cosmos/gogoproto` | `v1.7.0` |
| `cosmossdk.io/api` | `v0.7.6` |
| `cosmossdk.io/core` | `v0.11.0` |
| `cosmossdk.io/store` | `v1.1.1` |

`github.com/cosmos/ibc-go/v11`, `cosmossdk.io/log/v2` e
`github.com/cosmos/cosmos-sdk/store/v2` ainda não eram dependências conhecidas.

## 3. Resultado executado no snapshot pré-edição

| assertion | resultado | evidência | classe |
|---|---|---|---|
| root unitário sem race | **passou** | `go test -mod=readonly -count=1 ./...`: 357 testes / 51 pacotes | comportamento |
| root build | **passou** | `go build -mod=readonly ./...` | compilação |
| `interchaintest` compile-only no workspace | **passou** | `go test -mod=readonly -run '^$' ./...` | compilação |
| `interchaintest` build no workspace | **passou** | `go build -mod=readonly ./...` | compilação |
| lint pinado | **passou** | `make lint`: 0 issues; modules verified | estático |
| módulos raiz e `interchaintest` | **passou** | `GOWORK=off go mod verify` em ambos | integridade |
| diff whitespace | **passou** | `git diff --check` | higiene |
| root `go vet ./...` | **falhou, preexistente** | dois avisos no protobuf gerado Injective | estático |
| race em superfícies Classic | **falhou, preexistente e reproduzido** | 343 passaram/4 falharam; rerun `./cmd`: 63 passaram/3 falharam | comportamento/concorrência |
| módulo `interchaintest` isolado | **falhou, preexistente e reproduzido** | `GOWORK=off go list -mod=readonly ./...` exige tidy | reprodutibilidade |
| complexidade global | **falhou, preexistente** | 86 ciclomaticas, 138 cognitivas, união 144; máximos 48/99 | qualidade |
| interchain Docker | **não verificado neste snapshot** | daemon Docker 29.6.1 disponível; cenários não foram iniciados por este agente | interoperabilidade |
| release cross-platform/Docker | **não verificado neste snapshot** | apenas configuração inspecionada | distribuição |

Às 19:15, depois que outro agente começou a trocar imports, `make test` passou a
falhar em setup por imports v11/SDK novos ainda ausentes no `go.mod`. Este é um
estado intermediário esperado de trabalho compartilhado e **não** deve ser
registrado como falha preexistente nem como resultado final do lote.

## 4. Falhas preexistentes que precisam de tratamento explícito

### F1 — race da configuração global do SDK na CLI

`go test -mod=readonly -race -count=1 ./cmd` falhou novamente, com 63 testes
passando e três falhando. Os nomes que recebem a falha variam conforme o
escalonamento (`TestKeysExport`, `TestKeysDefaultCoinType`,
`TestKeysRestoreAll_Delete` e, em outra execução, subteste de debug server), mas
a origem é estável: testes `t.Parallel` criam providers que escrevem no singleton
`sdk.Config` por `SetSDKConfigContext`; o race detector aponta
`Config.SetBech32PrefixForAccount` e setters correlatos.

Classificação: **preexistente antes de M1.1b, severidade alta para CI**, porque
`make test` e o workflow de build usam `-race`. A migração não pode declarar
regressão zero apoiando-se apenas na suíte sem race. Opções aceitáveis são corrigir
a sincronização/isolamento ou registrar um gate serial temporário específico; não
é aceitável ignorar o detector.

### F2 — `interchaintest` só está tidy dentro do workspace

Dentro do `go.work`, compile e build passaram. Isolado, tanto
`GOWORK=off go list -mod=readonly ./...` quanto o rerun falharam. O diagnóstico
read-only `GOWORK=off go mod tidy -diff` pede exatamente:

- `github.com/golang/glog v1.2.2 -> v1.2.4`;
- `golang.org/x/net v0.29.0 -> v0.33.0`;
- atualização correspondente de `interchaintest/go.sum`.

Classificação: **preexistente, severidade média**. M1.1b deve sair com ambos os
módulos tidy isoladamente; o `go.work` não pode mascarar divergência de release.

### F3 — `go vet` raiz vermelho em protobuf gerado

`go vet -mod=readonly ./...` e o rerun focado reprovam
`relayer/codecs/injective/tx.pb.go:242-243`: campos não exportados `chainId` e
`chainIdMul` possuem JSON tags. O arquivo contém marcador determinístico
`Code generated ... DO NOT EDIT`.

Classificação: **preexistente, severidade baixa para M1.1b**, mas precisa de uma
das três decisões rastreáveis: regenerar com toolchain pinada, corrigir upstream,
ou manter uma exclusão explícita e estreita no gate de vet. Não atribuir a falha
automaticamente ao SDK v0.54.

### F4 — dívida de complexidade global

O comando pinado `make complexity` reproduziu:

```text
cyclomatic=86 cognitive=138 union=144 max=48/99
```

O contrato incremental de M1.1b exige que cada função manuscrita criada ou tocada
tenha score máximo 9 nas duas métricas e que o inventário global não aumente.
Isso permite aceitar a fatia de migração, mas **não** permite declarar atendido o
objetivo global do programa; `make complexity` continuará vermelho até a dívida
chegar a zero.

## 5. Gates ordenados da migração

### G0 — congelar o baseline

1. Registrar base, hashes dos cinco arquivos de módulos e lista exata de arquivos
   tocados pelo lote.
2. Separar F1–F4 de qualquer falha nova.
3. Não modificar wire v2 e imports Classic na mesma etapa sem um checkpoint
   compilável entre elas.

### G1 — grafo coerente

Critérios obrigatórios nos dois `go.mod`:

- alvo fixado pelo roadmap: SDK `v0.54.3`, CometBFT `v0.39.3`, ibc-go
  `v11.2.0`, Go `1.25.9`; versões transitivas documentadas pelo relatório de
  dependência;
- nenhuma referência a `github.com/cosmos/ibc-go/v8` em código manuscrito,
  testes, `go.mod` ou `go.sum`;
- nenhum import do removido `github.com/cosmos/cosmos-sdk/x/crisis` ou do removido
  `github.com/cometbft/cometbft/crypto/sr25519`;
- módulos `cosmossdk.io/x/*`, log e stores formam um grafo único resolvido. A
  coexistência transitiva de store v1/v2 só é aceitável se nenhuma interface do
  relayer mistura tipos incompatíveis e a razão estiver registrada;
- `GOWORK=off go mod tidy -diff`, `GOWORK=off go mod verify` e build passam
  separadamente na raiz e em `interchaintest`;
- `go list -m all` e `go mod graph` ficam anexados ao handoff, incluindo todas
  as substituições e exclusões.

A capability sr25519 não pode desaparecer silenciosamente. O gate exige uma
implementação independente compatível, ou erro de validação determinístico com
nota de migração/depreciação aprovada pelo mantenedor.

### G2 — compilação por fronteira

Todos os comandos devem passar com `-mod=readonly`:

| fronteira | gate mínimo |
|---|---|
| root | `go test -run '^$' ./...` e `go build ./...` |
| Cosmos | compile de `cclient`, `relayer/chains/cosmos`, `provider`, `processor`, codecs Ethermint/Injective |
| Penumbra | compile de todo `relayer/chains/penumbra/...`; tipos gerados não podem esconder erro no provider manuscrito |
| CLI | compile de `cmd` e binário `main.go` |
| interchaintest | compile/build dentro do workspace **e** com `GOWORK=off` |
| release | snapshot Goreleaser nas quatro combinações declaradas: darwin/linux × amd64/arm64, com CGO/cross compiler real |
| Docker | build multi-arch linux amd64/arm64 do Dockerfile e smoke do binário na imagem final |

O `Makefile` possui uma inconsistência a vigiar: o ramo não-Windows de `make
build` não usa explicitamente `-mod=readonly`, embora os outros builds usem. O
gate M1.1b deve invocar também `go build -mod=readonly ./...` diretamente.

### G3 — comportamento Classic preservado

Além de `go test -count=1 ./...`, executar com `-race` após tratar F1. Nenhum
teste Classic pode ser removido ou enfraquecido para acomodar a API nova.

| superfície | assertions obrigatórias |
|---|---|
| parsing/eventos | mesmos cinco tipos Classic, atributos, ordem, timeout height/timestamp, ack e nenhum fallthrough v2 |
| modelo/provider | round-trip `PacketInfo`, `PacketProof`, tipos URL e bytes protobuf Classic equivalentes ao golden v8 pré-migração |
| queries/proofs | commitment, acknowledgement, receipt, next-seq-recv, client/connection/channel e alturas retornam os mesmos modelos e propagam erros |
| builders | create/update client, connection/channel handshakes, recv/ack/timeout/transfer preservam signer, proof height, packet e type URL |
| tx/signing | fee, gas, feegrant, dynamic fee, keyring, Ethermint/Injective e resposta parcial preservam resultado/erro |
| sr25519 | restore, address, sign/verify e serialização preservados, ou rejeição explícita conforme decisão aprovada |
| config/path | YAML antigo continua round-trip sem adicionar campos; default permanece Classic; paths v2 continuam bloqueados antes do runtime |
| CLI | árvore de comandos, flags e help Classic; `config init/show`, `paths`, `chains`, `keys`, `query`, `tx`, `start` e `version` têm smoke determinístico |
| wire/sidecar v2 | goldens M1.1a continuam idênticos; sidecar segue sem consumidor de cache/broadcast |

O swap do wire local pelo tipo oficial v11 só pode ocorrer depois de G1–G3 verdes
com o wire local. Depois do swap, repetir todos os goldens/fuzz de
`relayer/protocol/v2` e `relayer/chains`; raw bytes, limites e erros tipados não
podem mudar.

### G4 — interoperabilidade Classic

Os cenários Docker atuais cobrem event processor, legacy processor, múltiplos
paths, misbehaviour, fee middleware, feegrant, ICA, localhost, backup RPC,
filters e cenários adicionais. Eles usam imagens heterogêneas antigas (por
exemplo Gaia v14.1.0, Osmosis v22.0.0, ibc-go-simd v8 e ibc-go-icad v0.5.0).

M1.1b exige duas faixas:

1. **compatibilidade para trás:** executar a matriz Classic existente sem alterar
   expectativas;
2. **compatibilidade da pilha nova:** adicionar duas chains explicitamente
   pinadas que executem SDK 0.54/Comet 0.39/ibc-go v11.2 e validar, via ambos os
   processors aplicáveis, client update, connection/channel Classic, transfer,
   ack, timeout, duplicate/NOOP, restart e erro de proof.

Sem a segunda faixa, o máximo que pode ser declarado é “compila contra v11 e
mantém interop com fixtures antigas”, não “interoperabilidade v11 validada”.

### G5 — release e operação

- `make lint`, módulo verify, vet com política explícita para F3 e race suite;
- Goreleaser snapshot sem publish e smoke `rly version`, `rly config init/show`;
- build Docker amd64/arm64 e execução como UID não-root da imagem scratch;
- nenhum acesso a RPC público em teste unitário;
- release workflow não publica durante QA; apenas snapshot/artifact local;
- `git diff --check` e ausência de alterações em arquivos gerados fora da
  regeneração pinada.

## 6. Matriz de regressão mínima

| superfície | compile | comportamento local | interoperabilidade | status do fixture atual |
|---|---|---|---|---|
| Cosmos provider Classic | obrigatório | obrigatório; cobertura existente é parcial | obrigatório em chain antiga e v11 | há interchaintest, mas unitários de query/proof/builder são escassos |
| Penumbra provider Classic | obrigatório | obrigatório por goldens/mocks | desejável; se indisponível, marcar não verificado | **nenhum `_test.go` no pacote e nenhum cenário Penumbra** |
| CLI/config/path | obrigatório | obrigatório, incluindo race | smoke dentro dos interchaintests | há testes de config/keys/start, mas não golden completo de help/command tree |
| processor event-based | obrigatório | caches/transições unitárias | matriz Classic antiga + v11 | cenários existem para chains antigas |
| processor legacy | obrigatório enquanto suportado | regressão mínima | matriz Classic antiga + v11 | cenário existe para chains antigas |
| Ethermint/Injective | obrigatório | signing/type URL/goldens | chain smoke quando disponível | Ethermint tem testes; Injective não tem testes e vet gerado falha |
| interchaintest module | workspace e isolado | descoberta/compile dos testes | Docker real | isolado não está tidy no baseline |
| release/Docker | quatro cross-builds | smoke do binário | execução da imagem | configuração existe; não executada neste snapshot |
| ingestão v2 M1.1a | obrigatório | goldens, negativos, race e fuzz | não aplicável a M1.1b | fixtures locais fortes; ainda sem block/tx real |

## 7. Lacunas de fixtures que bloqueiam conclusões fortes

1. **Penumbra:** não existe teste no pacote, mock de gRPC/proof, golden de
   mensagem Classic nem cenário interchain. Compilação é hoje a única evidência.
2. **Cosmos query/proof/builders:** os unitários atuais cobrem fee market, caches,
   gas, extension options e keys, mas não congelam bytes/type URLs de todos os
   builders nem responses/proofs relevantes.
3. **Chain v11 Classic:** não há fixture explicitamente pinada em SDK
   0.54/Comet 0.39/ibc-go v11.2. A ocorrência `Version: "v11.0.0"` em
   `backup_rpc_test.go` é versão de uma imagem de chain e não prova ibc-go/v11.
4. **Release:** não existe teste automatizado de Goreleaser snapshot, cross CGO
   ou execução das imagens amd64/arm64.
5. **CLI:** não há snapshot/golden completo da árvore de help e das type URLs
   emitidas pelos comandos transacionais.
6. **Injective:** protobuf gerado falha vet e não há suite de comportamento do
   codec após mudança de SDK/gogoproto.
7. **IBC v2 real:** M1.1a possui goldens de protobuf/eventos, mas não block/tx,
   query/proof/response ou duas chains v11; isso não bloqueia M1.1b, porém impede
   qualquer declaração de relay v2.

## 8. Checklist reproduzível de aceite final

Executar a partir da raiz, guardando os logs e repetindo falhas uma vez:

```sh
rtk env GOWORK=off go mod tidy -diff
rtk env GOWORK=off go mod verify
rtk go test -mod=readonly -run '^$' ./...
rtk go build -mod=readonly ./...
rtk go test -mod=readonly -count=1 ./...
rtk go test -mod=readonly -race -count=1 ./...
rtk go vet -mod=readonly ./...
rtk make lint
rtk make complexity
rtk git diff --check
```

Executar também dentro de `interchaintest`, primeiro com o workspace e depois
com `GOWORK=off`:

```sh
rtk go test -mod=readonly -run '^$' ./...
rtk go build -mod=readonly ./...
rtk env GOWORK=off go mod tidy -diff
rtk env GOWORK=off go mod verify
rtk env GOWORK=off go test -mod=readonly -run '^$' ./...
rtk env GOWORK=off go build -mod=readonly ./...
```

Depois, executar a matriz Docker Classic antiga, a nova matriz v11 e o snapshot
de release. Resultado dependente de daemon, imagem, registry ou RPC externo deve
ser `não verificado` quando a dependência estiver indisponível, nunca `aprovado`.

## 9. Critério de decisão do líder

M1.1b pode ser aceito incrementalmente quando G1–G5 estiverem verdes, F1–F4
estiverem corrigidas ou explicitamente isoladas sem mascarar regressões, e as
lacunas obrigatórias de Cosmos/v11 tiverem fixtures. Penumbra sem fixture real
deve sair marcada como **compile-compatible, interoperabilidade não verificada**.

Mesmo após essa aceitação, o projeto não pode declarar “IBC v2 suportado”: queries,
proofs, builders, state machine, broadcast e interchain v2 continuam pertencendo
aos lotes seguintes.
