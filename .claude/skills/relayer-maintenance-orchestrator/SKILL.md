---
name: relayer-maintenance-orchestrator
description: "Orquestra a manutencao do Go relayer por equipe: auditoria de todas as branches/PRs, complexidade ciclomatica e cognitiva abaixo de 10, upgrades Cosmos SDK, IBC Go e IBC v2, build, QA e roadmap. Use obrigatoriamente para iniciar, atualizar, corrigir, complementar, reexecutar, rodar apenas uma parte ou melhorar resultados anteriores do programa de manutencao do relayer."
---

# Orquestrador de manutencao do relayer

## Modo de execucao

Equipe em fan-out/fan-in com QA incremental. Em ambientes com `TeamCreate`, use equipe nativa; em ambientes Codex, use agentes paralelos equivalentes e mensagens diretas. Todos os agentes usam `model: "opus"` quando o runtime expuser esse parametro.

## Equipe e artefatos

| Agente | Skill | Artefato |
|---|---|---|
| branch-archaeologist | relayer-branch-audit | `_workspace/01_branch_archaeologist_inventory.md` |
| complexity-engineer | relayer-complexity | `_workspace/02_complexity_engineer_baseline.md` |
| ibc-gap-analyst | relayer-ibc-roadmap | `_workspace/03_ibc_gap_analyst_roadmap.md` |
| integration-qa | relayer-qa | `_workspace/04_integration_qa_report.md` |

## Phase 0: contexto e reexecucao

1. Verifique `_workspace/` e o pedido atual.
2. Sem artefatos: execucao inicial.
3. Com artefatos e pedido parcial: reexecute apenas os agentes afetados e preserve os demais.
4. Com nova base/entrada: mova o workspace anterior para `_workspace_YYYYMMDD_HHMMSS/` e crie um novo.
5. Nunca mova resultados no meio de uma execucao ativa.

## Phase 1: snapshot seguro

Registre HEAD, status, remotos, branch default, `go.mod`, ferramentas, workflows e data em `_workspace/00_input/snapshot.md`. Nao troque de branch em checkout destacado.

## Phase 2: fan-out paralelo

Inicie simultaneamente `branch-archaeologist`, `complexity-engineer` e `ibc-gap-analyst`. Cada agente le sua definicao e skill antes de agir, grava o artefato contratado e comunica descobertas que mudem o trabalho dos outros.

## Phase 3: fan-in e plano de lotes

O lider cruza os tres artefatos, resolve divergencias por evidencia e cria lotes ordenados:

1. gate de complexidade e baseline;
2. refatoracoes comportamentais;
3. atualizacao de fundacao SDK/IBC;
4. suporte IBC v2 por fatias verticais;
5. interoperabilidade e hardening operacional.

## Phase 4: implementacao com QA incremental

Para cada lote: implementar, formatar, rodar testes focados, medir as duas metricas e chamar `integration-qa`. Uma falha retorna ao produtor uma vez antes de ser marcada como bloqueio explicito.

## Phase 5: validacao final

Rode build, testes unitarios, complexidade e verificacoes de docs. Testes Docker/interchain ficam separados quando a infraestrutura nao estiver disponivel. Gere `docs/ibc-v2-roadmap.md` e mantenha `_workspace/` para auditoria.

## Fluxo de dados

`snapshot -> tres analises paralelas -> integracao -> lotes -> QA por lote -> build final -> roadmap`

## Erros e conflito

- Falha individual: uma repeticao; depois prossiga com lacuna declarada.
- Maioria falha: interrompa implementacao e reporte o bloqueio.
- Fontes conflitantes: preserve ambas, com versao/data, e valide na tag/codigo.
- Ref instavel: fixe SHA novamente antes de integrar resultados.

## Testes de trigger

Leia `references/trigger-tests.md` ao alterar a description ou adicionar um agente.

## Cenarios de teste

### Normal

Snapshot limpo, tres agentes produzem artefatos, QA valida, os gates e testes passam e o roadmap final e gerado.

### Falha

A consulta remota de branches falha duas vezes; o inventario local e preservado como possivelmente desatualizado, complexidade/IBC continuam e a QA marca a contagem remota como nao verificada.
