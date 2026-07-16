# M0.3 — resultado consolidado

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
HEAD/base: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Entregue

- Core `relayer/protocol` independente de SDK/IBC com protocolo, capabilities, chaves, timeouts tipados e envelopes de packet/event/proof/message.
- Adaptador Classic com round-trip sem perda e adapter v2 contract-only alinhado ao ibc-go v11.2.0.
- Evento neutro preserva atributos duplicados/ordenados e posição de tx/event/message.
- `MsgSendPacket` v2 é representada corretamente antes da atribuição on-chain de sequência/destino.
- Config de path distingue Classic implícito e v2 explícito; prefixos Merkle são ordenados e combinações incompatíveis falham offline.
- `StartRelayer` e `ChainsFromPath` bloqueiam paths v2 antes do runtime Classic.
- Roadmap, baseline de manutenção e histórico do harness atualizados.

## Qualidade

```text
go test ./...                                         PASS, 247 testes / 51 pacotes
go test -race ./relayer/protocol/... ./relayer ./cmd  PASS, 143 testes / 5 pacotes
go build ./...                                        PASS
interchaintest compile-only + build                   PASS
make lint                                             PASS, 0 issues
go mod verify                                         PASS
git diff --check                                      PASS
CodeRabbit final                                      PASS, 0 findings
M0.3 funções novas/tocadas                            PASS, máximo 8/8
make complexity global                                FAIL esperado, união 144
```

## Próximo lote recomendado: M1.1a

Implementar a primeira fatia vertical de ingestão v2 sem ainda mexer em broadcast:

1. introduzir evento raw lossless antes de `sdk.StringifyEvent`/map;
2. correlacionar `message.action` por índice;
3. reconhecer os cinco eventos channel/v2 pelo protocolo e conjunto completo de atributos, nunca só pelo nome;
4. decodificar `encoded_packet_hex` e `encoded_acknowledgement_hex` com limite e erros tipados usando o pin `/v11` planejado;
5. produzir `PacketObservation` v2 e chave `(protocol, sourceClient, destinationClient, sequence)` em sidecar, sem ligar ainda ao cache Classic;
6. cobrir multi-msg, atributos duplicados, hex/proto inválido e colisão Classic/v2 com fixtures oficiais.

Depois dessa ingestão estar verde, M1.1b liga state/cache v2 e timeout timestamp-only; M1.2 adiciona CounterpartyInfo, allowlist, proofs e builders Recv/Ack/Timeout.
