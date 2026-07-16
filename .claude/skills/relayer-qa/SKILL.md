---
name: relayer-qa
description: "Valida incrementalmente o harness do relayer cruzando branches, scores, build, testes e contratos IBC entre componentes. Use obrigatoriamente apos cada lote de refatoracao, mudanca IBC/SDK, atualizacao do roadmap, reexecucao ou melhoria; nao aceite apenas existencia de arquivos ou build verde."
---

# QA integrada do relayer

## Verificacao de fronteiras

1. Compare evento on-chain com parser e tipo interno.
2. Compare tipo interno com cache/runtime do processor.
3. Compare decisao do processor com mensagem do provider e codec registrado.
4. Compare mensagem enviada com confirmacao, retry, metricas e CLI/config.
5. Compare dependencias declaradas com APIs realmente importadas.
6. Compare tabela de branches com comandos live e tabela de complexidade com nova execucao pinada.

## Assertions obrigatorias

- Maximo ciclomatico <= 9 no escopo manuscrito.
- Maximo cognitivo <= 9 no mesmo escopo.
- Arquivos gerados sao excluidos por regra deterministica e auditavel.
- `go test ./...` e `go build ./...` passam em todos os modulos declarados pelo `go.work`, incluindo raiz e `interchaintest`, ou a falha preexistente esta isolada.
- Cada item de roadmap aponta para codigo atual, fonte primaria e teste de aceitacao.

## Saida

Grave `_workspace/04_integration_qa_report.md` com `assertion | resultado | evidencia | severidade | acao`.

Reexecute uma falha uma vez. Dependencia externa indisponivel produz `nao verificado`, nunca `aprovado`.
