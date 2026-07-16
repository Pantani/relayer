# Roadmap de modernização: IBC v2 + Cosmos SDK atual

Data da auditoria: 2026-07-15  
Snapshot analisado: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Escopo: inventário funcional e matriz completa de lacunas; nenhuma alteração em código de produção.

## Conclusão executiva

O relayer atual não tem uma implementação parcial de IBC v2: sua arquitetura central é IBC Classic, baseada em `connection/channel/port`, ordenação de canal, timeout por altura e caches indexados por `ChannelKey`. Portanto, migrar de `ibc-go/v8` para `ibc-go/v11` é necessário, mas insuficiente. A entrega exige primeiro separar o modelo interno neutro de protocolo dos adaptadores Classic e v2; depois implementar o fluxo v2 completo `evento -> parsing -> estado -> prova -> mensagem -> broadcast -> ack/timeout/retry`.

O alvo de dependências pinado nesta auditoria é **ibc-go v11.2.0 + Cosmos SDK v0.54.3 + CometBFT v0.39.3 + Go 1.25.9**. `ibc-go v11.2.0` foi publicado durante a auditoria, em 2026-07-15 15:54:50 UTC. Embora seja uma tag estável, precisa de uma janela de soak e validação de interoperabilidade antes de produção.

O critério solicitado de complexidade também reprova o snapshot: 158 de 1.327 funções manuscritas têm complexidade ciclomatica ou cognitiva `>= 10`; 151 são de produção. Os maiores hotspots são justamente o state machine, cache, parsing, broadcast e CLI Classic. Refatorá-los mecanicamente antes de definir a coexistência Classic/v2 criaria retrabalho. O plano abaixo transforma esses pontos junto com contratos e testes, mantendo o gate final estrito em **máximo 9 para ambas as métricas**. O baseline completo está em [`02_complexity_engineer_baseline.md`](./02_complexity_engineer_baseline.md).

## Verdade de versões em 2026-07-15

