## Harness: manutencao do Go relayer

**Objetivo:** manter o relayer moderno, com complexidade estritamente abaixo de 10 e suporte rastreavel ao IBC/SDK atuais, incluindo IBC v2.

**Trigger:** em trabalho de branches, complexidade, build de qualidade, upgrade Cosmos SDK/IBC Go, IBC v2, roadmap, reexecucao ou melhoria de resultados do relayer, use a skill `relayer-maintenance-orchestrator`. Perguntas conceituais simples podem ser respondidas diretamente.

**Historico de mudancas:**

| Data | Mudanca | Alvo | Motivo |
|---|---|---|---|
| 2026-07-15 | Criacao inicial do harness | `.claude/agents`, `.claude/skills` e `CLAUDE.md` | Manter o projeto com analise paralela, gates de complexidade e roadmap IBC v2 |
| 2026-07-15 | Contratos de QA/retry/base alinhados | Skills de branch, complexidade, QA e orquestrador | Corrigir inconsistencias encontradas na revisao do M0.1 |
| 2026-07-15 | Fan-out/fan-in incremental M0.2 executado | FeeGrant, CLI/config, Ethermint e QA | Reduzir hotspots isolados preservando contratos Classic antes do modelo IBC v2 |
| 2026-07-15 | Fundação protocol-neutral M0.3 executada | `relayer/protocol`, config de path e guardas Classic/v2 | Fixar contratos v11.2.0 e impedir dispatch v2 acidental antes das fatias verticais M1 |
| 2026-07-15 | Contrato de integração M1.1b-d modernizado | `interchaintest/v11`, módulo isolado, workspace e CI | Remover a fronteira Store v1 sem fork e tornar incompatibilidades SDK/IBC bloqueantes antes dos E2E |
