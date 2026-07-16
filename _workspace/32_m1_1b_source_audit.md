# M1.1b — auditoria de fonte para SDK 0.54.3 / CometBFT 0.39.3 / ibc-go v11.2.0

Data: 2026-07-15  
Baseline funcional auditado: `bef2e868f157659b403fe1303ee121fb69fec9e6`, incluindo as mudanças locais M0/M1.1a copiadas antes da prova.  
Escopo deste relatório: leitura do repositório e experimentos destrutivos apenas em `/tmp/relayer-m11b-audit-source`; nenhum código de produção foi editado por esta auditoria.

## Resumo executivo

A migração do módulo raiz é finita e foi levada até uma compilação verde numa cópia temporária. O baseline contém 19 caminhos de importação IBC distintos em 71 arquivos Go (65 no commit e 6 arquivos locais novos); a maioria aceita troca mecânica `/v8` -> `/v11`, mas dez grupos de API precisam adaptação semântica.

O bloqueio crítico remanescente não está no binário `rly`: está no submódulo `interchaintest`. Tanto `github.com/strangelove-ventures/interchaintest/v8@v8.0.1-...` quanto `github.com/cosmos/interchaintest/v10@v10.0.1` importam `cosmossdk.io/x/upgrade` e capability baseados em Store v1/log v1. Eles não compilam quando MVS seleciona SDK 0.54/Store v2/log v2. Portanto o upgrade não deve ser considerado completo enquanto uma destas opções não for escolhida:

1. manter/forkar interchaintest e portar `chain/cosmos` para SDK 0.54/Store v2;
2. substituir os testes in-process por testes Docker que não importem o app/harness Go antigo;
3. aguardar uma release upstream compatível, mantendo um gate separado explicitamente vermelho até lá.

Não basta retirar `interchaintest` do `go.work`: para testar o código local o módulo precisa de `replace github.com/cosmos/relayer/v2 => ..`, o que volta a selecionar SDK 0.54 e reproduz o conflito.

## Prova reproduzível

### Grafo selecionado na prova raiz

```text
github.com/cosmos/cosmos-sdk                 v0.54.3
github.com/cometbft/cometbft                 v0.39.3
github.com/cosmos/ibc-go/v11                 v11.2.0
github.com/cosmos/gogoproto                  v1.7.2
cosmossdk.io/api                             v1.0.0
cosmossdk.io/core                            v1.1.0
cosmossdk.io/log/v2                          v2.1.0
github.com/cosmos/cosmos-sdk/store/v2        v2.0.0
```

Depois das adaptações listadas abaixo, a cópia temporária passou:

```sh
GOWORK=off go test -mod=mod ./...
```

Resultado: todos os 51 pacotes do módulo raiz compilaram/testaram; a única falha anterior era um teste copiado junto do pacote temporário sr25519, removido da prova porque a proposta é portar somente a implementação e escrever testes próprios.

O submódulo reproduziu o bloqueio com:

```sh
cd interchaintest
GOWORK=off go test -mod=mod -run '^$' ./...
```

Falhas determinísticas:

- `cosmossdk.io/x/upgrade/types.StoreLoader` usa `cosmossdk.io/store/types.CommitMultiStore`, incompatível com `github.com/cosmos/cosmos-sdk/store/v2/types.CommitMultiStore`;
- capability usa KVStore v1 e retorna `cosmossdk.io/log.Logger`, enquanto SDK 0.54 fornece KVStore v2 e `cosmossdk.io/log/v2.Logger`;
- `go mod why` aponta `interchaintest/.../chain/cosmos` como importador de ambos.

## Inventário dos caminhos IBC Classic

