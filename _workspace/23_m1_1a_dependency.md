# M1.1a — compatibilidade de dependências do wire IBC v2

Data da verificação: 2026-07-15  
SHA analisado: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Alvo oficial: `github.com/cosmos/ibc-go/v11@v11.2.0`

## Decisão

**Não importar** `github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types` no módulo raiz durante M1.1a.

A menor fatia segura é manter o relayer em SDK `v0.50.11`, CometBFT `v0.38.12`, ibc-go/v8 `v8.2.0` e `gogoproto v1.7.0`, e adicionar somente o contrato protobuf de `Packet`, `Payload` e `Acknowledgement` gerado a partir do `packet.proto` da tag v11.2.0 em um pacote wire local. O pacote local deve ter wrappers manuais pequenos para hex, limites, validação e conversão ao modelo neutro. Nenhuma dependência `/v11` entra no `go.mod`.

Isto implementa **compatibilidade de wire e ingestão**, não declara compatibilidade de compilação com ibc-go/v11 nem migração de SDK.

## Estado fixado

O `go.mod` atual seleciona:

| módulo | atual | exigido por ibc-go/v11.2.0 |
|---|---:|---:|
| `github.com/cosmos/cosmos-sdk` | `v0.50.11` | `v0.54.0` |
| `github.com/cometbft/cometbft` | `v0.38.12` | `v0.39.0` |
| `github.com/cosmos/gogoproto` | `v1.7.0` | `v1.7.2` |
| `cosmossdk.io/api` | `v0.7.6` | `v1.0.0` |
| `cosmossdk.io/core` | `v0.11.0` indireto | `v1.1.0` |
| `cosmossdk.io/log/v2` | ausente | `v2.1.0` |
| `github.com/cosmos/cosmos-sdk/store/v2` | ausente | `v2.0.0` |
| IBC Classic | `github.com/cosmos/ibc-go/v8 v8.2.0` | continua necessário nesta etapa |

Fontes primárias fixadas:

- release/tag: <https://github.com/cosmos/ibc-go/releases/tag/v11.2.0>
- módulo: <https://github.com/cosmos/ibc-go/blob/v11.2.0/go.mod>
- schema: <https://github.com/cosmos/ibc-go/blob/v11.2.0/proto/ibc/core/channel/v2/packet.proto>
- validação do packet: <https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/v2/types/packet.go>

Hashes do checkout oficial local, úteis para detectar drift do fixture:

```text
packet.proto   3f74f5b89e53fb05777f13249fb9750c3c14deabf6126d1355ddc272b228d860
packet.pb.go   e395a5dce06187c78fc2908880737fbf158defae7731782ab5809fc446500cc5
```

## Prova 1 — importar o pacote seleciona a pilha nova

Um módulo temporário com v8 e v11, fora do repositório, produziu:

```text
cosmossdk.io/api v1.0.0
cosmossdk.io/core v1.1.0
cosmossdk.io/log/v2 v2.1.0
github.com/cometbft/cometbft v0.39.0
github.com/cosmos/cosmos-sdk v0.54.0
github.com/cosmos/gogoproto v1.7.2
github.com/cosmos/ibc-go/v11 v11.2.0
github.com/cosmos/ibc-go/v8 v8.2.0
```

Comando reproduzível:

```sh
d=$(mktemp -d /tmp/relayer-m11a-deps.XXXXXX)
cd "$d"
GOWORK=off go mod init example.com/compat
GOWORK=off go mod edit -require=github.com/cosmos/ibc-go/v8@v8.2.0
GOWORK=off go mod edit -require=github.com/cosmos/ibc-go/v11@v11.2.0
GOWORK=off go list -mod=mod -m all
GOWORK=off go mod graph
```

