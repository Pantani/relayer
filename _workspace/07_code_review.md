# Revisao do primeiro lote

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`

## Rodada 1

O CodeRabbit reportou oito observacoes. Cinco foram incorporadas:

- QA passou a validar os modulos raiz e interchaintest;
- auditoria de branches passou a receber base/default branch do orquestrador;
- o gate de complexidade passou a reprovar quando qualquer score for `>=10`;
- o retry do orquestrador foi alinhado a uma repeticao;
- o roadmap distingue a implementacao historica de dynamic fee do contrato atual.

Tres observacoes nao foram aplicadas:

- remover `rtk` do inventario contraria a instrucao local obrigatoria do repositorio;
- registrar SHA imutavel do patch exigiria commit, fora do escopo deste lote local; a base imutavel esta registrada e o working tree permanece explicitamente nao commitado;
- habilitar imediatamente `errcheck`, `staticcheck` e `unused` introduziria 118 falhas herdadas (50, 50 e 18 respectivamente); a ativacao progressiva ficou registrada sem criar exclusoes ou `nolint` de baseline.

## Rodada 2

Duas observacoes validas foram incorporadas ao contrato de `integration-qa`:

- os relatorios anteriores `_workspace/04_integration_qa_report.md` e `_workspace/05_harness_validation.md` agora sao entradas obrigatorias da QA incremental;
- a aprovacao exige matriz explicita de versoes IBC/SDK, rastreabilidade por superficie e teste de aceitacao especifico para IBC v2.

Resultado apos triagem: nenhuma observacao valida conhecida permanece sem tratamento neste lote.

## Rodada 3

Uma observacao documental foi incorporada: `docs/maintenance-baseline.md` agora referencia a trilha completa de artefatos `_workspace/00` a `_workspace/07`.

A tentativa de revalidacao automatizada dessa correcao foi bloqueada pelo limite gratuito do CodeRabbit, com espera informada de 46 minutos. A alteracao foi validada localmente com parsing YAML e `git diff --check`; nao houve mudanca adicional de codigo nesta rodada.
