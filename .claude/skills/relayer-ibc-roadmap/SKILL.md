---
name: relayer-ibc-roadmap
description: "Compara o relayer com IBC Go/IBC v2 e Cosmos SDK atuais e cria roadmap completo, versionado e testavel. Use obrigatoriamente para suporte IBC v2, upgrades IBC/SDK, lacunas funcionais, atualizacao, reexecucao ou melhoria do roadmap; nao confunda requisitos da chain com responsabilidades do relayer."
---

# Roadmap IBC/SDK do relayer

## Procedimento

1. Fixe o SHA atual, as dependencias diretas e as capacidades existentes por leitura de codigo e testes.
2. Resolva versoes atuais em fontes oficiais: release/tag, migration guide, docs e especificacoes.
3. Modele separadamente IBC Classic e IBC v2. Uma conexao pertence a um protocolo, nao e convertida implicitamente.
4. Cruze cada capacidade por toda a fronteira: descoberta de eventos, parsing, estado/cache, construcao de mensagem, assinatura/broadcast, confirmacao, retry, CLI/config e observabilidade.
5. Separe requisitos de relayer, chain, light client, app/middleware e infraestrutura de teste.
6. Para cada lacuna, defina dependencia, risco, teste unitario, teste interchain e criterio de aceite.
7. Grave `_workspace/03_ibc_gap_analyst_roadmap.md` e uma versao final em `docs/ibc-v2-roadmap.md`.

## Matriz obrigatoria

`capacidade | fonte/versao | estado atual | lacuna | componente | dependencia | teste | prioridade | marco`

## Regra temporal

Inclua data de consulta e links primarios. Releases e prereleases ficam em linhas distintas.
