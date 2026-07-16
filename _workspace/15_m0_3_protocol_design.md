# M0.3 — contrato protocol-neutral

Data da verificação: 2026-07-15  
Referência upstream fixada: `github.com/cosmos/ibc-go/v11@v11.2.0`, tag no commit `cfc072e53eee42b2ab804cd4344ba610016f793c`

## Decisão

O menor fundamento seguro para M0.3 é um pacote interno que só importa a biblioteca padrão e representa o envelope comum, mais dois adaptadores irmãos. Ele **não** substitui `provider.PacketInfo` nem entra no processor neste lote; isso preserva o runtime Classic e impede que um DTO local seja confundido com suporte v2 operacional.

IBC Classic e IBC v2 são valores discriminantes, não modos inferidos a partir de connection/channel. O zero value de `Protocol` é inválido no core. Somente a camada de migração de config pode converter `protocol: ""` em `classic` para manter YAML legado compatível.

## Evidência normativa v11.2.0

- [`channel/v2/types/packet.go`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/v2/types/packet.go): packet v2 tem sequence, source/destination client, timeout timestamp e payloads; o tag aceita exatamente um payload, no máximo 256 KiB, e exige version/encoding/value.
- [`channel/v2/types/msgs.go`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/v2/types/msgs.go): Send, Recv, Acknowledgement e Timeout; Recv usa proof commitment, Ack usa proof acked e Timeout usa proof unreceived.
- [`channel/v2/types/events.go`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/v2/types/events.go): nomes e atributos v2 são `send_packet`, `recv_packet`, `write_acknowledgement`, `acknowledge_packet`, `timeout_packet`, clients, sequence, timestamp e packet/ack codificados em hex.
- [`client/v2/types/counterparty.go`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/02-client/v2/types/counterparty.go): counterparty é client ID + Merkle prefix; não existe connection/channel handshake v2.
- [`client/v2/types/config.go`](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/02-client/v2/types/config.go): allowlist vazia é permissionless; o limite upstream é 20 relayers.

## Arquivos propostos

```text
relayer/protocol/
  protocol.go             Protocol, Capabilities, Height, Timeout
  packet.go               Endpoint, PacketID, Payload, PacketEnvelope
  event.go                EventAttribute, MessageAction, EventEnvelope
  proof.go                ProofKind, ProofEnvelope
  message.go              MessageKind, Acknowledgement, MessageEnvelope
  *_test.go                validação estrutural table-driven
  classic/
    adapter.go             provider.PacketInfo -> protocol.PacketEnvelope
    adapter_test.go        equivalência exata com PacketInfo.Packet()
  v2/
    contract.go            DTO v11.2.0 local, sem proto/marshal
    adapter.go             ContractPacket -> protocol.PacketEnvelope
    adapter_test.go        golden de campos e rejeições do tag
```

`relayer/protocol` não pode importar Cosmos SDK, ibc-go, provider ou processor. `classic` pode importar `provider` e ibc-go/v8. `v2` não importa ibc-go/v11 enquanto o módulo principal continuar pinado em v8; seu `ContractPacket` é apenas uma sentinela compilável da forma dos campos, nunca um wire type.

## API exata proposta

