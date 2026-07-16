# M0.3 — snapshot de entrada

Data: 2026-07-15 (America/Sao_Paulo)  
Branch: `Pantani/cx/m0-baseline`  
HEAD/base preservado: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Objetivo do lote

Introduzir a fundação protocol-neutral, com configuração explicitamente versionada e contratos separados para IBC Classic e IBC v2. Este lote não troca os pins atuais nem anuncia relay v2 operacional; ele prepara as fronteiras para que o runtime v2 seja implementado verticalmente no M1.

## Estado herdado

- M0.1 e M0.2 permanecem no working tree, sem commit ou push.
- O último rebaseline independente registrou 86 violações ciclomáticas, 139 cognitivas e união de 145 funções; máximos 48/99.
- Build, testes unitários, compilação do módulo `interchaintest`, lint e checks de whitespace passaram no fan-in do M0.2.
- `make complexity` continua falhando corretamente até a dívida global chegar a zero.

## Fronteiras Classic encontradas

| fronteira | contrato atual | consequência |
|---|---|---|
| config | `Path` contém `src`, `dst` e filtro de canais; `PathEnd` contém chain/client/connection | v2 precisa selecionar protocolo e rejeitar connection/channel filter |
| provider | `PacketInfo` importa `clienttypes.Height` e converte diretamente para packet channel/v1 | o modelo interno está acoplado ao Classic |
| eventos | `RelayerEvent.Attributes` é `map[string]string` | ordem, duplicatas e `message.action` não são preservadas |
| processor | caches são indexados por `ChannelKey` e sequência | v2 precisa de chave por client pair, protocolo e sequência |
| proof/msg | proofs e builders usam channel/v1 e timeout height | v2 usa client pair, timestamp e mensagens channel/v2 |

## Contrato de execução

- Fixar o desenho com três análises independentes antes de editar as fronteiras.
- Manter o runtime Classic intacto neste lote.
- Representar v2 sem importar `ibc-go/v11` enquanto o módulo estiver pinado em v8.
- Toda função criada ou tocada deve ter complexidade ciclomática e cognitiva máxima 9.
- Exigir testes de validação, round-trip e adaptação; depois executar unit, build, lint, complexidade focada e rebaseline global.
