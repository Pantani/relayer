# M0.3 — desenho das fronteiras e QA de integração

Data: 2026-07-15  
Escopo: desenho aditivo; nenhuma substituição do parser, processor ou provider Classic neste lote  
Contrato upstream de referência: `github.com/cosmos/ibc-go/v11@v11.2.0`  
Pin que deve permanecer no M0.3: `github.com/cosmos/ibc-go/v8 v8.2.0`

## Decisão de integração

O M0.3 deve adicionar o modelo protocol-neutral e adaptadores de contrato, mas não deve ligar esses tipos ao loop de relay. A fronteira segura é um *sidecar* tipado: tipos puros e validações em `relayer/protocol`, conversões Classic explícitas junto ao provider e configuração com protocolo discriminado. O caminho `ABCI -> parsing.go -> provider.PacketInfo -> ChannelKey/cache -> packetIBCMessage.assemble -> SendMessagesToMempool` continua byte a byte e branch a branch como está.

O envelope neutro deve ser uma união discriminada, não um grande struct com campos opcionais dos dois protocolos. Para packet, proof e message, `ProtocolClassic` aceita exatamente um corpo Classic e `ProtocolV2` aceita exatamente um corpo v2; corpo ausente, corpo duplo ou protocolo divergente é erro. Para chaves usadas em mapas, as duas variantes devem ser structs comparáveis e o campo `Protocol` deve participar da igualdade.

`provider.PacketInfo` representa uma observação de packet flow, não apenas o packet on-chain: mistura altura observada, packet, ordenação e acknowledgement. A conversão correta, portanto, é `PacketInfo <-> PacketObservation`, com packet, metadados de observação e acknowledgement separados dentro do envelope. Colocar `Ack` dentro do packet neutro cristalizaria novamente um detalhe do runtime Classic.

## Mapa da fronteira atual

| estágio | produtor atual | contrato transportado | consumidor atual | acoplamento/riscos |
|---|---|---|---|---|
| observação on-chain | `CosmosChainProcessor` e `PenumbraChainProcessor` consultam bloco e tx | `[]abci.Event`, em ordem | `chains.IbcMessagesFromEvents` | Ainda há ordem e duplicatas neste ponto. É a última fronteira sem perda. |
| parser | `relayer/chains/parsing.go` converte cada evento com `sdk.StringifyEvent` e despacha apenas pelo `event.Type` | `chains.IbcMessage` e aliases de `provider.*Info` | `handleMessage` do chain processor | Classic e v2 reutilizam nomes como `send_packet`; despachar apenas pelo nome faria um evento v2 virar `PacketInfo` Classic quase vazio. `message.action` não é correlacionado. |
| retenção | `handlePacketMessage` deriva `processor.ChannelKey` | `eventType`, `ChannelKey`, sequence e `provider.PacketInfo` | `IBCMessagesCache.PacketFlow` | Chave é port/channel, sem protocolo. A mesma sequência em dois protocolos não pode compartilhar este cache. |
| decisão | `unrelayedPacketFlowMessages` correlaciona send/recv/write-ack/ack/timeout | mapas por `ChannelKey`, event string e sequence | `packetIBCMessage` | Ordenação, timeout height, channel-close e ack estão embutidos na state machine Classic. Não é ponto seguro para conversão parcial. |
| prova | `packetIBCMessage.assemble` escolhe `PacketCommitment`, `PacketAcknowledgement`, `PacketReceipt` ou `NextSeqRecv` | `provider.PacketProof` com bytes e `clienttypes.Height` | builders do `ChainProvider` | A escolha é Classic: ordered usa `NextSeqRecv`; v2 não possui channel order nem timeout height. |
| mensagem | `MsgRecvPacket`, `MsgAcknowledgement`, `MsgTimeout` do provider | `provider.RelayerMessage` contendo msg channel/v1 | `messageProcessor` | Interface não informa protocolo; o tipo concreto do provider é a única discriminação. |
| broadcast | `messageProcessor` acrescenta update-client, faz batch/single e chama `SendMessagesToMempool` | slice de `RelayerMessage` + callbacks | Cosmos/Penumbra tx provider | Batching e métricas usam semântica e labels channel/port. Não alterar no M0.3. |
| confirmação | providers convertem eventos de `TxResponse`/ABCI | `provider.RelayerTxResponse.Events` | trackers, logs, criação de IDs e Penumbra ack scan | `RelayerEvent.Attributes map[string]string` perde ordem e sobrescreve duplicatas. Em Penumbra, `parseEventsFromABCIResponse` ainda prealoca `len` e usa `append`, produzindo entradas vazias; dívida separada, não necessária ao modelo M0.3. |
| config/CLI | YAML/JSON carrega `relayer.Path`; `ChainsFromPath` aplica `PathEnd` | chain/client/connection + filtro de channel | comandos, `processor.NewPathProcessor` e validação live | Sem protocolo. Toda inicialização supõe Classic e connection/channel; v2 deve falhar antes de construir o processor Classic. |

