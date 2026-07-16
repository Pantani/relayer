# M0.3 — contrato consolidado para implementação

Data: 2026-07-15  
Fonte das decisões: `_workspace/15_m0_3_protocol_design.md`, `16_m0_3_config_design.md` e `17_m0_3_boundary_design.md`.

## Decisões de fan-in

1. `relayer/protocol` será independente de Cosmos SDK, ibc-go, provider e processor.
2. O zero value de `protocol.Protocol` é inválido no core; somente `Path.EffectiveProtocol` converte campo ausente em Classic para compatibilidade.
3. Packet, proof, acknowledgement e message carregam protocolo explícito. Mistura de campos Classic/v2 falha; não existe fallback.
4. O timeout carrega unidade explícita: Classic usa nanossegundos e v2 usa segundos.
5. `PacketObservation` separa altura/evento/order/ack do packet wire e permite round-trip exato de `provider.PacketInfo`.
6. Eventos conservam `[]EventAttribute` ordenado, chaves duplicadas e `message.action` com bit de presença para distinguir índice zero de ausência.
7. O adaptador v2 é um DTO contract-only alinhado ao tag `v11.2.0`; não faz protobuf, queries, proofs ou broadcast.
8. O tag alvo aceita exatamente um payload. Multi-payload continua desabilitado e adiado.
9. `Path.Protocol` usa `protocol,omitempty`; ausente continua Classic sem reescrever YAML. `PathEnd.MerklePrefix []string` usa `merkle-prefix,omitempty`.
10. Validação estrutural v2 exige chain/client/prefix nos dois lados e proíbe connection e channel filter. Prontidão bilateral/allowlist fica para M1.
11. `Config.AddPath` compara protocolos efetivos e não converte Classic/v2 implicitamente.
12. `StartRelayer` e `ChainsFromPath` rejeitam v2 antes de `SetPath`, query, cache ou broadcast. Parser, processor, proof selection e broadcast existentes não serão alterados.

## Owners

| owner | arquivos |
|---|---|
| modelo | `relayer/protocol/**`, exceto `classic/**` |
| config | `relayer/path.go`, `relayer/pathEnd.go`, `relayer/ics24.go`, `cmd/config.go` e testes focados |
| boundary | `relayer/protocol/classic/**`, guarda central em `relayer/strategies.go` e testes |

## Aceitação

- round-trip Classic de packet observation e proof, com cópias defensivas;
- contrato v2 literal de packet/payload/event/message e validação de um payload;
- atributos ordenados/duplicados e chaves isoladas por protocolo;
- round-trip YAML/JSON legado e explícito; combinações inválidas falham antes de RPC;
- path v2 não alcança runtime Classic;
- nenhuma importação `/v11` em `go.mod`, `go.sum` ou core novo;
- todas as funções novas/tocadas com complexidade máxima 9 nas duas métricas;
- testes focados, suíte raiz, build, lint e `git diff --check` passam; gate global continua reportado separadamente.