Não existe granularidade por arquivo no sistema de módulos ou no compilador Go: referenciar `types.Packet` importa o pacote inteiro `04-channel/v2/types`. Esse diretório também compila `codec.go`, `commitment.go`, `expected_keepers.go`, `keys.go`, `msgs.go`, `query.pb.go` e `tx.pb.go`, que importam SDK e `cosmossdk.io/errors`. Portanto, importar “só Packet” não isola a dependência.

## Prova 2 — o pacote v11 compila, mas o relayer deixa de compilar

Em uma cópia temporária criada com `git archive HEAD`, foi adicionado apenas o requisito v11. Depois de testar o pacote v2, o módulo selecionou SDK `v0.54.0`, Comet `v0.39.0`, gogoproto `v1.7.2`, `log/v2 v2.1.0` e `store/v2 v2.0.0`.

O próprio pacote oficial passou:

```text
ok github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types
```

O build do relayer falhou com incompatibilidades reais:

```text
github.com/cosmos/cosmos-sdk@latest ... does not contain package github.com/cosmos/cosmos-sdk/x/crisis
github.com/cometbft/cometbft@latest ... does not contain package github.com/cometbft/cometbft/crypto/sr25519
cosmossdk.io/x/upgrade@v0.1.1 ... cosmossdk.io/store/types.CommitMultiStore ...
... github.com/cosmos/cosmos-sdk/store/v2/types.CommitMultiStore ...
```

Comando reproduzível:

```sh
d=$(mktemp -d /tmp/relayer-m11a-build.XXXXXX)
git archive HEAD | tar -x -C "$d"
cd "$d"
GOWORK=off go mod edit -require=github.com/cosmos/ibc-go/v11@v11.2.0
GOWORK=off go test -mod=mod github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types
GOWORK=off go test -mod=mod ./relayer/chains/...
```

Conclusão: coexistência dos caminhos de módulo `/v8` e `/v11` é sintaticamente permitida, mas não preserva duas versões das dependências compartilhadas. MVS seleciona uma única versão de SDK/Comet para o build e a pilha nova quebra o código atual.

## Prova 3 — o wire mínimo oficial funciona com a dependência atual

O arquivo gerado oficial `packet.pb.go` da tag v11.2.0 importa somente stdlib e `github.com/cosmos/gogoproto`. Copiado sozinho para outro módulo temporário e fixado em `gogoproto v1.7.0`, ele compilou:

```text
? example.com/wire [no test files]
github.com/cosmos/gogoproto v1.7.0
```

O módulo de prova não selecionou SDK, CometBFT nem ibc-go/v11.

```sh
d=$(mktemp -d /tmp/relayer-m11a-wire.XXXXXX)
cd "$d"
GOWORK=off go mod init example.com/wire
GOWORK=off go mod edit -require=github.com/cosmos/gogoproto@v1.7.0
cp "$(go env GOMODCACHE)/github.com/cosmos/ibc-go/v11@v11.2.0/modules/core/04-channel/v2/types/packet.pb.go" .
GOWORK=off go test -mod=mod .
GOWORK=off go list -m github.com/cosmos/gogoproto
```

Esta prova é de compilação do contrato gerado exato. A equivalência de bytes deve continuar sendo um gate obrigatório com goldens produzidos pelo tipo oficial v11.2.0.

## Comparação das opções

| opção | resultado | risco/complexidade | decisão M1.1a |
|---|---|---|---|
| Importar `/v11/.../v2/types` e fazer upgrade completo | exige SDK 0.54, Comet 0.39, store/v2 e migrações do relayer; a prova já quebra `x/crisis`, `sr25519` e stores | muito alto; mistura ingestão com migração estrutural | rejeitada; marco separado |
| Contrato wire local gerado do `packet.proto` v11.2.0 | compila com gogoproto 1.7.0 e não altera o grafo atual | baixo e removível; requer proveniência, limites e goldens | **selecionada** |
| Tipos manuais apenas com tags protobuf/reflection | pode manter pouco código, mas precisa provar nullable/repeated, unknown fields e determinismo | médio; mais fácil divergir silenciosamente do schema | usar somente se goldens provarem todos os campos; gerado é preferível |
| Submódulo isolado dependente de v11 | só isola se for compilado com `GOWORK=off` e consumido por IPC/arquivo; uma importação pelo root reintroduz MVS | alto custo operacional, outro binário/protocolo/release | rejeitada para ingestão in-process |