| caminho v8 | ocorrências | migração |
|---|---:|---|
| `modules/apps/27-interchain-accounts` | 1 | trocar para `/v11`; somente interchaintest |
| `modules/apps/27-interchain-accounts/types` | 1 | trocar para `/v11`; somente interchaintest |
| `modules/apps/29-fee` | 1 | removido em v10; não existe em v11 |
| `modules/apps/29-fee/types` | 3 | removido; criar compatibilidade local mínima ou retirar ICS-29 |
| `modules/apps/transfer` | 3 | trocar para `/v11`; revisar ModuleBasic |
| `modules/apps/transfer/types` | 18 | trocar para `/v11`; migrar DenomTrace/queries |
| `modules/core` | 3 | trocar para `/v11` |
| `modules/core/02-client/types` | 28 | trocar; revisar ClientState e misbehaviour |
| `modules/core/03-connection/types` | 19 | troca mecânica compilou |
| `modules/core/04-channel/types` | 48 | trocar; preservar constantes de evento legadas localmente |
| `modules/core/23-commitment/types` | 8 | troca mecânica compilou nos usos atuais |
| `modules/core/24-host` | 6 | troca mecânica compilou nos usos atuais |
| `modules/core/exported` | 19 | trocar; `ClientState.GetLatestHeight` foi removido |
| `modules/core/types` | 1 | troca mecânica; somente interchaintest |
| `modules/light-clients/06-solomachine` | 1 | trocar; altura agora deriva de `Sequence` |
| `modules/light-clients/07-tendermint` | 11 | trocar; usar campo `LatestHeight` |
| `modules/light-clients/09-localhost` | 2 | cliente tornou-se stateless; tipos/registro foram removidos |
| `testing` | 1 | somente interchaintest; revisar `TrustedValidators` |
| `testing/mock` | 1 | somente interchaintest |

Os dois arquivos protobuf Penumbra gerados também incorporam imports `/v8`: `relayer/chains/penumbra/cnidarium/v1/cnidarium.pb.go` e `relayer/chains/penumbra/core/component/ibc/v1/ibc.pb.go`. A troca compilou, mas deve ser reproduzível pelo gerador/proto de origem; editar somente o `.pb.go` deixa a próxima geração regressar para v8.

## Quebras e substituições por arquivo/símbolo

### 1. SDK x/* que voltou ao monorepo

Arquivos:

- `relayer/chains/cosmos/codec.go:ModuleBasics`, `MakeCodecConfig`
- `relayer/chains/penumbra/codec.go:moduleBasics`
- `relayer/chains/cosmos/feegrant.go`
- `relayer/chains/cosmos/query.go`
- `relayer/chains/penumbra/query.go`
- `interchaintest/feegrant_test.go`
- `interchaintest/localhost_client_test.go`

Substituições:

```text
cosmossdk.io/x/feegrant[/module] -> github.com/cosmos/cosmos-sdk/x/feegrant[/module]
cosmossdk.io/x/upgrade[/types]   -> github.com/cosmos/cosmos-sdk/x/upgrade[/types]
cosmossdk.io/x/tx/signing        -> github.com/cosmos/cosmos-sdk/x/tx/signing
```

Risco: médio. Type URLs permanecem Cosmos, mas os tipos Go são de módulos diferentes e não podem coexistir no mesmo registry. Remover também os requisitos diretos antigos do `go.mod`; manter um import antigo reintroduz Store v1/API v0.

### 2. `x/crisis`

Arquivos/símbolos:

- `relayer/chains/cosmos/codec.go:ModuleBasics`
- `relayer/chains/penumbra/codec.go:moduleBasics`

Substituição: `github.com/cosmos/cosmos-sdk/x/crisis` -> `github.com/cosmos/cosmos-sdk/contrib/x/crisis`. `crisis.AppModuleBasic{}` continua compilando.

Risco: baixo para codec, médio para CLI/genesis. O relayer usa o módulo para registrar tipos; testar decode de uma tx `MsgVerifyInvariant` antiga evita regressão silenciosa.

### 3. capability removido

Arquivos/símbolos:

- `relayer/chains/cosmos/codec.go:ModuleBasics`
- `relayer/chains/penumbra/codec.go:moduleBasics`
- `interchaintest/localhost_client_test.go` app module list