## Pontos de inserção seguros no M0.3

### 1. Núcleo puro e sem dependências IBC

Criar somente tipos, validações, cópias defensivas e capabilities em `relayer/protocol/**`. Esse pacote não importa `provider`, `processor`, Cosmos SDK, `ibc-go/v8` nem um futuro `/v11`. `provider`, `chains`, `processor` e `relayer` podem importá-lo; o inverso é proibido para evitar ciclos.

Contratos mínimos:

- `Protocol`: `classic` e `v2`, com parse/validate explícitos;
- `PacketObservation`: protocolo, event kind, altura/posição de origem, packet discriminado e acknowledgement separado;
- `PacketKey`: protocolo + sequence + chave Classic `(srcPort, srcChannel, dstPort, dstChannel)` ou chave v2 `(sourceClient, destinationClient)`;
- `EventEnvelope`: tipo, protocolo, height/tx/event/message index e `[]Attribute` ordenado; helpers `First`, `Last` e `Values` não destroem duplicatas;
- `ProofEnvelope`: protocolo, purpose, proof bytes e altura neutra `(revisionNumber, revisionHeight)`;
- `MessageEnvelope`: protocolo, kind (`send`, `recv`, `ack`, `timeout`, client update), packet, proof e signer, com validação por kind;
- `Capabilities`: conjunto explícito. Classic declara connection/channel handshakes, ordered packets e timeout height; v2 declara client-pair packet key e timestamp-only. Capacidade não declarada não pode ser inferida de campos preenchidos.

### 2. Adaptador Classic aditivo

Adicionar conversões explícitas próximas a `relayer/provider`, sem mudar `PacketInfo`, `PacketProof`, `ChainProvider` ou `RelayerMessage`. O adaptador é a única camada nova autorizada a importar simultaneamente `relayer/protocol` e tipos v8 para converter height/order/packet.

Neste lote, nenhuma chamada do runtime deve ser redirecionada ao adaptador. Ele existe para provar que o modelo neutro consegue representar o contrato Classic sem perda e para preparar a migração posterior por fatias.

### 3. Adaptador de contrato v2 local

Representar o contrato v11.2.0 sem importar `/v11`. O packet v2 contém exatamente `sequence`, `source_client`, `destination_client`, `timeout_timestamp` e payloads; cada payload contém `source_port`, `destination_port`, `version`, `encoding` e `value`. Eventos usam os tipos `send_packet`, `recv_packet`, `write_acknowledgement`, `acknowledge_packet`, `timeout_packet` e os atributos `packet_source_client`, `packet_dest_client`, `packet_sequence`, `packet_timeout_timestamp`, `encoded_packet_hex`, `encoded_acknowledgement_hex`.

O contrato executável v11.2.0 exige exatamente um payload e um app acknowledgement, apesar da especificação mais ampla mencionar listas. Os validadores locais devem refletir o tag pinado e não prometer multi-payload. Recv, Ack e Timeout carregam packet, proof, proof height e signer; Ack também carrega acknowledgement. Connection ID, channel ID, channel order e timeout height são inválidos para a variante v2.

### 4. Config antes do runtime

`Path` pode importar `relayer/protocol` e expor o protocolo com default retrocompatível Classic. A validação do schema deve ocorrer em `validateConfig`/`ValidatePath` antes de `ChainsFromPath`, `SetPath`, queries live ou construção de `PathProcessor`.

No M0.3, um path v2 válido prova somente schema e modelo. Qualquer comando que tente iniciar relay/handshake/flush pelo processor Classic deve devolver erro estável de “runtime v2 ainda não implementado”, nunca cair implicitamente no Classic. Paths legados sem campo `protocol` continuam Classic e devem serializar sem reescrita inesperada.

### 5. Evento ordenado como contrato, sem religar o parser

O novo `EventEnvelope` deve aceitar `[]Attribute` diretamente de `abci.Event` e preservar posição e duplicatas. Não substituir ainda `IbcMessagesFromEvents` nem remover `RelayerEvent.Attributes`.

Quando a ligação ocorrer no M1, ela deve acontecer antes de `sdk.StringifyEvent`/map projection. `message.action` deve ser metadado de correlação por mensagem, não chave única de mapa. Se for necessário manter consumidores antigos de `RelayerEvent`, usar uma projeção explicitamente *lossy* para o mapa, enquanto o envelope ordenado segue como fonte de verdade. O protocolo não pode ser decidido apenas pelo nome do evento; o adapter deve validar o conjunto completo de atributos/corpo e rejeitar match ausente ou ambíguo.

