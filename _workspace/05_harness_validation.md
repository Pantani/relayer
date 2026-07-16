# Validação do build do harness

Data: 2026-07-15  
SHA-base: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Estrutura

| Verificação | Resultado |
|---|---|
| Agentes especializados | PASS — 4 arquivos |
| Skills especializadas + orquestrador | PASS — 5 arquivos |
| Agentes com `model: opus` | PASS — 4/4 |
| Comandos legados em `.claude/commands` | PASS — diretório ausente |
| Trigger tests | PASS — 10 positivos e 10 negativos |
| Skills com menos de 500 linhas | PASS — maior arquivo com 78 linhas |
| Fan-out/fan-in, retry, reexecução e QA incremental | PASS — definidos no orquestrador |

## Artefatos do fan-out/fan-in

| Produtor | Artefato | QA |
|---|---|---|
| branch-archaeologist | `01_branch_archaeologist_inventory.md` | 79 linhas/79 nomes únicos, iguais ao origin live |
| complexity-engineer | `02_complexity_engineer_baseline.md` | 1.327 funções; união independente de 158 violações |
| ibc-gap-analyst | `03_ibc_gap_analyst_roadmap.md` | 51 capacidades, nove colunas válidas, PFM corrigido para Classic-only |
| integration-qa | `04_integration_qa_report.md` | parecer parcial com blockers e comandos reproduzíveis |

## Verificações executadas

```text
GOTOOLCHAIN=go1.21.13 go build -mod=readonly ./...  PASS
bash -n scripts/check-complexity.sh               PASS
git diff --check                                  PASS
make complexity                                   FAIL esperado: 158 violações
go test -mod=readonly ./...                       FAIL preexistente: TestQueryBaseFee
roadmap matrix shape                              PASS: 51/51
harness structure                                 PASS: 4 agents, 5 skills, 20 triggers
```

O gate vermelho e o teste live não são falhas do harness: são blockers reais que ele tornou explícitos. O próximo lote deve corrigir o teste de dynamic fee e iniciar a redução de complexidade sem relaxar o limite.

## Casos de execução exercitados

1. **Auditoria de branches:** o produtor gerou nomes incorretos na primeira passagem; QA detectou a quebra de fronteira e o produtor corrigiu 79/79 nomes antes da integração.
2. **Roadmap IBC:** a primeira matriz atribuía PFM a v2; inspeção do tag v11.2.0 mostrou implementação Classic-only, e produtor/QA corrigiram e revalidaram a matriz.
3. **Complexidade:** o baseline e a recontagem independente chegaram às mesmas contagens, e o runner confirmou que score 9 passa enquanto score 10 falha.

Esses casos demonstram o caminho normal, retorno de QA ao produtor e preservação explícita de blocker quando o código ainda não atende ao contrato.