Substituição: remover `github.com/cosmos/ibc-go/modules/capability` e `capability.AppModuleBasic{}`. Seu `RegisterInterfaces` é no-op; portanto a retirada do codec não perde tipos de mensagens.

Risco: baixo no binário; alto no app in-process antigo do interchaintest, que precisa ser refeito conforme a composição v11 sem capability.

### 4. Store v1 -> Store v2 e log v1 -> log v2

Arquivos/símbolos diretos:

- `relayer/chains/cosmos/tx.go:ABCIQueryRequiresProof`
- `relayer/chains/penumbra/tx.go:ABCIQueryRequiresProof`

Substituição: `cosmossdk.io/store/rootmulti` -> `github.com/cosmos/cosmos-sdk/store/v2/rootmulti`. `rootmulti.RequireProof` manteve assinatura e comportamento na prova.

O binário não importa logger Cosmos diretamente; continua usando zap. `cosmossdk.io/log/v2` é consequência do SDK/IBC. O conflito de logger observado pertence a capability/interchaintest antigos e desaparece do módulo raiz quando capability é removido.

Risco: baixo nos dois usos diretos; alto se qualquer módulo antigo `cosmossdk.io/x/*` ou harness app continuar no grafo.

### 5. sr25519 removido do CometBFT

Arquivos/símbolos:

- `cclient/cmbft_client_wrapper.go:convertPubKey`
- `relayer/chains/cosmos/keys.go:SupportedAlgorithms`, `SupportedAlgorithmsLedger`, `KeyRestore`
- `relayer/chains/cosmos/keys/sr25519/algo.go:sr25519Algo`
- `relayer/chains/cosmos/keys/sr25519/privkey.go:PrivKey`
- `relayer/chains/cosmos/keys/sr25519/pubkey.go:PubKey`
- `relayer/chains/cosmos/keys/sr25519/keys.pb.go:PubKey.Key`

Quebras:

- `github.com/cometbft/cometbft/crypto/sr25519` não existe em 0.39;
- `hd.Sr25519Type` não existe em SDK 0.54;
- o `.pb.go` usa `casttype` apontando para o pacote removido.

Substituição proposta:

1. portar para um pacote local pequeno a implementação sr25519 de Comet 0.38, atualizando-a para implementar `cometcrypto.PubKey/PrivKey` 0.39 e mantendo bytes, endereço e domínio de assinatura;
2. usar `hd.PubKeyType("sr25519")`;
3. regenerar `keys.pb.go` para bytes/local type, preservando o protobuf name `cosmos.crypto.sr25519.PubKey`;
4. adaptar a chave da biblioteca `strangelove-ventures/cometbft-client` em `convertPubKey`, sem cast de slice entre tipos não relacionados.

A prova temporária com a implementação Comet 0.38 portada compilou. Risco: alto — qualquer diferença em derivação, signing context, serialização ou address hash pode tornar chaves existentes inutilizáveis. Gates obrigatórios: vetores conhecidos seed->pubkey->address->signature, restore de keyring antigo, encode/decode Any e conversão do consenso.

### 6. ICS-29 fee removido

Arquivos/símbolos:

- `relayer/chains/cosmos/codec.go:ModuleBasics` (`ibcfee.AppModuleBasic`)
- `relayer/chains/cosmos/tx.go:MsgRegisterCounterpartyPayee`
- `relayer/chains/cosmos/log.go:getFeePayer`
- `relayer/chains/penumbra/log.go:getFeePayer`

Símbolos ainda usados: `MsgRegisterPayee`, `MsgRegisterCounterpartyPayee`, `MsgPayPacketFee`, `MsgPayPacketFeeAsync` e `NewMsgRegisterCounterpartyPayee`.