| Componente | Alvo estável pinado | Pré-release observada | Decisão |
|---|---|---|---|
| ibc-go | [`v11.2.0`](https://github.com/cosmos/ibc-go/releases/tag/v11.2.0), commit `cfc072e53eee42b2ab804cd4344ba610016f793c`, publicado 2026-07-15 | A API de releases devolvia `v10.3.0-rc.0` (2025-06-07), mais antiga que a estável | Implementar contra v11.2.0; não usar v9, que foi retraída/substituída por v10 |
| Cosmos SDK | [`v0.54.3`](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.54.3), publicado 2026-05-05 | `v0.53.0-rc.3` (2025-04-10), mais antiga que a estável | Migrar conjuntamente; v11 requer as APIs novas de log/store e remove APIs antigas |
| CometBFT | `v0.39.3`, pin transitivo do SDK v0.54.3 | Não usar a série 1.x como alvo implícito | Permanecer na família suportada pelo SDK alvo |
| Go | `1.25.9`, declarado por ibc-go v11.2.0 e SDK v0.54.3 | n/a | Atualizar toolchain, workflows, imagens e matriz CI antes do upgrade |
| Especificação IBC v2 | [`cosmos/ibc` commit `6f4bd557`](https://github.com/cosmos/ibc/commit/6f4bd55738c95b9e85220c1382d87d0e7b3a55aa), 2026-07-10 | O frontmatter ainda marca IBC v2 como `EXPERIMENTAL` | Usar o tag implementado de ibc-go como contrato executável e a spec como modelo; registrar divergências |
| Relayer de referência v2 | [`cosmos/ibc-relayer v1.1.0`](https://github.com/cosmos/ibc-relayer/releases/tag/v1.1.0), commit `1fefe7b7f208ba19fb4405a473b1de5b64187980`, 2026-05-26 | `v1.1.0-rc`, mais antiga que a estável | Referência operacional para Cosmos↔EVM, retries, estado durável, batching e proof API; não copiar seus pins de SDK |

Fontes primárias complementares: [IBC v2 overview](https://docs.cosmos.network/ibc/latest/spec/IBC_V2/README), [algoritmos de relayer ICS-18](https://docs.cosmos.network/ibc/latest/spec/relayer/ics-018-relayer-algorithms/README), [eventos para relayers](https://docs.cosmos.network/ibc/latest/ibc/relayer), [updates e misbehaviour](https://docs.cosmos.network/ibc/latest/light-clients/developer-guide/updates-and-misbehaviour), [upgrade guide SDK](https://docs.cosmos.network/sdk/latest/upgrade/upgrade), [release notes SDK](https://docs.cosmos.network/sdk/latest/upgrade/release) e [changelog ibc-go](https://docs.cosmos.network/ibc/latest/changelog/release-notes).

### Mudanças de base que não podem ser tratadas como bump trivial

- O módulo muda de `github.com/cosmos/ibc-go/v8` para `/v11`; SDK v0.54 migra para `cosmossdk.io/log/v2`, `github.com/cosmos/cosmos-sdk/store/v2`, address codecs configuráveis e APIs novas de client/misbehaviour.
- `MsgSubmitMisbehaviour` foi removida; misbehaviour passa como `ClientMessage` em `MsgUpdateClient`. A recuperação de clients usa `MsgRecoverClient`.
- IBC v2 elimina handshakes de connection/channel. `MsgRegisterCounterparty` associa clients e prefixos; `MsgUpdateClientConfig` controla a allowlist de relayers por client.
- Um packet v2 usa `source_client`, `destination_client`, `sequence`, `timeout_timestamp` em segundos e `repeated Payload`; commitment/receipt/ack são indexados por client e sequência. Não há `channel order`, timeout por altura ou `nextSeqRecv`.
- Os eventos v2 carregam `encoded_packet_hex` e `encoded_acknowledgement_hex`; o índice `message.action` precisa ser preservado para separar múltiplas mensagens na mesma transação.
- Há uma divergência importante no próprio tag v11.2.0: changelog/spec descrevem múltiplos payloads com execução atômica, mas [`MsgSendPacket.ValidateBasic`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/v2/types/msgs.go) exige `len(Payloads) == 1`. Assim, multi-payload é um **gate de verificação/interoperabilidade**, não uma capacidade já prometida. Acknowledge assíncrono com mais de um payload também não deve ser assumido.

## Inventário do snapshot atual

| Área | O que existe | Evidência | Consequência para v2 |
|---|---|---|---|
| Dependências | Go 1.21, ibc-go v8.2.0, SDK v0.50.11, Comet v0.38.12, store v1 | `go.mod:3-23` | Quatro gerações de API/toolchain atrás do alvo |
| Tipos do provider | Imports diretos de transfer/client/connection/channel v8; `PacketInfo` contém port/channel/order/data/timeout height | `relayer/provider/provider.go:8-18,82-108` | Interface pública interna cristaliza IBC Classic |
| Configuração de path | `Path` tem filtro de canais; cada `PathEnd` tem chain/client/connection | `relayer/path.go:91-109`; `relayer/pathEnd.go:9-15` | v2 precisa de modo de protocolo, clients e prefixos sem connection/channel |
| Eventos | Parser reconhece packet/channel/connection/client Classic e Client ICQ | `relayer/chains/parsing.go:56-89` | Nenhum evento/atributo v2 nem separação por `message.action` |
| Estado/cache | Packet flow/state indexados por `ChannelKey`; caches separados de handshake connection/channel | `relayer/processor/types.go:87-168` | Chave e lifecycle são incompatíveis com v2 |
| Montagem | Recv/Ack/Timeout Classic; timeout escolhe `NextSeqRecv` para ordered e receipt para unordered | `relayer/processor/types_internal.go:46-92` | v2 precisa de proofs e mensagens próprias e não tem ordered/unordered |
| Consulta de eventos | Combina block search e tx search; converte atributos de resultado em `map[string]string` | `relayer/chains/cosmos/query.go:47-115,176-190` | O mapa perde ordem e atributos duplicados; não sustenta múltiplas ações/payloads com segurança |
| Cosmos provider/config | RPC, backup RPC, gas, feegrant e broadcast; sem endpoint gRPC/proof API/finality/protocolo | `relayer/chains/cosmos/provider.go:36-65` | Config insuficiente para v2 e Eureka |
| Métricas | Observed/relayed, height, wallet/fees, falhas, client expiry e backlog; labels `channel/port` | `relayer/processor/metrics.go:10-77` | Sem dimensões por protocolo/client/estágio/resultado/retry/prova |
| Dynamic fee | Query Osmosis hardcoded interpreta resposta protobuf como decimal; teste acessa RPC público | `relayer/chains/cosmos/fee_market.go:12-60`; `fee_market_test.go:42-55` | Baseline de testes não é determinístico e já falha por drift de resposta |

### Capacidades Classic que precisam permanecer cobertas durante a transição

O código atual possui create/update/upgrade/misbehaviour de clients, handshakes connection/channel open e close, packet recv/ack/timeout/timeout-on-close ordered e unordered, flush, ICS-20 transfer, ICS-29 fee/payee, ICA, localhost, Stride client ICQ, feegrant, backup RPC, channel filters, stuck-packet recovery, Prometheus e provider Penumbra. Elas não podem desaparecer silenciosamente: cada uma deve ser classificada como **mantida**, **adaptada**, **legado opcional** ou **removida com migration note**. Em particular, ICS-29 não faz parte do stack novo de ibc-go v11; deve ficar somente como adaptador para cadeias Classic antigas se esse suporte for requisito explícito.

## Limite de responsabilidade

| Responsável | Deve entregar | Não deve ser atribuído ao relayer |
|---|---|---|
| Este repositório | Observar e ordenar eventos, preservar índices/atributos, atualizar clients, consultar estado/provas, montar Recv/Ack/Timeout, assinar/broadcast, confirmar, retry idempotente, checkpoint, CLI/config/status/metrics e testes interchain | Implementar keeper/middleware dentro da chain ou validar criptografia dentro do light client |
| Chain/app Cosmos | Wire ibc-go v11, routers/middleware v2, ICS-20 v2, GMP, callbacks v2 e rate limit v2; manter PFM somente no fluxo ICS-20 Classic enquanto o tag não oferecer adapter v2; expor gRPC/RPC/proofs/eventos corretos; governança/genesis | O relayer não cria suporte v2 numa chain que só expõe Classic, nem pode tornar PFM Classic compatível com v2 |
| Light client on-chain | Tendermint/08-wasm/attestation/Ethereum e suas regras de verify/update/misbehaviour/recovery | O relayer apenas produz/transporta client messages e provas |
| Stack Eureka/EVM | Contratos, SP1 prover, proof API, finality/reorg semantics e permissões de destino | Não incorporar prover/contratos no processo Cosmos por acidente |
| Operador | Registrar counterparties, autorizar signer na config do client, manter gas/keys/endpoints e políticas de finality/retry | Empty allowlist é permissionless; configuração errada deve falhar em preflight, não em loop infinito |

## Matriz completa de lacunas

Prioridades: `P0` bloqueia qualquer caminho v2; `P1` bloqueia produção Cosmos↔Cosmos; `P2` completa stack/app/operabilidade atual; `P3` é expansão opcional. Marcos: `M0` fundação, `M1` IBC v2 Cosmos completo mínimo, `M2` apps/light clients/produção, `M3` Eureka Cosmos↔EVM, `M4` ecossistemas futuros.

| capacidade | fonte/versao | estado atual | lacuna | componente | dependencia | teste | prioridade | marco |
|---|---|---|---|---|---|---|---|---|
| Toolchain e módulos v11 | ibc-go v11.2.0; SDK v0.54.3 | Go 1.21, ibc-go v8.2, SDK 0.50.11 | Go 1.25.9, `/v11`, log/v2, store/v2, Comet 0.39.3, regeneração/adaptação de codecs | build/CI/provider | Nenhuma | `go build/test`, module verify, duas versões mínimas de macOS/Linux | P0 | M0 |
| Gate ciclomatico+cognitivo `<10` | contrato do projeto; baseline 2026-07-15 | 158 violações, máximo 48/169 | Tooling pinado, inventário gerado e gate obrigatório sem baseline/nolint | CI/todos | Modelo v1/v2 definido antes dos hotspots | zero saída de `gocyclo -over 9` e `gocognit -test -over 9` | P0 | M0-M2 |
| Modelo interno neutro | IBC v2 spec + Classic | Tipos e interfaces importam v8 diretamente | `Protocol`, identificadores e packet/event/proof/message envelopes versionados; adaptadores Classic/v2 | provider/core | Toolchain | Contract tests dos dois adaptadores | P0 | M0 |
| Compatibilidade Classic explícita | ibc-go linhas suportadas v10/v11; chains legadas | v8.2 hardcoded | Matriz de chains suportadas; leitura/relay Classic preservados ou deprecation documentada | provider/processor | Modelo neutro | Classic ordered/unordered, close, timeout, upgrade | P0 | M0 |
| Política ICS-29 legado | migração ibc-go v10/v11 | Fee middleware/payee presente | Decidir suporte somente a chain Classic antiga; remover do core novo e documentar | app adapter/CLI | Matriz Classic | Fee packet em chain legada ou teste de remoção/migration error | P2 | M2 |
| Dynamic fee tipado/determinístico | SDK/Osmosis API atual | Parser decimal sobre bytes protobuf; RPC público | Decodificar response protobuf por tipo/version; fixture local; fallback observável | Cosmos provider | SDK alvo | Unit fixture + integração opt-in; sem rede no `go test ./...` | P0 | M0 |
| Config de path por protocolo | IBC v2 Counterparty | Path exige connection e filtro channel | `protocol` com valor `classic` ou `v2`, client pair, prefix, app/port/payload filter; validações mutuamente exclusivas | config/CLI | Modelo neutro | Round-trip YAML, schema migration e invalid combinations | P0 | M0 |
| Query de CounterpartyInfo | client/v2 v11.2 | Ausente | Consultar counterparty por client e validar client/prefix simétricos | provider/query | gRPC v2 | Mock + duas chains com mismatch | P0 | M1 |
| MsgRegisterCounterparty | client/v2 v11.2 | Ausente; usa handshakes connection/channel | Builder, signer, broadcast, status e idempotência | provider/msg/CLI | CounterpartyInfo | Registro bilateral, replay NOOP/falha clara, prefix incorreto | P0 | M1 |
| Config/allowlist do relayer | `MsgUpdateClientConfig`, client/v2 | Ausente | Query Config, preflight de signer, comando update e diagnóstico permissionless/forbidden | provider/config/CLI | Client v2 | Allowed, denied e empty permissionless | P0 | M1 |
| Update/misbehaviour genérico | ibc-go v11; client developer guide | Fluxo v8 e `MsgSubmitMisbehaviour` | Produzir `ClientMessage` para `MsgUpdateClient`; desacoplar tipo Tendermint | light-client adapter | SDK/v11 | Header update, duplicate, stale, misbehaviour | P0 | M1 |
| MsgRecoverClient | ibc-go v11 | Ausente; PR antiga só traz ideia de fork/halt | Query subject/substitute, builder e CLI com preflight | provider/msg/CLI | Update genérico | Frozen/expired recovery + rejeições | P1 | M2 |
| 08-wasm | ibc-go v11 + wasm client | Só scaffolding/test imports; branch `steve/wasm` não integrada | Codec/client message/query/proof adapter formal e matriz de checksums | light-client adapter | Modelo neutro | Interchain 08-wasm update/misbehaviour/recovery | P2 | M2 |
| Attestation client | ibc-go v11.0 | Ausente | Capability adapter para client messages e status; crypto fica on-chain | light-client adapter | Interface genérica | Fixture oficial + chain integrada quando disponível | P2 | M2 |
| Localhost regressions | ibc-go v11 | Tratamento especial Classic | Revalidar ou isolar; não misturar sentinela Classic com proof v2 | processor/provider | Modelo neutro | Localhost Classic e comportamento v2 documentado | P2 | M2 |
| Ingestão dos eventos v2 | channel/v2 v11.2 | Switch só Classic | Reconhecer send/recv/write_ack/ack/timeout e campos client/sequence/timestamp/hex | events/parser | Tipos v2 | Golden events oficiais para cada fase | P0 | M1 |
| Preservação de `message.action` | docs oficiais de relayer | `map[string]string` perde duplicatas/ordem | Estrutura ordered multimap + action index; correlacionar evento à mensagem certa | events/parser | Ingestão v2 | Tx com múltiplas Msgs do mesmo tipo e atributos repetidos | P0 | M1 |
| Decode seguro de packet/ack hex | eventos v2 | Ausente | Hex/protobuf decode com limites, version/encoding e erro tipado; nunca confiar em attrs soltos | parser/security | Ingestão v2 | Truncado, oversized, invalid proto, unknown encoding/version | P0 | M1 |
| Backfill, gaps e reorg | ICS-18; Comet 0.39 | Busca concorrente sem checkpoint durável | Cursor por chain/protocolo, detecção de gap, finality policy, reprocessamento idempotente | chain processor/state | Storage/checkpoint | Restart, RPC failover, missed block, short reorg | P1 | M2 |
| Fixtures determinísticas de eventos | protos v11.2 | Testes focam Classic | Corpus block/tx oficial com action index, multi-msg, async ack e failures | tests | Parser v2 | Golden decode sem rede | P0 | M1 |
| Chave de packet v2 | IBC v2 core | `ChannelKey` + sequence | Chave `(protocol, srcClient, dstClient, sequence)`; payloads ordenados; direction explícita | processor/state | Modelo neutro | Colisões entre clients/protocolos e restart | P0 | M1 |
| State machine v2 | IBC v2 core | State machine Classic misturada a handshakes | `Send -> Recv -> WriteAck -> Ack` XOR `Timeout`, estados in-flight/done/failure/NOOP | processor | Chave v2 | Tabela de todas transições e transições inválidas | P0 | M1 |
| Timeout somente timestamp | IBC v2 core | Height+timestamp e branch ordered/unordered | Segundos UTC, clock/finality tolerance, sem `nextSeqRecv`/channel order | processor/query | Clock/finality | Boundary before/at/after expiry e skew | P0 | M1 |
| Idempotência/ResponseResultType | channel/v2 v11.2 | Sucesso/falha orientados a tx Classic | Interpretar NOOP/SUCCESS/FAILURE por mensagem e limpar/reter cache corretamente | processor/broadcast | Msg v2 | Duplicate recv/ack/timeout e batch parcial | P0 | M1 |
| Multi-payload atomicidade | spec/changelog vs `MsgSendPacket.ValidateBasic` v11.2.0 | Um `Data` Classic | Não prometer; testar keeper/API/tag real, preservar ordem e versões, definir fallback 1-payload | provider/processor/interoperability | Decisão upstream | Enviar 1 e >1 payload; verificar atomic rollback e resultado por tag | P1 | M2 |
| Async acknowledgement | IBC v2/channel v11 | Classic write ack parsing | Correlacionar write posterior; documentar/rejeitar combinação multi-payload não suportada | processor/events | State machine | Sync/async ack, restart entre recv/write, timeout race | P1 | M2 |
| Separação Classic/v2 | IBC v2 design | Lifecycle central mistura packet/connection/channel | Estratégias separadas; nenhuma conversão implícita de connection/channel Classic em v2 | processor | Modelo neutro | Ambos protocolos simultâneos sem cross-cache | P0 | M1 |
| Alias v1 channel para packet v2 | ibc-go v11 release notes | Ausente | Verificar e expor alias somente se keeper/tag suportar; não inferir por nome | config/query | Chain capability | Compatibilidade positiva e chain sem alias | P2 | M2 |
| Queries channel/v2 | v11.2 Query RPC | Só commitment/receipt/ack Classic | NextSequenceSend, commitment(s), ack(s), receipt, unreceived packets/acks | provider/query | gRPC endpoint | Query fixtures e duas chains reais | P0 | M1 |
| Merkle paths/proofs v2 | IBC v2 core | Host keys v8 channel-centric | Paths client+sequence, prefix e proof height; validação de proof freshness | provider/proof | CounterpartyInfo | Valid/invalid/stale/wrong-prefix proofs | P0 | M1 |
| Builders Recv/Ack/Timeout v2 | channel/v2 Msg RPC | Apenas mensagens Classic | Builders type-safe com packet, proof, proof height e signer | provider/msg | Queries/proofs | Proto golden + deliver em chain v11 | P0 | M1 |
| Send/Transfer v2 | transfer v2 em ibc-go v11 | CLI ICS-20 Classic | Expor envio v2 apenas como operação de usuário; relayer observa e completa o packet | CLI/app adapter | Chain ICS20 v2 | Fungible token happy path/timeout/refund | P1 | M2 |
| GMP / ICS27-GMP | ibc-go v11.0, IBC v2 only | ICA Classic; sem GMP | Core deve relatar payload/version/ack sem conhecer app; comandos/status opcionais | app awareness | Chain GMP | Call success/error, auth, timeout e ack decode | P2 | M2 |
| Packet Forward Middleware Classic | [`ibc-go v11.2.0 PFM`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/apps/packet-forward-middleware/ibc_middleware.go) importa channel/port Classic e implementa handshakes Classic; não há pacote `/v2` | Não inventariado no core | Cobrir PFM como comportamento ICS-20 Classic A→B→C; monitorar hops/acks/timeouts; registrar PFM v2 como limitação upstream e não expor configuração que prometa esse suporte | app awareness/metrics | Chain PFM Classic | A→B→C Classic success, timeout/refund/failure; teste de capability rejeita `protocol: v2` + PFM | P2 | M2 |
| Rate limiting | ibc-go v11.2 | Ausente | Transparente no relay; classificar quota rejection, async accounting e retry não-infinito | app awareness/retry | Chain rate-limit | Send/recv/ack/timeout, quota reset e v1/v2 | P2 | M2 |
| Callbacks substituindo hooks | SDK/ibc-go latest | Sem suporte explícito | Preservar ack/error e gas behavior; métricas/status sem interpretar callback como falha de transporte | app awareness | Chain callbacks | Success, callback revert/OOG e async | P2 | M2 |
| Classificação de retry | ICS-18 + ibc-relayer v1.1 reference | Retry/batching Classic central e complexo | Taxonomia transient/permanent/NOOP, backoff+jitter, limite, account sequence refresh | broadcast/retry | State machine | RPC down, mempool, wrong sequence, OOG, invalid proof, forbidden | P0 | M1 |
| Confirmação/finality/batch parcial | ibc-relayer v1.1 reference | Tx response Classic; cache tracking complexo | Confirmar inclusão/finality, mapear resposta por msg, reorg/rebroadcast sem duplicar | broadcast/state | Checkpoint | Batch com 1 falha, dropped tx, reorg e timeout | P1 | M2 |
| Estado durável e restart | ICS-18 + referência v1.1 | Caches em memória | Checkpoint versionado e migração; reconsulta on-chain como truth source | processor/storage | State machine | Kill -9 em cada fase, corrupt/old checkpoint | P1 | M2 |
| Batching/remote signing/cost | ibc-relayer v1.1 reference | Batch e keyring/feegrant locais | Capabilities opcionais, signer remoto e custo por tx/payload sem acoplar ao core | broadcast/ops | API design | Signer unavailable, partial signing, gas/cost accounting | P3 | M3 |
| CLI v2 lifecycle | channel/client v2 | `link/create connection/channel/flush` Classic | `init/register-counterparty/validate/start/flush/status` protocol-aware; erro claro para operação inválida | CLI | Config/provider v2 | Golden help + e2e de cada comando | P1 | M1 |
| Config operacional v2 | ICS-18/Eureka | Sem gRPC/proof API/finality | Endpoints gRPC/proof API, client pair/prefix, confirmations/finality, batch/retry, relayer allowlist | config | Schema v2 | Migration, secrets redaction, validation offline/online | P1 | M1-M3 |
| Métricas v2 | Prometheus best practice | Labels channel/port e contadores básicos | Labels bounded por protocol/client/stage/result; latency, retry, NOOP, proof age, client update, gas/cost, backlog | metrics | State machine | Registry assertions e cardinality budget | P1 | M2 |
| Logs estruturados/OpenTelemetry | SDK v0.54 guide | Zap local, sem trace distribuído | Correlation id chain/client/sequence/tx/action, spans event→proof→broadcast→finality | observability | log/v2 | Log/trace assertions sem dados sensíveis | P2 | M2 |
| Unit/golden v11 | Protos oficiais v11.2 | Testes v8 | Golden protobuf/event/proof/message e state transition tables | tests | M0/M1 code | Sem rede, race e fuzz seed corpus | P0 | M1 |
| Interchain Cosmos↔Cosmos v2 | ibc-go v11.2 | Só Classic integration | Duas chains v11, counterparty registration, update clients, relay packet/ack/timeout | interchaintest | Chain images v11 | Success, ack error, timeout, duplicate, restart | P0 | M1 |
| Matriz apps v2 atual | v11.0-11.2 | ICS20/ICA Classic | ICS20 v2, GMP, callbacks v2 e rate limit v1/v2; async/multi-payload gates | interchaintest | Chain modules e capabilities reais do tag | Cenários positivos/negativos por middleware; capability matrix pinada pelo tag | P1 | M2 |
| Fault injection | operação de produção | Parcial/ausente | RPC/gRPC failover, latency, reorg, stale proof, gas/sequence, signer e disk failures | tests/ops | Checkpoint/retry | Deterministic chaos suite e SLO assertions | P1 | M2 |
| Fuzz/security | encoded packet/ack + untrusted RPC | Ausente para v2 | Limits, unknown versions, malformed proofs/hex/proto, log injection, memory bounds | security/tests | Parser/provider | Go fuzz + corpus regressions, race, leak check | P1 | M2 |
| Eureka Cosmos↔EVM | ibc-relayer v1.1; solidity-ibc-eureka | Ausente | Provider EVM/proof API, finality/reorg, contract events/messages, signer e cost | provider/eureka | Contratos/prover/proof API | Devnet + public testnet round trip e recovery | P2 | M3 |
| SVM/Solana | referência tem scaffolding, não suporte estável declarado | Ausente | Rastrear upstream; não colocar no gate de “IBC atual completo” antes de release estável | future provider | Upstream stable release | Acceptance definido somente após contrato estável | P3 | M4 |

## Marcos e gates de aceitação

### M0 — Fundação e baseline confiável

- Toolchain Go 1.25.9 e módulos v11/SDK 0.54.3 compilam em Linux/macOS; `go test ./...` não acessa internet.
- O teste de dynamic fee usa response protobuf tipada e fixture determinística.
- Existe um modelo interno protocol-neutral com adaptadores Classic/v2 e config schema versionado.
- A suíte Classic de caracterização passa antes/depois. Nenhum comportamento é removido sem decisão e migration note.
- O gate de complexidade está pinado e reporta o inventário; cada arquivo tocado fica `<10/<10`, sem `nolint`. O repositório inteiro chega a zero violações no máximo até o final de M2.

### M1 — Cosmos↔Cosmos IBC v2 mínimo completo

- Duas chains v11 registram counterparties, validam allowlist e mantêm clients atualizados.
- O relayer ingere eventos com `message.action`, decodifica packet/ack, consulta proofs v2 e executa Recv, Ack e Timeout.
- State machine é idempotente, entende NOOP/SUCCESS/FAILURE e sobrevive a restart sem duplicar transação.
- Interchaintest cobre happy path, ack de erro, timeout, duplicate, wrong prefix/proof, signer negado e RPC failover.
- Classic e v2 rodam simultaneamente em paths distintos sem compartilhar chaves/cache.

### M2 — Cobertura do stack IBC atual e readiness de produção

- ICS20 v2, GMP, callbacks v2 e rate limit v1/v2 passam a matriz positiva/negativa; 08-wasm, attestation e recover-client têm adapters/testes conforme disponibilidade de chain.
- PFM passa a matriz multi-hop somente em ICS-20 Classic; paths v2 rejeitam essa capability enquanto não existir adapter upstream.
- Multi-payload permanece desabilitado por default até o teste do tag/keeper/API demonstrar suporte; a divergência upstream fica documentada. Async ack e sua combinação com payloads têm política explícita.
- Finality/reorg, batch parcial, retry taxonomy, checkpoint/migração, fault injection, fuzz, métricas e traces atendem SLOs definidos.
- Todas as 1.327+ funções manuscritas têm ciclomatica e cognitiva no máximo 9.

### M3 — Eureka Cosmos↔EVM

- Usar componentes estáveis pinados: [solidity-ibc-eureka `solidity-v3.0.2`](https://github.com/cosmos/solidity-ibc-eureka/releases/tag/solidity-v3.0.2), proof API `proof-api-v0.8.1`, SP1 programs `v2.0.0` e cw ICS08 Ethereum `v1.3.0`.
- Devnet e testnet comprovam packet round-trip, timeout, reorg/finality, prover indisponível, retry, remote signing e custo observável.
- O provider EVM é isolado por capability; não introduz tipos EVM no processor neutro nem regressão Cosmos/Penumbra.

### M4 — Novos ecossistemas

- Somente abrir gate de SVM/Solana ou outro destino após release upstream estável, contrato/proof source documentado e suite de interoperabilidade reproduzível.

## Sequenciamento com a redução de complexidade

Os lotes de complexidade precisam seguir o desenho v2, não antecedê-lo nos hotspots:

1. Gate/toolchain, fixtures e testes de caracterização; decompor testes gigantes e codecs isolados.
2. Config/CLI builders e modelo protocol-neutral.
3. Adaptadores de eventos/query/proof/msg Classic e v2.
4. State machine v2 e separação do lifecycle Classic; só então quebrar `getMessagesToSend` (42/99), `queuePreInitMessages` (48/90), `shouldTerminate` (45/82), `mergeMessageCache` (26/70) e `queuePendingRecvAndAcks` (34/63).
5. Broadcast/retry/checkpoint; decompor `buildMessages`, `SendMsgsWith`, `trackAndSendMessages` e batch mantendo ordem dos efeitos.
6. Penumbra por capability e decisão documentada sobre processador legacy. `UnrelayedSequences` e `relayerStartLegacy` são candidatos a remoção, não a uma grande refatoração.
7. Ativar gate global obrigatório quando o inventário chegar a zero.

Cada PR deve ter escopo pequeno, testes de transição/erro/retry correspondentes e score `<10` nas duas métricas para qualquer função criada ou tocada. Mudança funcional e redução mecânica só devem coexistir quando o código antigo está sendo substituído pela nova abstração.

## Branches/PRs que podem ser minerados, sem merge direto

O inventário integral das 79 heads do origin, 35 PRs upstream e seus grupos de equivalência está em [`01_branch_archaeologist_inventory.md`](./01_branch_archaeologist_inventory.md). Estes são os deltas diretamente relevantes ao roadmap:

| Fonte | Conteúdo aproveitável | Regra |
|---|---|---|
| [PR #1530](https://github.com/cosmos/relayer/pull/1530), `eaaeefda` | Upgrade v9, DenomTraces, LatestHeight, interchaintest e testes de trusting-period/misbehaviour | v9 foi retraída; reaproveitar somente testes/refactors após rediff contra v11.2 |
| [PR #1147](https://github.com/cosmos/relayer/pull/1147), `a6fbf764` | Handshake flush, `processor/flush.go`, ICA close test | Draft/conflitante; extrair design/testes, não cherry-pick em bloco |
| [PR #876](https://github.com/cosmos/relayer/pull/876), `d428c6ff` | Recuperação de light client fork/halt | Comparar intenção com `MsgRecoverClient` v11 e reimplementar pela API atual |
| PR #1550, `4fcf84c4` | Bump v8.2→v8.7 | Obsoleto como alvo; evidencia dívida/security drift |
| PR #1549, `fab1c0ef` | SDK 0.50.11→0.50.13 | Obsoleto como alvo; pode fornecer correções de teste isoladas |
| branch `steve/wasm`, `ec23a923` | 08-wasm codecs/flags | Forte candidato a testes/adapters, após rediff v11 |
| branch `andrew/fix_ica_handshake` | Parser de múltiplas mensagens e retenção de eventos | Reaproveitar fixtures/semântica no parser action-indexed |
| branch `andrew/remove_legacy` | Consolidação/removal do processor legacy | Só após provar paridade event-based Classic+v2 |

## Riscos e decisões abertas

- **Contrato upstream em movimento:** a spec v2 ainda diz `EXPERIMENTAL`, enquanto ibc-go oferece releases estáveis. Toda capacidade deve ser pinada por tag e testada, não deduzida apenas da documentação.
- **Multi-payload divergente:** documentação e validação do `MsgSendPacket` v11.2.0 discordam. Manter feature flag off e abrir teste/upstream issue antes de prometer suporte.
- **Upgrade não é automático:** um connection/channel Classic existente não vira v2. Counterparty registration e config de client são um fluxo operacional separado.
- **Responsabilidade da chain:** GMP, callbacks, rate limit, light clients e prova EVM exigem módulos/keepers/contratos implantados. No tag v11.2.0, PFM também é responsabilidade da chain, mas seu módulo é Classic-only; o relayer deve cobri-lo em ICS-20 Classic e recusar uma promessa de PFM v2 até existir implementação upstream. O relayer fornece transporte, diagnósticos e testes de interoperabilidade.
- **Compatibilidade:** suportar ao mesmo tempo cadeias ibc-go v8 antigas e v11 aumenta bastante o custo. A matriz mínima suportada e o prazo de deprecation precisam de decisão do mantenedor no M0.
- **Provider Penumbra:** hoje replica uma superfície grande baseada no modelo Classic. A interface por capabilities deve impedir que v2 Cosmos force tipos ibc-go ou EVM sobre Penumbra.
- **Release recém-publicada:** v11.2.0 saiu durante a auditoria. Adotar o pin, mas exigir soak, leitura de advisories e regressão interchain antes de release do relayer.
- **Test truth:** `go build ./...` passa; `go test -mod=readonly ./...` tem uma falha live preexistente em `TestQueryBaseFee`. Nenhum marco deve aceitar teste público não determinístico como gate.

## Definition of done do programa

O projeto só pode declarar “IBC mais recente implementado” quando: (1) os pins e a data estiverem publicados; (2) a matriz M1+M2 estiver verde contra chains reais v11; (3) Classic tiver política explícita e regressão verde; (4) restart/reorg/timeout/retry forem idempotentes; (5) nenhuma função manuscrita tiver complexidade ciclomatica ou cognitiva `>=10`; (6) limites chain/app/light-client estiverem documentados; e (7) capacidades ainda divergentes ou experimentais, como multi-payload, estiverem claramente desabilitadas ou comprovadas por teste do tag exato.
