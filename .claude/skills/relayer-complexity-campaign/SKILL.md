---
name: relayer-complexity-campaign
description: "Orquestra por equipe toda a campanha de complexidade do Go relayer ate zero violacoes. Use obrigatoriamente para iniciar, continuar, recuperar falha, retomar parcialmente, atualizar, corrigir, reexecutar, rodar uma subwave ou melhorar resultados anteriores de complexidade."
---

# Orquestrador da campanha de complexidade

## Modo de execucao: equipe com supervisor e producer-reviewer

Use os quatro agentes persistentes em `.claude/agents/`. Em runtimes com selecao de modelo, use `opus`. O `complexity-orchestrator` e o unico integrador Git/GitHub; os demais editam apenas arquivos sob ownership disjunto. Um worktree tem lease exclusivo: apenas um editor pode estar `ACTIVE`; finalize ou interrompa o holder anterior antes do handoff.

## Phase 0: contexto e retomada

1. Leia ledger, ownership, inventory, characterization e reviews existentes.
2. Reconcile com `origin/main`, branch atual, status, PRs e heads vivos.
3. Sem estado: execucao inicial. Estado + pedido parcial: reexecute so a subwave afetada. Base nova: preserve o historico no ledger e gere inventario novo.
4. Em falha anterior, retome do primeiro gate nao comprovado; nao repita trabalho aprovado sem mudanca de base/diff.

## Phase 1: snapshot e ownership

1. Exija worktree limpo, `git fetch origin --prune`, base SHA e PRs abertos.
2. Execute inventario integral com pins `gocyclo@v0.6.0`, `gocognit@v1.2.1`, `-over 10`.
3. Atualize `_workspace/complexity/{inventory,ownership,ledger}.md`.
4. Escolha o menor escopo seguro de 1-3 arquivos de producao; registre holder/status do lease e nao compartilhe arquivo/worktree entre editores.

## Phase 2: pipeline por subwave

1. O caracterizador escreve testes e documento no original.
2. O orquestrador confirma testes verdes e libera producao.
3. O engenheiro faz somente extracao estrutural e mede os arquivos tocados.
4. O verificador revisa independentemente e roda os gates.
5. Falha relacionada ao diff retorna uma vez ao produtor; conflito de ownership retorna ao orquestrador.

## Phase 3: integracao e publicacao

Somente o orquestrador:

1. confirma reducao global estrita e review aprovado;
2. atualiza ledger, ownership e tabela de progresso;
3. commita o escopo, faz push e abre PR pequeno;
4. usa PR stacked quando necessario e nunca faz merge automatico;
5. verifica checks do GitHub quando disponiveis e segue para a proxima subwave segura.

## Gates por PR

- Scores locais com pins/limite estritos; arquivos de producao tocados sem violacao.
- Testes focais `-race`, pacote sem race, `make lint`, `go build -mod=readonly ./...`, `git diff --check`.
- Review independente, inventario global e checks GitHub quando disponiveis.
- `make complexity` e obrigatorio no fechamento; intermediarios permanecem vermelhos sem alterar o gate.

## Condicao terminal

Continue enquanto houver frente segura. Termine somente com ciclo/cognitiva/uniao `0/0/0`, maximos globais `<=10/<=10`, complexity/build/lint/test/race verdes, CI impedindo score `>10`, todos os PRs abertos e ledger completo. Sem frente segura, reporte apenas um bloqueio que realmente exija decisao, credencial ou autoridade do usuario.

## Dados e comunicacao

`snapshot -> inventory/ownership -> characterization -> refactor -> independent review -> gates -> ledger -> commit/push/PR -> next subwave`

Mensagens devem incluir subwave, base SHA, arquivos, scores, contratos, comandos e bloqueios. Descobertas que mudem ownership vao imediatamente ao orquestrador.

## Erros

- Ref/PR mudou: atualize snapshot e ownership antes de integrar.
- Teste original falha: separe baseline e nao atribua ao diff.
- CI externo/flaky: registre logs; nao mude producao/workflow para mascarar.
- IBC v2/upgrades/Bech32/backpressure sem prova: pule para arquivo disjunto e registre.

## Testes de trigger

Leia `references/trigger-tests.md` ao alterar description, papéis ou fluxo.

## Cenarios de teste

### Normal

Base limpa, ownership livre, inventario vivo, caracterizacao verde, refatoracao local zero, review aprovado, reducao global estrita, PR aberto e proxima subwave iniciada.

### Falha e retomada

Um teste focal falha apos o diff; o engenheiro recebe uma tentativa de correcao. Persistindo, o escopo fica bloqueado no ledger, ownership e liberado quando seguro e o orquestrador continua em arquivo disjunto. Na retomada, inicia no gate falho.