```go
package protocol

type Protocol string

const (
    ProtocolClassic Protocol = "classic"
    ProtocolV2      Protocol = "v2"
)

func ParseProtocol(value string) (Protocol, error)
func (p Protocol) Validate() error

type Capabilities struct {
    ClientRouting            bool
    ConnectionHandshake      bool
    ChannelHandshake         bool
    OrderedDelivery          bool
    TimeoutHeight            bool
    TimeoutTimestamp         bool
    AsyncAcknowledgement     bool
    PerClientRelayerAllowlist bool
    MaxPayloads              int
}

func CapabilitiesFor(p Protocol) (Capabilities, error)

type Height struct {
    RevisionNumber uint64
    RevisionHeight uint64
}

func (h Height) IsZero() bool
func (h Height) Validate() error

type TimestampUnit string

const (
    TimestampNanoseconds TimestampUnit = "nanoseconds"
    TimestampSeconds     TimestampUnit = "seconds"
)

type Timeout struct {
    Height        Height
    Timestamp     uint64
    TimestampUnit TimestampUnit
}

type Endpoint struct {
    ClientID  string
    PortID    string
    ChannelID string
}

type PacketID struct {
    Protocol    Protocol
    Source      Endpoint
    Destination Endpoint
    Sequence    uint64
}

func (id PacketID) Validate() error
func (id PacketID) Counterparty() PacketID

type Payload struct {
    SourcePort      string
    DestinationPort string
    Version         string
    Encoding        string
    Value           []byte
}

type PacketEnvelope struct {
    ID       PacketID
    Timeout  Timeout
    Payloads []Payload
}

func (p PacketEnvelope) Validate() error

type EventAttribute struct {
    Key   string
    Value string
    Index bool
}

type MessageAction struct {
    Present bool
    Index   uint32 // zero-based message ordinal inside the transaction
    Type    string // the preserved message.action value
}

type EventEnvelope struct {
    Protocol   Protocol
    Type       string
    Height     uint64
    Action     MessageAction
    Attributes []EventAttribute // ordered; duplicate keys are intentional
}

func (e EventEnvelope) Validate() error
func (e EventEnvelope) RequireAction() error
func (e EventEnvelope) AttributeValues(key string) []string

type ProofKind string

const (
    ProofPacketCommitment  ProofKind = "packet_commitment"
    ProofAcknowledgement   ProofKind = "acknowledgement"
    ProofReceiptAbsence    ProofKind = "receipt_absence"
    ProofNextSequenceRecv  ProofKind = "next_sequence_receive" // Classic ordered only
)

type ProofEnvelope struct {
    Protocol Protocol
    Kind     ProofKind
    Height   Height
    Data     []byte
}

func (p ProofEnvelope) Validate() error

type MessageKind string

const (
    MessageSendPacket      MessageKind = "send_packet"
    MessageRecvPacket      MessageKind = "recv_packet"
    MessageAcknowledgement MessageKind = "acknowledgement"
    MessageTimeout         MessageKind = "timeout"
)

type Acknowledgement struct {
    Protocol            Protocol
    Value               []byte   // Classic opaque acknowledgement
    AppAcknowledgements [][]byte // v2 ordered application acknowledgements
}

func (a Acknowledgement) Validate() error

type MessageEnvelope struct {
    Protocol      Protocol
    Kind          MessageKind
    Packet        PacketEnvelope
    Proof         *ProofEnvelope
    Acknowledgement *Acknowledgement
    Signer        string
}

func (m MessageEnvelope) Validate() error
```

`TimestampUnit` é obrigatório porque Classic serializa timeout timestamp em nanossegundos e channel/v2 v11.2.0 o define em segundos. Um único `uint64` sem unidade criaria uma conversão silenciosa perigosa.

## Capabilities fixadas neste tag

| capability | Classic | v2 v11.2.0 |
|---|---:|---:|
| client routing | não | sim |
| connection/channel handshake | sim | não |
| ordered delivery | sim | não |
| timeout height | sim | não |
| timeout timestamp | sim | sim |
| async acknowledgement | sim | sim |
| allowlist por client | não | sim |
| `MaxPayloads` | 1 | 1 |

`MaxPayloads=1` é deliberado. A especificação descreve lista/multi-payload, mas `ValidateBasic` no tag v11.2.0 rejeita qualquer comprimento diferente de 1. M0.3 não deve anunciar a capability futura.

## Invariantes de validação

