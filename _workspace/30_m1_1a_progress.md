# M1.1a — progresso e handoff

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
Base preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Estado Git: mudanças não commitadas e não publicadas, por decisão do usuário.

## Entregue

1. `relayer/protocol/v2` agora decodifica o wire Packet/Payload/Acknowledgement
   exato de v11.2.0 sem importar o grafo `/v11`.
2. `ParseIBCEventBatch` cria envelopes lossless, mensagens Classic e sidecars
   v2 separados.
3. Correlação usa `msg_index` em duas passagens e fallback legado apenas para a
   action precedente; actions inválidas envenenam índice/estado até recuperação.
4. Os cinco event types colidentes são classificados pelo contrato completo;
   nenhum packet v2 pode virar `PacketInfo` Classic.
5. Packet/ack são limitados, semanticamente validados e comparados com os
   atributos redundantes antes de publicar uma observação.
6. Eventos rejeitados preservam evidência raw defensiva; issues são tipadas.
7. V2 permanece sidecar-only, sem cache, state machine, builder ou broadcast.
8. Goldens, matriz negativa, regressões Classic, race e fuzz estão cobertos.
9. O review também fechou o panic Ethermint para protobuf `AuthInfo` sem Fee.

## Decisão de arquitetura

A prova isolada mostrou que importar apenas `channel/v2/types` seleciona SDK
0.54/Comet 0.39 por MVS e quebra o relayer atual em `x/crisis`, `sr25519` e
store v1/v2. Portanto M1.1a entrega compatibilidade de wire/observação, não
compatibilidade de compilação ou interoperabilidade completa v11.

## Próximo lote — M1.1b

Objetivo: migrar o grafo raiz de forma coerente para SDK 0.54, Comet 0.39 e
ibc-go/v11, preservando Classic e substituindo o wire transitório somente após
build/testes verdes.

Fronteiras mínimas:

1. remover/substituir `x/crisis` e `crypto/sr25519` obsoletos;
2. alinhar store v1/v2, `cosmossdk.io/x/*`, log/v2 e codecs;
3. migrar imports Classic `/v8` para a linha v11 sem alterar semântica;
4. revalidar Cosmos/Penumbra, CLI, interchaintest e release;
5. executar regressão Classic completa e manter todo código tocado em máximo
   9/9;
6. somente então iniciar CounterpartyInfo, queries/proofs e builders v2.

## Artefatos

- `_workspace/22_m1_1a_snapshot.md`
- `_workspace/23_m1_1a_dependency.md`
- `_workspace/24_m1_1a_ingestion.md`
- `_workspace/25_m1_1a_qa.md`
- `_workspace/26_m1_1a_contract.md`
- `_workspace/27_m1_1a_static_review.md`
- `_workspace/28_m1_1a_code_review.md`
- `_workspace/29_m1_1a_qa.md`
