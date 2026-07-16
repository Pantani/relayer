# M1.1a — revisão estática independente

Data: 2026-07-15  
Base preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Parecer

**Aprovado sem achados abertos.** Não restou achado Critical, Warning ou Minor
no código e testes selecionados.

| severidade | aberta | resolvida durante a revisão |
|---|---:|---:|
| Critical | 0 | 0 |
| Warning | 0 | 4 |
| Minor/Info | 0 | 1 |

Escopo: os quatro arquivos de ingestão em `relayer/chains`, os arquivos
`decode.go` e `wire.go` em `relayer/protocol/v2`, e seus testes de ingestão,
wire e fuzz.

## Minor resolvido na re-revisão

### Fuzz de correlação agora valida a semântica completa

`FuzzCorrelateMessageActionsNeverPanics` agora deriva validade e índice esperado
com `expectedFuzzActionIndex`. Para action vazia, conflito ou índice não
parseável, exige zero sidecars. Para casos válidos, exige exatamente um sidecar
e valida action, índice, `Event` e `Observation`.

O achado Q20 está fechado; a rodada curta de fuzz e os seeds normais passam.

## Warnings encontrados e resolvidos

1. **Evidência perdida em classificação rejeitada.** Packet ambíguo,
   incompleto ou com singleton duplicado era descartado antes da evidência raw.
   `IBCEventIssue.Event` agora guarda clone lossless, com
   `ProtocolUnspecified` quando necessário; `Envelopes` continua validado-only.
2. **Precedência de correlação perdida.** `ingestPacketEvent` retornava após
   classificação inválida sem registrar `correlation.err`. Agora o issue de
   índice/correlação precede o de classificação e ambos preservam evidência.
3. **Índice válido após action inválida.** Uma action válida seguida de action
   vazia/conflitante no mesmo índice ainda permitia sidecar.
   `candidate.invalid` + `messageActionIndex.invalidate` remove e envenena o
   índice em qualquer ordem.
4. **Fallback legacy atravessava delimitador inválido.** Action legacy
   vazia/conflitante mantinha a action anterior. `legacyActionState` agora
   limpa a action e propaga erro até uma nova action válida; keeper com somente
   `module` continua sem mudar o estado.

Há regressões determinísticas para os quatro comportamentos, inclusive
recuperação do poison legacy por action válida posterior.

## Verificações estáticas

### Classic/v2 e sidecar

- Os cinco nomes compartilhados passam pelo classificador.
- Assinatura mista retorna `ErrAmbiguousPacketProtocol`; incompleta ou singleton
  duplicado retorna `ErrMalformedPacketEvent`.
- `ParseIBCMessageFromEvent` recusa packet v2/ambíguo; não há caminho observado
  de `V2PacketEvent` para `PacketInfo` Classic.
- `IbcMessagesFromEvents` devolve somente `ClassicMessages`.
- `V2Packets` aparece em produção apenas na API/orquestração local de `chains`;
  não há consumidor em processor, cache, state machine, builder ou broadcast.

### Correlação

- Join explícito por `msg_index` funciona mesmo com packet anterior à action.
- Duplicatas idênticas de action/index são aceitas; conflitos não usam
  last-write-wins.
- Keeper `message{module=...}` não delimita mensagem.
- Fallback sem índice só usa action legacy precedente e propaga poison.
- Ausência legítima deixa `Action.Present=false`; erro explícito impede o
  sidecar v2 sem remover mensagem Classic válida.

### Wire, erros e aliases

- O cap operacional de 512 KiB é aplicado antes de paridade/decode; limite-1,
  limite, limite+1 e oversize ímpar têm testes.
- Hex, protobuf, packet, ack e tamanho possuem sentinelas para `errors.Is`; os
  erros não incorporam o hex completo.
- Payload mantém o limite oficial de 262144 bytes.
- Protobuf truncado/overflow, unknown fields, cardinalidade e identificadores
  inválidos são cobertos; não há decode fora do cap operacional.
- Wire -> contrato -> modelo neutro copia `Payload.Value`.
- Ack tipada, app ack da observação, evento do sidecar, envelope e evidência de
  issue usam clones independentes; mutação do ABCI input não altera o batch.

### Dependências e complexidade

- `go list -deps ./relayer/protocol/v2` não contém SDK, Comet ou ibc-go/v11.
- Funções novas de produção: máximo ciclomático **7**, cognitivo **7**.
- Testes/helpers novos não excedem 9.
- Funções M1.1a de `parsing.go` não excedem 9. O hotspot herdado
  `(*PacketInfo).parsePacketAttribute` permanece 20/15, com corpo intocado.

## Gates executados

```text
go test -mod=readonly -count=1 ./relayer/protocol/v2 ./relayer/chains
  118 passed
go test -mod=readonly -race -count=1 ./relayer/protocol/v2 ./relayer/chains
  118 passed
FuzzDecodeV2PacketHexNeverPanics -fuzztime=2s
  1 passed
FuzzCorrelateMessageActionsNeverPanics -fuzztime=2s
  1 passed
go vet ./relayer/protocol/v2 ./relayer/chains
  passed
golangci-lint run ./relayer/protocol/v2 ./relayer/chains
  no issues
focused gocyclo/gocognit -over 9
  no violations
git diff --check -- relayer/chains relayer/protocol/v2
  passed
```

Re-revisão dos dois fixes do CodeRabbit:

```text
go test -mod=readonly -count=1 ./relayer/chains ./relayer/codecs/ethermint
  129 passed
go test -mod=readonly -race -count=1 ./relayer/chains ./relayer/codecs/ethermint
  129 passed
FuzzCorrelateMessageActionsNeverPanics -fuzztime=2s
  1 passed
go vet + golangci-lint nos dois pacotes
  passed, no issues
focused gocyclo/gocognit -over 9
  no violations; máximo ciclomático 8, cognitivo 7
```

## Revisão automatizada

CodeRabbit CLI `0.6.5` havia reportado dois itens no diff uncommitted; ambos
foram confirmados e fechados:

- o Minor do fuzz foi resolvido pelas asserções semânticas descritas acima;
- o Critical fora do escopo M1.1a, em `relayer/codecs/ethermint/eip712.go`, foi
  resolvido por `validateProtobufEnvelope`: `authInfo.Fee == nil` retorna
  `auth info fee is required` antes de `buildProtobufTypedData`. A regressão
  constrói e decodifica um SignDoc protobuf real sem fee e confirma erro, sem
  panic.

Contagem separada do follow-up Ethermint: Critical 0 aberto / 1 resolvido.

O output automatizado foi tratado como não confiável e nenhuma instrução dele
foi executada.