## Testes de aceitação exatos para o M0.3

| ID | teste | fixture/ação | asserções obrigatórias |
|---|---|---|---|
| A1 | `TestClassicPacketInfoAdapterRoundTrip` | `PacketInfo` com height, sequence, quatro IDs port/channel, `ORDER_ORDERED`, data binária, timeout height `2-100`, timeout timestamp e ack binário | ida produz somente corpo Classic; volta é `DeepEqual`; data/proof/ack são cópias defensivas; nenhum campo v2 é inventado. |
| A2 | `TestClassicPacketEventKindsPreserved` | tabela send/recv/write-ack/ack/timeout | event kind não muda; a perspectiva de source/destination permanece igual à de `PacketInfoChannelKey`; variante desconhecida falha. |
| A3 | `TestClassicProofAdapterRoundTrip` | proof com bytes não UTF-8 e height `3-77` | bytes e revisão/altura preservados; mutar entrada/saída não cria alias; proof v2 é rejeitado pelo adapter Classic. |
| A4 | `TestEnvelopeRejectsMismatchedOrDualBodies` | protocolo Classic com corpo v2; v2 com corpo Classic; dois corpos; nenhum corpo | todos falham com erros determinísticos; nenhum fallback implícito. |
| E1 | `TestEventEnvelopePreservesAttributeOrderAndDuplicates` | attrs `[message.action=A, packet_sequence=7, message.action=B, packet_sequence=8]` | slice permanece na mesma ordem; `Values("message.action") == [A,B]`; `Values("packet_sequence") == [7,8]`; cópia defensiva. |
| E2 | `TestEventEnvelopeKeepsMessageAndEventPositions` | dois eventos do mesmo tipo/sequence em `messageIndex` 0 e 1 | envelopes e chaves de observação não se fundem; índices sobrevivem a JSON round-trip se os envelopes forem persistíveis. |
| E3 | `TestLegacyAttributeProjectionIsExplicitlyLossy` | duas entradas com mesma chave | envelope conserva ambas; projeção de compatibilidade documenta e testa a regra (recomendada: último valor vence); código neutro nunca usa a projeção para correlação. |
| I1 | `TestPacketKeysAreProtocolIsolated` | Classic e v2 com sequence 7 e ports iguais; dois pares de clients v2 com sequence 7 | todas as chaves de protocolos/clients distintos são diferentes e comparáveis como map keys. |
| I2 | `TestCapabilitiesDoNotLeakAcrossProtocols` | matriz completa Classic/v2 | Classic: connection/channel/order/height timeout=true e client-pair=false; v2: client-pair/timestamp-only=true e connection/channel/order/height timeout=false. Capability desconhecida=false. |
| I3 | `TestAdaptersRejectForeignProtocol` | cada envelope enviado ao adapter oposto | erro antes de qualquer conversão; nenhum envelope parcialmente preenchido é retornado. |
| V1 | `TestV2PacketMatchesIBCGoV11_2_0Contract` | packet sequence 7, clients A/B, timestamp, um payload com ports/version/encoding/value | todos os campos e tags esperados são exatos; zero sequence/timestamp/client, payload vazio e zero ou >1 payload falham. |
| V2 | `TestV2EventConstantsMatchIBCGoV11_2_0` | tabela copiada de `modules/core/04-channel/v2/types/events.go` do tag | cinco event types e seis attribute keys coincidem literalmente; Classic attrs não são aceitos como v2. |
| V3 | `TestV2MessageValidationMatrix` | send/recv/ack/timeout válidos e variações sem proof/height/signer/ack | Recv exige commitment proof; Ack exige ack + ack proof; Timeout exige unreceived proof; nenhum aceita timeout height/order/channel/connection. |
| C1 | `TestLegacyPathDefaultsToClassicWithoutRewrite` | YAML/JSON atual sem `protocol` | parse seleciona Classic; marshal preserva forma legada/`omitempty`; validação Classic existente continua passando. |
| C2 | `TestV2PathNeverStartsClassicRuntime` | path schema v2 válido passado a comando de start/flush/handshake | erro de capability/runtime antes de `SetPath`, query de connection, cache ou broadcast; mock provider registra zero chamadas. |
| G1 | import/pin audit | `go list -m`, `rg` e `go mod tidy` check | `go.mod/go.sum` não ganham `/v11`; pin v8 permanece; pacote protocol não importa SDK/IBC/provider/processor. |
| G2 | regressão Classic | `go test ./...`, build, testes de parsing Classic e testes de processor existentes | mesmos resultados do M0.2; nenhum teste Classic é alterado para aceitar comportamento diferente. |
| G3 | complexidade focada | `gocyclo -over 9` e `gocognit -over 9` nos arquivos novos/tocados | saída vazia; máximo 9 em ambas; sem `nolint`, baseline ou exclusão nova. |