| tipo | Classic | v2 |
|---|---|---|
| `PacketID` | sequence > 0; port/channel dos dois lados presentes; client IDs vazios | sequence > 0; client IDs presentes; port/channel do ID vazios, pois ports pertencem a cada payload |
| `Timeout` | pelo menos height ou timestamp; timestamp não zero usa nanossegundos | height zero; timestamp > 0 e em segundos |
| `Payloads` | exatamente 1; source/destination port iguais aos do ID; version/encoding podem ficar vazios; total <= 256 KiB | exatamente 1; ports, version, encoding e value não vazios; total <= 256 KiB |
| `EventEnvelope` | protocol/type válidos; action pode estar ausente no estágio raw | mesmo; `RequireAction` é gate obrigatório antes de correlação/processamento |
| `ProofEnvelope` | data e height não vazios; commitment/ack/receipt-absence/next-sequence permitidos | data e height não vazios; somente commitment/ack/receipt-absence |
| `Acknowledgement` | `Value` não vazio e lista v2 vazia | `Value` vazio e exatamente um app acknowledgement não vazio neste tag |
| `MessageEnvelope` | protocol deve coincidir com packet/proof/ack; signer não vazio | idem |

Validação de formato host (`ClientIdentifierValidator`, `PortIdentifierValidator`, channel ID), Bech32, protobuf e prova criptográfica fica no adaptador upstream. O core neutro verifica apenas forma, exclusividade e unidades; não deve copiar regex/crypto do SDK.

Para manter ciclomatica e cognitiva abaixo de 10, `Validate` só faz validações comuns e despacha para helpers pequenos: `validateClassicPacket`, `validateV2Packet`, `validateSendMessage`, `validateRecvMessage`, `validateAckMessage` e `validateTimeoutMessage`. Não usar um único switch aninhado sobre protocolo + kind + proof.

## Contrato dos adaptadores em M0.3

### Classic

`classic.FromPacketInfo(provider.PacketInfo) (protocol.PacketEnvelope, error)` copia todos os bytes defensivamente, converte `clienttypes.Height` para `protocol.Height`, marca nanossegundos e cria um payload opaco. O teste compara todos os campos com `PacketInfo.Packet()` e prova que mutar a origem depois não altera o envelope. Este adaptador não é conectado ao processor ainda.

### v2 contract-only

```go
package v2

type ContractPayload struct {
    SourcePort, DestinationPort, Version, Encoding string
    Value []byte
}

type ContractPacket struct {
    Sequence uint64
    SourceClient, DestinationClient string
    TimeoutTimestamp uint64
    Payloads []ContractPayload
}

func FromContractPacket(packet ContractPacket) (protocol.PacketEnvelope, error)
```

Os nomes/campos espelham o tag v11.2.0. Não adicionar marshal/unmarshal, event parsing, queries ou builders: isso pareceria suporte wire sem a dependência real. Após o upgrade para `/v11`, testes de paridade substituem o DTO por conversões de/para `channelv2types.Packet`.

## Testes mínimos deste lote

1. tabela `Protocol`/capabilities, incluindo zero e desconhecido;
2. PacketID rejeita mistura de client routing com channel routing;
3. diferença de unidade e timeout Classic/v2;
4. v2 aceita um payload e rejeita 0, 2, vazio e >256 KiB;
5. `EventAttribute` conserva ordem e duplicatas; action index 0 continua distinguível por `Present`;
6. proof kinds por protocolo;
7. message matrix Send/Recv/Ack/Timeout, incluindo proof/ack inesperado;
8. equivalência Classic e cópia defensiva;
9. paridade de campos do `ContractPacket` v2 e nenhuma importação `/v11` no módulo.

## Adiado obrigatoriamente para M1

- importar ibc-go/v11, protobuf encode/decode e `ValidateBasic` real;
- correlacionar eventos por `message.action` e decodificar `encoded_packet_hex`/`encoded_acknowledgement_hex`;
- trocar `RelayerEvent.Attributes map[string]string` ou os caches do processor;
- chaves/cache/state machine v2 e coexistência runtime;
- queries/proofs Merkle v2, CounterpartyInfo, register/update config;
- builders/broadcast de Recv/Ack/Timeout e interpretação NOOP/SUCCESS/FAILURE;
- multi-payload, que permanece capability desativada até evidência upstream executável.

Assim M0.3 entrega uma fronteira testável e honesta sem alterar comportamento Classic nem declarar interoperabilidade v2.