Substituição proposta: retirar `ibcfee.AppModuleBasic` e criar pacote de compatibilidade local mínimo, gerado dos protos v8, que preserve os type URLs ICS-29 e registre somente essas quatro mensagens. Não manter `/v8` apenas para fee: seus tipos puxam core v8, `cosmossdk.io/x/upgrade`, Store v1 e quebram MVS.

Risco: alto. IBC-go v11 não oferece replacement de ICS-29. O pacote local serve apenas para chains Classic antigas; ele não deve ser interpretado como suporte fee em IBC v2. Testar goldens protobuf/type URL e uma tx assinada de cada mensagem.

### 7. eventos Classic cujas constantes foram removidas

Arquivos/símbolos:

- `relayer/chains/packet_event_classifier.go:classicPacketAttributeKeys`
- `relayer/chains/parsing.go:(*PacketInfo).parsePacketAttribute`
- `relayer/chains/parsing.go:ParseIBCMessageFromEvent`
- `relayer/chains/parsing_test.go`

Substituições:

```text
chantypes.AttributeKeyData       -> constante local "packet_data" (legado)
chantypes.AttributeKeyAck        -> constante local "packet_ack" (legado)
clienttypes.AttributeKeyHeader   -> constante local "header" (legado)
chantypes.AttributeVersion       -> chantypes.AttributeKeyVersion
```

V11 mantém `AttributeKeyDataHex`/`AttributeKeyAckHex`. Os literais legados são necessários para relatar chains antigas, embora não sejam mais emitidos por v11.

Risco: médio; o parser é lossless e não pode confundir `packet_data` Classic com o `packet` protobuf de v2. Manter os testes M1.1a de classificação ambígua.

### 8. `exported.ClientState.GetLatestHeight` removido

Ocorrências de produção:

- `relayer/client.go:CreateClient`
- `relayer/packet-tx.go:(*Chain).SendTransferMsg`
- `relayer/provider/matcher.go:cometMatcher`
- `relayer/chains/cosmos/query.go:QueryConnectionHandshakeProof`
- `relayer/chains/cosmos/tx.go:ConnectionOpenTry`, `ConnectionOpenAck`, `InjectTrustedFields`
- `relayer/chains/cosmos/cosmos_chain_processor.go:clientState`
- `relayer/chains/penumbra/query.go:QueryConnectionHandshakeProof`
- `relayer/chains/penumbra/tx.go` connection handshakes e `InjectTrustedFields`
- `relayer/chains/penumbra/penumbra_chain_processor.go:clientState`
- `interchaintest/misbehaviour_test.go`

Substituição proposta: centralizar `classicClientLatestHeight(state)` num adapter testado:

- Tendermint: `state.(*tendermint.ClientState).LatestHeight`;
- Solomachine: `clienttypes.NewHeight(0, state.(*solomachine.ClientState).Sequence)`;
- outros tipos: erro explícito com type URL/tipo concreto, nunca panic.

Não espalhar type assertions sem `ok`: a prova temporária usou assertions diretas apenas para expor as demais quebras, não como implementação recomendada.

Risco: alto. A interface v11 removeu altura de propósito ao desacoplar light-client routing. Um registry de adapters deixa extensões futuras (08-wasm/attestation/custom clients) explícitas.

### 9. localhost stateless

Arquivos/símbolos:

- `relayer/chains/cosmos/module/app_module.go:AppModuleBasic.RegisterInterfaces`
- `relayer/chains/cosmos/tx.go:queryLocalhostClientState`
- `relayer/chains/cosmos/cosmos_chain_processor.go:clientState`
- `interchaintest/localhost_client_test.go`

Quebras: `localhost.RegisterInterfaces` e `localhost.ClientState` não existem em v11. A implementação v11 retorna a altura do próprio contexto e não armazena client/consensus state.

Substituição: remover o registro e `queryLocalhostClientState`; no processor, calcular `clienttypes.NewHeight(clienttypes.ParseChainID(chainID), latestBlockHeight)` para `09-localhost` e validar a semântica com a chain real. O teste in-process antigo precisa ser redesenhado, não apenas recompilado.