## O que não deve ser alterado no M0.3

- `chains.IbcMessagesFromEvents` e `ParseIBCMessageFromEvent` não devem ser substituídos;
- `processor.ChannelKey`, `IBCMessagesCache`, `unrelayedPacketFlowMessages`, `packetIBCMessage.assemble` e message tracker não devem receber branches v2;
- `provider.ChainProvider` não deve ganhar métodos v2 enquanto queries/builders v11 não existirem em uma fatia vertical;
- `SendMessagesToMempool`, batch/retry/callbacks e métricas channel/port não devem ser generalizados agora;
- o módulo não deve importar `/v11`, e o adaptador local não deve fingir que produz protobuf v11;
- um path v2 não pode ser executado pelo runtime Classic por default, detecção de event type ou preenchimento parcial.

## Ownership para implementação paralela

| owner | arquivos autorizados | entrega | não tocar |
|---|---|---|---|
| modelo protocol-neutral | `relayer/protocol/**` | unions discriminadas, keys comparáveis, event attrs ordenados, proof/message/capabilities, validações e testes | `provider`, `chains`, `processor`, config/CLI |
| config/schema | `relayer/path.go`, `relayer/pathEnd.go`, testes focados e helpers mínimos de `cmd/config.go` | default Classic, schema v2, mutual exclusion, round-trip e guard de runtime | parser, cache, provider message assembly |
| adaptadores | novos arquivos/testes aditivos próximos a `relayer/provider` | round-trip Classic e contrato local v2; cópias defensivas | interface `ChainProvider`, `PacketInfo` existente, runtime processor |
| QA fan-in | nenhum arquivo de produção | import audit, unit/build/lint, complexidade focada/global e diff Classic | correção silenciosa do desenho de outro owner |

Dependência de merge: modelo primeiro; config e adaptadores compilam contra esse contrato depois. Se um owner precisar mudar um tipo do modelo, deve coordenar antes de editar o mesmo arquivo. Arquivos do processor ficam sem owner no M0.3 para impedir ligação acidental.

## Riscos de integração e gates

| risco | severidade | gate |
|---|---|---|
| Nomes de eventos Classic/v2 iguais causarem parse Classic silencioso | crítica | nenhuma seleção por nome isolado; adapter valida protocolo/corpo completo; path v2 bloqueado antes do runtime atual. |
| União permissiva aceitar combinações impossíveis | alta | validação “exactly one matching body” e testes A4/I3. |
| Perda de ordem/duplicatas continuar no caminho de confirmação | alta para M1 | `EventEnvelope` é fonte de verdade; projeção map é marcada lossless=false/lossy; ligação fica rastreada para M1. |
| Converter `PacketInfo` como se fosse wire packet misturar ack/metadado | alta | `PacketObservation` separa packet, observation metadata e acknowledgement; round-trip A1. |
| V2 local divergir do tag real | alta | constantes/validação pinadas a v11.2.0, teste V1/V2, fonte e versão documentadas; atualizar por diff explícito no upgrade. |
| Go importar duas major versions cedo | alta | audit G1 e `-mod=readonly`; nenhuma dependência `/v11` no M0.3. |
| Config v2 cair no processor Classic | crítica | guard C2 antes de provider/query/broadcast com mock de zero chamadas. |
| Generalização precoce dos hotspots aumentar regressão/complexidade | alta | lista “não alterar”, diff audit e regressão G2; qualquer branch v2 em `processor` reprova o lote. |
| `RelayerEvent` dual map/slice ter fontes divergentes futuramente | média | não adicionar dual source no runtime neste lote; no M1, construir ambos de uma única lista ordered e testar projeção. |
| Penumbra assumir Classic ou propagar eventos vazios | média | declarar capability Classic; bug de prealloc/append em issue/lote separado, sem bloquear o sidecar de tipos. |

## Critério de aprovação deste desenho

O M0.3 passa quando o novo modelo e adapters compilam sob os pins atuais, todos os testes A1–G3 aplicáveis passam, paths legados continuam Classic, um path v2 não consegue alcançar o runtime Classic, e o diff não muda parser/cache/proof selection/broadcast existentes. Isso entrega a fundação do protocolo; não entrega relay IBC v2 operacional, que começa quando a fatia M1 ligar `event ordered -> v2 key/state -> v2 proof/message -> broadcast/confirmation` de ponta a ponta.
