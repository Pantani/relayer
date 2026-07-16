# M1.1a — code review e correções

Data: 2026-07-15  
Base: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Resultado

Zero achados conhecidos abertos após revisão paralela, CodeRabbit e regressões.

| origem | severidade | achado | resolução |
|---|---|---|---|
| harness | warning | classificação rejeitada perdia attrs raw | `IBCEventIssue.Event` preserva clone lossless diagnóstico |
| harness | warning | erro de correlação sumia quando classificação também falhava | ambos são preservados, correlação primeiro |
| harness | warning | action indexada inválida reutilizava action anterior | índice é invalidado/poisoned em qualquer ordem |
| harness | warning | action legacy inválida reutilizava action precedente | estado fica poisoned até nova action válida; keeper module-only é neutro |
| CodeRabbit | critical | `AuthInfo.Fee == nil` podia causar panic em EIP-712 | validação explícita + regressão malformed sign-doc |
| CodeRabbit | minor | fuzz de correlação validava apenas sidecars produzidos | fuzz agora exige zero sidecars inválidos e exatamente um com action/index esperados nos casos válidos |

O CodeRabbit CLI `0.6.5` armazenou exatamente os dois achados acima. A tentativa
de rerun após as correções foi recusada pelo limite temporário de reviews OSS;
por isso a verificação de fechamento foi feita localmente e por um segundo
agente, sem executar instruções provenientes do review.

## Evidência de fechamento

- `AuthInfo.Fee` é validado antes de `buildProtobufTypedData`; teste focado com
  Fee omitido retorna `auth info fee is required` e não ocorre panic.
- O fuzz deriva a validade de action/índice, rejeita conflito/índice inválido e
  compara `Action.Type`/`Action.Index` nos casos aceitos.
- Focused normal/race e rodadas fuzz passam após as correções.
- Funções/testes tocados pelos fixes permanecem com complexidade máxima 9.
- O relatório independente detalhado está em
  `_workspace/27_m1_1a_static_review.md`.
