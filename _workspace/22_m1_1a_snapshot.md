# M1.1a — snapshot de entrada

Data: 2026-07-15 (America/Sao_Paulo)  
Branch: `Pantani/cx/m0-baseline`  
HEAD/base preservado: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Objetivo

Implementar a primeira fatia vertical de ingestão IBC v2 sem conectar packets v2 ao cache, state machine ou broadcast Classic:

1. preservar eventos ABCI e atributos em ordem, incluindo duplicatas;
2. correlacionar cada evento ao `message.action` e índice da mensagem;
3. discriminar eventos packet Classic/v2 pelo contrato completo, nunca apenas por `event.Type`;
4. decodificar `encoded_packet_hex` e `encoded_acknowledgement_hex` com limites e erros tipados;
5. produzir `protocol.PacketObservation` v2 em sidecar;
6. manter o runtime Classic inalterado e v2 bloqueado pelas guardas M0.3.

## Estado herdado

- M0.1–M0.3 permanecem no working tree, sem commit ou push.
- `go.mod` ainda usa Cosmos SDK `v0.50.11`, CometBFT `v0.38.12` e `ibc-go/v8 v8.2.0`.
- O tag alvo `ibc-go/v11.2.0` requer SDK `v0.54.0`, CometBFT `v0.39.0`, `cosmossdk.io/log/v2` e store/v2; importar `/v11` diretamente pode elevar dependências compartilhadas por MVS e precisa ser provado antes de editar pins.
- O parser atual chama `sdk.StringifyEvent` e despacha apenas por nome; Classic e v2 compartilham os cinco event types de packet.
- O modelo M0.3 já possui envelopes lossless, PacketObservation, chaves protocol-qualified e DTOs v2 contract-only.
- Baseline global M0.3: 86 violações ciclomáticas, 138 cognitivas, união 144, máximos 48/99.

## Restrições

- Nenhuma seleção v2 apenas pelo nome do evento.
- Nenhuma projeção para `map[string]string` antes da correlação action-indexed.
- Nenhuma branch v2 em `processor.ChannelKey`, caches ou broadcast neste lote.
- Nenhum bump de dependência sem prova de build/teste proporcional e contrato de migração.
- Toda função criada ou tocada deve ter complexidade ciclomática e cognitiva máxima 9.
