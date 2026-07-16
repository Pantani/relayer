---
name: relayer-complexity-characterization
description: "Congela comportamento original antes de qualquer refatoracao de complexidade no Go relayer. Use obrigatoriamente para iniciar, continuar, reexecutar, atualizar ou corrigir testes de caracterizacao de uma subwave."
---

# Caracterizacao de complexidade do relayer

## Procedimento

1. Confirme SHA original, ownership e funcoes violadoras.
2. Liste entradas/saidas, API, erros exatos, logs, metricas, ordem e dependencias globais.
3. Para estado/concorrencia, liste identidade e conteudo de caches, deduplicacao, retries, sinais, cancelamento e efeitos parciais.
4. Escreva testes contra a implementacao original, sem acoplar ao helper que sera extraido.
5. Rode o teste focal com `-count=1`, depois com `-race -count=1`.
6. Registre `_workspace/complexity/characterization/<subwave>.md` com contratos e comandos.
7. So libere a refatoracao quando os testes novos passarem no original.

## Saida minima

```markdown
# <subwave> characterization
- Base SHA:
- Ownership:
- Functions/scores:
- Observable contracts:
- Tests added before refactor:
- Original implementation evidence:
- Gaps/blockers:
```

## Casos sensiveis

Para IBC v2, upgrades, Bech32 ou backpressure, exija caracterizacao adicional e limite o trabalho seguinte a extracao estrutural. Se o contrato nao puder ser observado com seguranca, bloqueie esse arquivo e devolva-o ao orquestrador.