## Fatia vertical recomendada

1. Criar um pacote wire interno contendo somente `Packet`, `Payload` e `Acknowledgement` gerados do `packet.proto` v11.2.0. Registrar tag, commit/hash e licença/proveniência no arquivo ou em README adjacente.
2. Manter o pacote sem imports de SDK, CometBFT ou `github.com/cosmos/ibc-go/v11`.
3. Criar wrappers manuais para:
   - validar comprimento do hex antes de alocar/decodificar;
   - limitar bytes protobuf antes do `Unmarshal`;
   - decodificar packet e acknowledgement com erros tipados;
   - executar equivalentes locais das invariantes relevantes de `ValidateBasic`;
   - converter para o modelo neutro e preservar também os bytes brutos observados.
4. Aplicar o limite oficial de soma dos payloads (`262144`, 256 KiB), exatamente como v11.2.0. Definir ainda um limite operacional explícito para o blob protobuf completo e para acknowledgement, pois `version`, `encoding` e app-ack não têm limite total equivalente nessa validação oficial.
5. Não ligar a observação v2 a cache, construção de mensagem ou broadcast nesta fatia.
6. Remover/substituir o wire local quando o marco de migração completa tornar seguro importar os tipos oficiais diretamente.

## Gates de compatibilidade e aceite

### Grafo de módulos

- `go.mod` e `go.sum` não ganham `github.com/cosmos/ibc-go/v11`.
- SDK permanece `v0.50.11`, CometBFT `v0.38.12`, ibc-go/v8 `v8.2.0`, gogoproto `v1.7.0`.
- `go list -m all` não passa a selecionar `github.com/cosmos/cosmos-sdk/store/v2` ou `cosmossdk.io/log/v2` por causa de M1.1a.
- Um teste/checagem CI falha se o pacote wire importar SDK, Comet ou ibc-go/v11.

### Conformidade de wire

- Goldens de `Packet` e `Acknowledgement` são gerados em ferramenta/fixture isolado usando exatamente v11.2.0.
- O decoder local lê todos os campos, payload binário e app-ack sem perda; o encoder local, se exposto, produz bytes deterministicamente equivalentes para os fixtures.
- Campos protobuf desconhecidos não causam panic nem mudam a classificação; os bytes brutos do evento continuam preservados para reprocessamento futuro.
- Testes cobrem hex inválido/ímpar, protobuf truncado, wire type errado, length overflow, blob acima do limite, zero/mais de um payload, payload total `262144` e `262145`, sequência/timeout zero e acknowledgement vazio/múltiplo.
- Fuzz do decoder não causa panic ou alocação não limitada.

### Regressão e escopo

- `go test ./...` e os testes focados de parsing/protocolo passam com o grafo original.
- Eventos Classic continuam produzindo os mesmos resultados.
- Eventos v2 observados não entram em cache nem broadcast neste lote.
- Todo código manual novo/tocado tem complexidade ciclomática e cognitiva máxima `9`; arquivos realmente gerados ficam identificados como gerados e fora do gate manual.
- O diff de `go.mod`/`go.sum` para M1.1a é vazio, salvo uma mudança explicitamente justificada e novamente provada.

## Marco separado necessário para upgrade completo

Importar ibc-go/v11 diretamente deve ser tratado como uma migração própria, com pelo menos: SDK 0.54, Comet 0.39, store/v2, substituição de `x/crisis`, estratégia para `sr25519`, atualização dos módulos `cosmossdk.io/x/*`, codecs/interfaces, Penumbra, interchaintest, CLI/build/release e matriz de chains. Só depois desses gates o wire local pode ser removido em favor dos tipos oficiais.
