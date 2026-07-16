# M1.1a — contrato consolidado de implementação

Data: 2026-07-15  
Base preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Decisão de dependências

- Não importar `github.com/cosmos/ibc-go/v11/.../v2/types` no módulo raiz.
- Não alterar `go.mod` nem `go.sum` neste lote.
- Implementar o wire local de `Packet`, `Payload` e `Acknowledgement` a partir
  do `packet.proto` da tag `v11.2.0`, sem imports de SDK, Comet ou ibc-go/v11.
- Fixar compatibilidade por goldens produzidos pelo tipo oficial v11.2.0:
  - packet: 92 bytes,
    `0807120f30372d74656e6465726d696e742d301a0f30372d74656e6465726d696e742d31207b2a340a087472616e7366657212087472616e736665721a0769637332302d3122106170706c69636174696f6e2f6a736f6e2a03010203`;
  - acknowledgement: 4 bytes, `0a02aabb`.
- O limite do blob protobuf completo será operacional e explícito; o limite
  semântico total dos payloads continua exatamente `262144` bytes.

## API wire local

O pacote `relayer/protocol/v2` expõe:

```go
var (
    ErrEncodedValueTooLarge        error
    ErrInvalidHex                  error
    ErrInvalidProtobuf             error
    ErrInvalidPacketContract       error
    ErrInvalidAcknowledgement      error
)

func DecodePacketHex(string) (protocol.PacketEnvelope, error)
func DecodeAcknowledgementHex(string) (protocol.Acknowledgement, error)
```

O decoder valida o tamanho antes de alocar, preserva unknown fields sem panic,
valida as invariantes locais equivalentes ao tag e retorna erros compatíveis
com `errors.Is`. Nenhum erro/log inclui o valor hex completo.

## API de ingestão

```go
type IBCEventMetadata struct {
    ChainID string
    Height  uint64
    TxHash  string
}

type IBCEventBatch struct {
    Envelopes       []protocol.EventEnvelope
    ClassicMessages []IbcMessage
    V2Packets       []V2PacketEvent
    Issues          []IBCEventIssue
}

type V2PacketEvent struct {
    Event           protocol.EventEnvelope
    Observation     protocol.PacketObservation
    Acknowledgement *protocol.Acknowledgement
}

type IBCEventIssue struct {
    EventIndex uint32
    EventType  string
    Err        error
    Event      *protocol.EventEnvelope
}

func ParseIBCEventBatch(
    log *zap.Logger,
    events []abci.Event,
    metadata IBCEventMetadata,
) IBCEventBatch
```

`IbcMessagesFromEvents` permanece com a assinatura atual e retorna somente
`ClassicMessages`. `V2Packets` é sidecar sem consumidor neste lote.
`ParseIBCMessageFromEvent` recusa atributos exclusivos v2 para impedir uso
direto que projete packet v2 em `PacketInfo` Classic.

`IBCEventIssue.Event` preserva a evidência bruta rejeitada, incluindo ordem,
duplicatas e `Index`. Quando a classificação falha, esse diagnóstico usa
`ProtocolUnspecified` e não é inserido no slice validado `Envelopes`.

## Correlação

1. Copiar todos os `abci.EventAttribute` antes de qualquer projeção,
   preservando ordem, duplicatas e `Index`.
2. Primeira passagem: indexar `message` com `action` por `msg_index`; ignorar
   `message` do keeper contendo apenas `module`.
3. Segunda passagem: resolver pelo índice explícito. Para formatos legados
   sem `msg_index`, usar somente a última action sem índice que precede o
   evento. Nunca correlacionar para frente.
4. Ausência legítima deixa `Action.Present=false`; índice/action inválido ou
   conflitante vira issue tipada e não é resolvido por last-write-wins.

## Classificação e decode

- Aplicar somente aos cinco nomes compartilhados de packet.
- Qualquer mistura de atributo exclusivo Classic e exclusivo v2 é ambígua.
- Classic exige exatamente uma ocorrência de src/dst port+channel, sequence,
  timeout height e timeout timestamp.
- v2 exige exatamente uma ocorrência de src/dst client, sequence, timeout
  timestamp e encoded packet.
- `write_acknowledgement` v2 exige exatamente um encoded acknowledgement; os
  outros quatro eventos rejeitam esse atributo.
- Duplicata de singleton é malformada, ainda que os valores sejam iguais.
- Atributos desconhecidos permanecem preservados e são aceitos.
- Depois do protobuf decode, source client, destination client, sequence e
  timeout timestamp dos atributos devem coincidir exatamente com o packet.
- O protobuf validado é a fonte da `PacketObservation`; o raw envelope é a
  evidência. A ack tipada fica em `V2PacketEvent.Acknowledgement` e o único app
  acknowledgement é copiado para `PacketObservation.Acknowledgement`.

## Fronteira e gates

- Nenhuma referência nova do sidecar a processor, cache, state machine,
  builder ou broadcast.
- Classic válido mantém o mesmo resultado dos testes de caracterização.
- Testar os cinco eventos, multi-message, duplicates/Index, action conflitante,
  ambiguidade, hex/protobuf inválidos, oversize, mismatch e ack inválida.
- `go test`, race focado, build, lint e `go mod verify` passam.
- Toda função manual nova/tocada tem complexidade ciclomática e cognitiva no
  máximo 9. Arquivo gerado, se usado, mantém o marker oficial.

## Fora do lote

- Migração do root para SDK 0.54/Comet 0.39/ibc-go v11.
- Queries/proofs/builders v2, state machine, cache, retry e broadcast.
- Ligação do sidecar aos chain processors.
- Declaração de interoperabilidade IBC v2 completa.
