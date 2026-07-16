---
name: relayer-complexity
description: "Mede e reduz complexidade ciclomatica e cognitiva do Go relayer com maximo estrito 10/10. Use obrigatoriamente em auditoria, refatoracao, regressao, reexecucao, atualizacao ou melhoria de complexidade; nao use para codigo gerado com marcador canonico."
---

# Complexidade do relayer

## Contrato

- Use apenas `gocyclo@v0.6.0` e `gocognit@v1.2.1` com `-over 10`; score 10 e aprovado, score 11 e violacao.
- Audite todo Go manuscrito, incluindo testes. Exclua somente cabecalho com `// Code generated ... DO NOT EDIT.`.
- A condicao terminal e ciclo/cognitiva/uniao `0/0/0`, maximos `<=10/<=10` e `make complexity` verde.
- Nao use `nolint`, suppression, allowlist, baseline permissivo, threshold maior que 10 ou exclusao por nome.

## Procedimento

1. Fixe SHA, status, PRs e ownership vivos.
2. Execute o gate do repo `<repo>/scripts/check-complexity.sh` e gere `_workspace/complexity/inventory.md` com `<repo>/.claude/skills/relayer-complexity-campaign/scripts/inventory.sh`.
3. Compare por `arquivo:funcao`, nunca apenas por linha.
4. Antes de editar, registre scores do arquivo e conclua a caracterizacao no original.
5. Prefira extracoes por responsabilidade e retornos antecipados; preserve comportamento observavel.
6. Meça todos os helpers novos e zere todas as violacoes dos arquivos de producao tocados.
7. Rode testes focais com `-race`, pacote sem race, scores locais e inventario global.
8. Aceite o lote somente com reducao global estrita e nenhum contrato alterado.

## Saida minima

- Funcoes/scores antes e depois.
- Inventario global antes e depois.
- Testes/gates e exclusoes.
- Riscos, dependencias e bloqueios.

## Proibicoes semanticas

Nao altere APIs, mensagens de erro, logs/campos/niveis, metricas, ordem de efeitos, sorting observavel, retries, cancelamento, timeouts, concorrencia ou efeitos parciais. IBC v2, upgrades, Bech32 e backpressure ficam fora da campanha funcional.
