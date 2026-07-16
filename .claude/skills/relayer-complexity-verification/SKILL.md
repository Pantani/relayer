---
name: relayer-complexity-verification
description: "Revisa independentemente diffs da campanha de complexidade do Go relayer e executa gates locais/globais. Use obrigatoriamente apos cada lote, em reexecucao, atualizacao, correcao, retomada ou melhoria do resultado."
---

# Verificacao da campanha de complexidade

## Assertions obrigatorias

1. Caracterizacao passou na implementacao original e continua verde.
2. Diff contem somente extracao estrutural no escopo atribuido.
3. APIs, erros, logs, metricas, ordem, retries, timeouts, cancelamento e concorrencia nao mudaram.
4. `gocyclo@v0.6.0 -over 10` e `gocognit@v1.2.1 -over 10 -test` nao listam os arquivos de producao tocados.
5. Todo helper novo esta `<=10/<=10`.
6. Testes focais com `-race` e testes do pacote sem race passam.
7. `make lint`, `go build -mod=readonly ./...` e `git diff --check` passam.
8. Inventario global reduziu estritamente e nao contem nova violacao.

## Procedimento

1. Leia base, diff, caracterizacao e inventarios antes/depois.
2. Inspecione cada hunk por alteracao observavel e ownership.
3. Rode os gates na ordem de feedback mais rapido: formato/diff, testes focais, scores locais, pacote, build/lint, global.
4. Reexecute uma falha uma vez quando seguro; nao masque falha externa.
5. Grave `_workspace/complexity/reviews/<subwave>.md`.

## Saida

Use `assertion | resultado | evidencia | severidade | acao` e finalize com `APPROVED`, `CHANGES_REQUESTED` ou `BLOCKED`. Um gate nao executado e `NOT_VERIFIED`, nunca `PASS`.