Risco: alto em proofs/timeouts localhost; exige teste E2E de self-relaying e sentinel proof v11.

### 10. misbehaviour endpoint removido

Arquivos/símbolos:

- `relayer/chains/cosmos/tx.go:MsgSubmitMisbehaviour`
- `relayer/chains/penumbra/tx.go:MsgSubmitMisbehaviour`
- `relayer/chains/cosmos/log.go:getFeePayer`
- `relayer/chains/penumbra/log.go:getFeePayer`

Substituição: construir `clienttypes.NewMsgUpdateClient(clientID, misbehaviour, signer)`. Remover os cases `*clienttypes.MsgSubmitMisbehaviour`; o case de `MsgUpdateClient` já devolve o signer correto.

Risco: médio/alto. Testar que uma misbehaviour concreta é empacotada no Any de `MsgUpdateClient`, aceita no endpoint v11 e ainda produz freeze/evidence esperado.

### 11. ICS-20 `DenomTrace` -> `Denom`

Arquivos/símbolos:

- `relayer/provider/provider.go:ChainProvider.QueryDenomTrace`, `QueryDenomTraces`
- `relayer/chains/cosmos/query.go:QueryDenomTrace`, `QueryDenomTraces`
- `relayer/chains/penumbra/query.go:QueryDenomTrace`, `QueryDenomTraces`
- `relayer/query.go:QueryDenomTraces`
- `cmd/tx.go` saída de denom trace
- `interchaintest/multi_channel_test.go`

Substituições:

```text
DenomTrace                       -> Denom
QueryClient.DenomTrace           -> QueryClient.Denom
QueryDenomTraceRequest           -> QueryDenomRequest
response.DenomTrace              -> response.Denom
QueryClient.DenomTraces          -> QueryClient.Denoms
QueryDenomTracesRequest          -> QueryDenomsRequest
response.DenomTraces             -> response.Denoms
GetFullDenomPath()               -> Path()
```

`ParseDenomTrace` ainda existe como deprecated e retorna `Denom`; migrar chamadas para `ExtractDenomFromPath` para não carregar dívida nova.

Risco: médio. O modelo mudou de `Path/BaseDenom` para `Trace []Hop/Base`; comparar JSON/CLI output, paginação, hash e `IBCDenom()` em traces de zero, um e múltiplos hops.

### 12. codec/registry

Arquivos/símbolos:

- `relayer/chains/cosmos/codec.go:MakeCodec`, `MakeCodecConfig`
- `relayer/chains/penumbra/codec.go:makeCodec`, `makeCodecConfig`
- `relayer/chains/cosmos/module/app_module.go:RegisterInterfaces`

