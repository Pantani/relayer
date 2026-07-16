---
name: relayer-complexity
description: "Mede, reduz e bloqueia complexidade ciclomatica e cognitiva do codigo Go manuscrito do relayer abaixo de 10. Use obrigatoriamente em auditorias, refatoracoes, builds, CI, regressao, atualizacao ou reexecucao de complexidade; nao use para codigo protobuf gerado."
---

# Complexidade do relayer

## Contrato

- O limite e estrito: cada funcao auditada deve ter score maximo 9 em ambas as metricas.
- O gate falha se qualquer funcao atingir score 10 ou superior em `gocyclo` ou `gocognit`.
- Audite todo Go manuscrito, incluindo testes. Exclua apenas arquivos cujo cabecalho contenha o marcador canonico `Code generated ... DO NOT EDIT.`; nome/extensao sozinhos nao bastam.
- Nao use `nolint`, baseline tolerante ou alteracao do limite para produzir verde artificial.

## Procedimento

1. Leia Makefile, workflows e configuracao do linter antes de instalar ferramentas.
2. Use versoes pinadas de `gocyclo` e `gocognit`; registre versoes e comandos.
3. Produza ranking completo por `arquivo:funcao`, score, linha, pacote e categoria producao/teste.
4. Refatore em lotes pequenos: retornos antecipados, extracao por responsabilidade, tabelas de despacho e tipos intermediarios.
5. Depois de cada lote, rode testes focados, ambas as metricas e QA nas fronteiras alteradas.
6. Integre um alvo `make complexity` e CI que falhe ao encontrar score >= 10.
7. Grave baseline e progresso em `_workspace/02_complexity_engineer_baseline.md`.

## Criterios de aceite

- Nenhuma funcao manuscrita com score >= 10 nas duas ferramentas.
- Build e testes unitarios passam.
- Gate e reproduzivel sem alterar `go.mod`.
