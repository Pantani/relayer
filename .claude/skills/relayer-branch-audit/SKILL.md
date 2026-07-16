---
name: relayer-branch-audit
description: "Audita todas as branches e PRs do relayer com Git/GitHub live, divergencia, commits unicos, duplicacao e relevancia para IBC/SDK. Use obrigatoriamente para inventario de branches, atualizacao, nova varredura ou analise parcial de refs; nao use para apagar ou mesclar branches."
---

# Auditoria de branches do relayer

## Procedimento

1. Registre `HEAD`, status, remotos e branch default sem trocar de checkout.
2. Consulte `git ls-remote --heads` e PRs abertos. Preserve os dois conjuntos separadamente.
3. Para cada branch, determine merge-base, ahead/behind, ultima data, autor, assunto e arquivos exclusivos em relacao ao SHA-base auditado fornecido pelo orquestrador; se ele nao existir, detecte a branch default. Use a mesma base em todas as metricas.
4. Classifique: `integrada`, `duplicada`, `ativa`, `historica`, `desconhecida`; uma classificacao nao autoriza exclusao.
5. Pesquise nomes e diffs por IBC, SDK, processor, event, client, connection, channel, fee, ICA, ICQ e middleware.
6. Grave `_workspace/01_branch_archaeologist_inventory.md` com comandos, snapshot temporal e tabela completa.

## Saida minima

- SHA-base e timestamp.
- Contagens separadas de branches e PRs.
- Tabela completa e grupos de equivalencia.
- Branches que merecem cherry-pick/rebase/revisao, sem executar essas operacoes.

## Falhas

Repita consulta remota uma vez. Se continuar falhando, use refs locais somente como snapshot e rotule a limitacao.