Mudanças combinadas: paths SDK x/*, retirada capability/localhost/ICS-29 ModuleBasic e registro local explícito das mensagens fee legadas. A ordem de registro deve permanecer determinística e sem tipos duplicados v8/v11.

Risco: alto porque uma compilação verde não prova unpack de Any. Criar uma tabela de goldens para todas as mensagens que o relayer constrói ou inspeciona: client, connection, channel, transfer, feegrant, upgrade, ICS-29 compat e Ethermint.

## Complexidade e divisão em lotes

Contrato verificado em `scripts/check-complexity.sh`: `gocyclo v0.6.0`, `gocognit v1.2.1`, máximo 9 em ambos e exclusão somente por cabeçalho canônico gerado.

Hotspots já acima do limite que esta migração inevitavelmente toca:

| função | ciclomática/cognitiva baseline | ação para ficar <10 |
|---|---:|---|
| `relayer/client.go:CreateClient` | 21/32 | extrair seleção de altura, query/retry e construção em helpers independentes |
| `relayer/packet-tx.go:(*Chain).SendTransferMsg` | 19/22 | extrair resolução de client height, timeout e packet info |
| `relayer/provider/matcher.go:cometMatcher` | 13/20 | extrair carregamento/validação/consensus match; usar adapter de altura |
| `relayer/provider/matcher.go:checkTendermintMisbehaviour` | 7/10 | extrair comparação/expiração para reduzir cognitiva |
| `relayer/chains/parsing.go:(*PacketInfo).parsePacketAttribute` | 20/15 | tabela de handlers Classic; não adicionar branches v11/v2 |
| `relayer/chains/cosmos/log.go:getFeePayer` | 12/4 | dividir dispatch em dois helpers ou tabela por tipo; remoção de MsgSubmit sozinha ainda deixa >=10 |
| `relayer/chains/penumbra/log.go:getFeePayer` | 12/4 | idem |

Funções tocadas que hoje cabem no limite, mas têm pouca margem:

| função | baseline | regra do lote |
|---|---:|---|
| `CosmosChainProcessor.clientState` | 6/7 | extrair branch localhost; meta <=6/6 |
| `PenumbraProvider.InjectTrustedFields` | 6/7 | usar helper de altura; não adicionar type switch local |
| `CosmosProvider.InjectTrustedFields` | 6/6 | idem |
| `CosmosProvider.QueryDenomTraces` | 5/6 | preservar loop num helper de paginação |
| `cosmos.MakeCodec` / `penumbra.makeCodec` | 4/3 | registrar módulos por lista; evitar novos switches |
| `cclient.convertPubKey` | 4/1 | manter um case sr25519 delegando ao adapter |

Helpers novos propostos, todos com alvo <=5/5:

- `classicClientLatestHeight`
- `localhostHeight`
- `registerLegacyFeeInterfaces`
- `legacyFeePayerPartA` / `legacyFeePayerPartB` ou dispatch table
- `decodeDenomPage`
- `adaptSR25519PubKey`

Arquivos `.pb.go` realmente gerados continuam fora do gate; wrappers, registries e adapters manuais entram no gate mesmo quando acompanham código gerado.

## Sequência recomendada

1. **M1.1b-a, grafo/codec:** SDK/Comet/v11, paths x/*, Store v2, crisis contrib, retirar capability e registrar ICS-29 local. Gate: codec goldens e `go list -m` sem v8/Store v1/log v1.
2. **M1.1b-b, crypto:** portar sr25519 com vetores e keyring migration. Gate: assinatura/Any/address.
3. **M1.1b-c, Classic API:** adapter de altura, misbehaviour via UpdateClient, localhost stateless, eventos legados e Denom. Refatorar os sete hotspots no mesmo lote em que forem tocados.
4. **M1.1b-d, interchaintest:** fork SDK54 ou Docker-only; migrar simapp encoding (`cosmossdk.io/simapp/params` -> `github.com/cosmos/cosmos-sdk/types/module/testutil`) e retirar capability. Gate de compilação separado não pode ser dispensado.
5. Só depois remover o wire local M1.1a em favor dos tipos v11 oficiais e iniciar queries/builders operacionais v2.

## Gates de aceite

- nenhum import `github.com/cosmos/ibc-go/v8`, `cosmossdk.io/store`, `cosmossdk.io/log` ou `github.com/cosmos/ibc-go/modules/capability` nos módulos selecionados;
- nenhuma dependência `cosmossdk.io/x/feegrant`, `cosmossdk.io/x/upgrade` ou `cosmossdk.io/x/tx` antiga;
- `go list -m` fixa exatamente SDK 0.54.3, Comet 0.39.3, ibc-go/v11.2.0, Store v2.0.0 e log/v2.1.0;
- `go test -race ./...`, build root, build/release cross-platform, lint e `make complexity` sem novas violações;
- testes de wire/type URL para ICS-29 e Penumbra gerado;
- vetores sr25519 e restauração de keyring antigo;
- E2E Classic para client/connection/channel/transfer/misbehaviour/localhost e denom multi-hop;
- interchaintest compila contra o relayer local; não aceitar `github.com/cosmos/relayer/v2 v2.0.0` remoto como substituto do código em revisão.
