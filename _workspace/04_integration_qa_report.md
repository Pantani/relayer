# QA incremental do programa de manutenção do relayer

Data: 2026-07-15 (America/Sao_Paulo)  
SHA auditado: `bef2e868f157659b403fe1303ee121fb69fec9e6`  
Escopo: snapshot, harness, inventário de branches, baseline/runner de complexidade, `Makefile`, build/test Go 1.21 e amostra do roadmap IBC/SDK. Nenhum código de produção foi alterado por esta QA.

## Parecer

**PARCIAL / ainda não pronto para enforcement final.** O harness, o inventário corrigido, o baseline e o gate local são reproduzíveis. As contagens centrais foram confirmadas de forma independente e o gate interpreta corretamente `< 10` como máximo 9. Porém, o código auditado ainda possui 158 funções fora do contrato, o teste unitário ainda depende de um endpoint externo e falha, e nenhum workflow CI chama `make complexity`. O roadmap chegou durante a QA, teve a afirmação de PFM v2 corrigida e passou por amostragem de versão/fronteiras, mas suas 51 linhas não foram todas validadas semanticamente contra tags upstream nesta rodada.

## Assertions

| assertion | resultado | evidencia | severidade | acao |
|---|---|---|---|---|
| Snapshot fixado e sem troca de branch | **APROVADO** | `HEAD=bef2e868…`, checkout detached, `origin=git@github.com:Pantani/relayer.git`; snapshot e artefatos usam o mesmo SHA | baixa | Manter o SHA congelado até fechar este ciclo de análise |
| Inventário de branches do fork | **APROVADO após correção** | Reexecução live: 79 heads; tabela: 79 linhas e 79 nomes únicos; `comm` entre nomes reportados e `git ls-remote --heads origin` retornou 0 diferenças. O literal `${ref#refs/heads/}` foi removido | alta, resolvida | Repetir o diff de nomes quando o snapshot remoto mudar |
| PRs fork versus parent | **APROVADO** | `gh pr list`: 0 PRs abertos em `Pantani/relayer` e 35 em `cosmos/relayer`; o relatório mantém os conjuntos separados | média | Nunca tratar PR do parent como branch/PR aberto do fork |
| Estrutura do harness | **APROVADO** | 4 agentes, 5 skills, todos os agentes com `model: opus`; definições contêm papel, I/O, erro e colaboração; `.claude/commands` ausente; `CLAUDE.md` contém ponteiro e histórico; orquestrador declara fan-out/fan-in, reexecução, retry e QA incremental | média | Preservar validação estrutural a cada evolução do harness |
| Escopo de arquivos Go | **APROVADO** | Recontagem independente do tree do SHA: 173 `.go`, 38 gerados, 135 manuscritos; destes, 95 produção e 40 teste/test-support | alta | Usar a mesma regra em qualquer regeneração do baseline |
| Exclusão de gerados | **APROVADO** | Os 38 arquivos com marcador em qualquer posição também são exatamente os 38 encontrados no bloco inicial de 40 linhas por `^// Code generated .* DO NOT EDIT[.]$`; extensão/nome não participam da decisão | alta | Manter a exclusão canônica e falhar fechado para arquivo ausente/malformado |
| Tooling pinado compatível com Go 1.21 | **APROVADO** | `gocyclo@v0.6.0` declara Go 1.18; `gocognit@v1.1.4` declara Go 1.19; `v1.2.1` declara Go 1.24. O gate foi executado com `GOTOOLCHAIN=go1.21.13` | alta | Manter `gocognit@v1.1.4` enquanto o módulo declarar Go 1.21 |
| Inventário de funções e violações | **APROVADO** | Nova medição: 1.327 funções, 98 com ciclo >=10, 152 com cognitiva >=10, 92 em ambas e união 158. Máximos amostrados: 48/169. Confere com `02_complexity_engineer_baseline.md` | crítica | Usar `arquivo:função` como chave estável e refazer a união em cada lote |
| Semântica estrita do limite | **APROVADO** | Em arquivo cujo máximo é 9, ambas as ferramentas retornaram 0 com `-over 9` e 1 com `-over 8`; no repo atual ambas retornam 1 com `-over 9`. Logo, score 9 passa e score >=10 falha | crítica | Não trocar `-over 9` por `-over 10` |
| Gate local e `Makefile` | **APROVADO** | `bash -n` passa; `make -n complexity` aponta para `bash ./scripts/check-complexity.sh`; alvo está em `.PHONY`; `make complexity` retorna 2 porque o script retorna 1 diante da dívida atual | alta | Manter o alvo como entrada canônica local |
| Contrato final de complexidade no código atual | **REPROVADO** | 158 funções manuscritas ainda têm pelo menos um score >=10; o estado verde solicitado ainda não foi implementado | crítica | Executar os lotes do baseline até ciclo=0, cognitiva=0 e união=0 |
| Enforcement em CI | **REPROVADO** | Busca em `.github/workflows` não encontrou `complexity`, `check-complexity` ou `make complexity`; apenas o `Makefile` e o script local conhecem o gate | alta | Adicionar job Go 1.21 que execute `make complexity`; ele ficará vermelho até a dívida ser zerada ou os lotes serem coordenados sem relaxar o limite |
| Imutabilidade de manifests durante medição | **APROVADO** | Hashes continuam `2d272a… go.mod`, `7d9294… go.sum`, `89c29c… interchaintest/go.mod`, `52dbaae… interchaintest/go.sum`; `git status` não mostra mudanças nesses quatro arquivos | alta | Continuar usando `go run módulo@versão` sem adicionar ferramentas ao módulo de produção |
| Build sob contrato Go 1.21 | **APROVADO** | `GOTOOLCHAIN=go1.21.13 go build -mod=readonly ./...` retornou 0 | alta | Repetir em Linux no CI |
| Testes unitários sob Go 1.21 | **PARCIAL: falha preexistente isolada** | `go test -mod=readonly ./...` falhou somente em `relayer/chains/cosmos.TestQueryBaseFee`; repetição focada com `-count=1` falhou igual, ao tentar ler bytes protobuf (`0x11...`) como decimal base 10 a partir de RPC público | alta | Substituir a rede pública por fixture protobuf tipada e deixar integração de rede opt-in |
| Roadmap: forma e cobertura de testes | **APROVADO estruturalmente** | Matriz corrigida tem 51 capacidades, 9 colunas por linha e nenhuma célula vazia de fonte ou teste. Marcos M0-M4 e critérios de aceite estão presentes | média | Manter uma checagem de shape para impedir pipes Markdown não escapados |
| Roadmap: versões alvo | **APROVADO por amostragem** | Metadados oficiais confirmam `ibc-go/v11@v11.2.0` (`cfc072e…`, Go 1.25.9, publicado em 2026-07-15) e SDK `v0.54.3` (`046046a…`, Go 1.25.9, 2026-05-05). O release SDK referencia CometBFT 0.39.3 | alta | Revalidar tags/advisories antes da primeira PR de upgrade, pois v11.2.0 saiu no dia da auditoria |
| Roadmap: divergência multi-payload | **APROVADO por amostragem** | No tag v11.2.0, `MsgSendPacket.ValidateBasic` exige `len(msg.Payloads) == 1`; o roadmap não promete multi-payload e cria gate de interoperabilidade | alta | Manter feature desabilitada até teste do tag/keeper exato |
| Roadmap: middleware v2 | **APROVADO após correção** | No módulo v11.2.0 existem `callbacks/v2` e `rate-limiting/v2`; PFM importa channel/port Classic e não possui diretório `/v2`. O artefato 03 agora trata PFM como Classic-only e exige rejeição explícita em path v2 | alta, resolvida | Não prometer PFM v2 até existir adapter upstream verificável |
| Roadmap: lacuna v2 do código atual | **APROVADO por fronteira amostrada** | Código importa `/v8`, `PacketInfo` e `ChannelKey` são channel-centric, atributos usam `map[string]string`, e não há tipos/eventos `source_client`, `encoded_packet_hex`, `MsgUpdateClientConfig` ou channel/v2. A conclusão “IBC Classic, sem implementação parcial v2” é sustentada | crítica | Implementar fatias verticais evento -> parser -> estado -> proof/msg -> broadcast -> confirmação |
| Roadmap: rastreabilidade linha a linha | **PARCIAL** | Cada linha tem fonte/versão e teste, e fontes-chave foram amostradas; porém nem todas as 51 capacidades apontam diretamente para tag/arquivo upstream e evidência local específica na própria linha | média | Na execução de cada lote, converter a linha correspondente em evidência exata: URL/tag upstream, arquivo/símbolo local e teste de aceite executável |

## Evidência reproduzível resumida

```text
tracked=173 generated=38 manual=135 production=95 test=40
functions=1327 cyclo_ge_10=98 cognit_ge_10=152
intersection=92 union=158

gocyclo: -over 9 => 0 em score máximo 9; -over 8 => 1
gocognit: -over 9 => 0 em score máximo 9; -over 8 => 1

Go 1.21.13 build: PASS
Go 1.21.13 tests: FAIL isolado em TestQueryBaseFee, reproduzido duas vezes
Make complexity no baseline: FAIL esperado

branches report/live/unique: 79/79/79; name diff: 0
open PRs fork/parent: 0/35
roadmap rows/bad shape/missing source/missing test: 51/0/0/0
```

Fontes primárias amostradas durante a QA:

- <https://github.com/cosmos/ibc-go/releases/tag/v11.2.0>
- <https://github.com/cosmos/ibc-go/blob/v11.2.0/go.mod>
- <https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/v2/types/msgs.go>
- <https://github.com/cosmos/cosmos-sdk/releases/tag/v0.54.3>
- <https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/apps/packet-forward-middleware/ibc_middleware.go>

## Comandos finais de revalidação

```bash
GOTOOLCHAIN=go1.21.13 make complexity
GOTOOLCHAIN=go1.21.13 go build -mod=readonly ./...
GOTOOLCHAIN=go1.21.13 go test -mod=readonly ./...
GOTOOLCHAIN=go1.21.13 go test -mod=readonly -count=1 -run '^TestQueryBaseFee$' ./relayer/chains/cosmos

git ls-remote --heads origin
gh pr list --repo Pantani/relayer --state open --limit 1000 --json number
gh pr list --repo cosmos/relayer --state open --limit 1000 --json number

git diff --check
git status --short -- go.mod go.sum interchaintest/go.mod interchaintest/go.sum
```

## Limitações desta rodada

- Context7 não estava exposto no runtime desta equipe; a amostragem documental usou exclusivamente repositórios/releases oficiais e módulos Go baixados pelas versões exatas.
- Não foram executados interchaintests Docker, testnets v11, proof API/Eureka ou testes de interoperabilidade; essas capacidades permanecem plano, não suporte validado.
- Esta QA não refatorou funções nem adicionou workflow CI, pois o escopo recebido permitia correção apenas de erro mecânico no harness/build. As correções mecânicas dos artefatos 01 e 03 foram feitas pelos respectivos produtores e revalidadas aqui.

## Reexecução incremental — M0.3

Data: 2026-07-15  
Parecer: **APROVADO para o lote M0.3, com dívida global explícita.**

O modelo protocol-neutral, adaptadores Classic/v2 contract-only, schema de path discriminado e guardas contra dispatch v2 no runtime Classic passaram por testes unitários/race, build dos dois módulos, lint, module verify, diff-check e revisão CodeRabbit final sem findings. Toda função criada ou tocada fica abaixo de 10 nas duas métricas; o core novo tem máximo 8/8.

O gate global continua vermelho corretamente: 86 violações ciclomáticas, 138 cognitivas, interseção 80, união 144 e máximos 48/99. A evidência completa, limitações e comandos estão em `_workspace/20_m0_3_qa.md`.

## Reexecução incremental — M1.1a

Data: 2026-07-15  
Parecer: **APROVADO para ingestão/observação IBC v2; relay v2 continua bloqueado.**

| assertion | resultado | evidência | severidade residual | ação seguinte |
|---|---|---|---|---|
| Importar `/v11` preserva o grafo atual | **REPROVADO e isolado** | Prova MVS eleva SDK 0.54/Comet 0.39 e o build legado falha em `x/crisis`, `sr25519` e store v1/v2 | alta, fora desta fatia | M1.1b migra o grafo como unidade coerente |
| Wire local equivale a v11.2.0 | **APROVADO** | Golden oficial packet 92 bytes e ack 4 bytes; marshal/decode equivalentes; sem import SDK/Comet/v11 | baixa/transitória | remover após a migração completa |
| Eventos raw são lossless | **APROVADO** | Ordem, duplicatas e `Index` preservados; rejeições guardam clone diagnóstico `ProtocolUnspecified` | nenhuma conhecida | manter a evidência nos consumidores futuros |
| Correlação action-indexed é determinística | **APROVADO** | Join por `msg_index`, fallback precedente legado, keeper module-only neutro, conflitos e poisoning indexado/legado cobertos | nenhuma conhecida | propagar correlation id no runtime v2 |
| Classic e v2 não sofrem fallthrough | **APROVADO** | Cinco nomes compartilhados usam assinatura completa; direct parser recusa v2/ambíguo; wrapper retorna somente Classic | nenhuma conhecida | manter sidecar desconectado até state machine v2 |
| Decode packet/ack é limitado e tipado | **APROVADO** | Caps de 512 KiB antes do hex, payload 256 KiB, sentinelas `errors.Is`, cross-check de quatro attrs e fuzz | política local de cap | revalidar caps com telemetria/testnet |
| Regressão Classic | **APROVADO** | Batch misto produz uma mensagem Classic e um sidecar v2; suíte root 357/51 e focused race 118/2 | nenhuma conhecida | ampliar interchaintest após migração v11 |
| Complexidade M1.1a | **APROVADO local / REPROVADO global herdado** | Código novo máximo 7/7; teste novo abaixo de 10; global permanece 86/138/144, máximos 48/99 | dívida alta conhecida | reduzir nos lotes de state machine/broadcast |
| Review | **APROVADO após correções** | Quatro warnings do harness resolvidos; CodeRabbit minor do fuzz e critical Ethermint nil Fee resolvidos com regressões | zero aberto | rerodar review em cada lote |

Gates finais executados: focused race 118, fuzz 2x2s, root `go test ./...`
357/51, chains/protocol 175/38, build root/interchaintest, lint 0, module
verify e diff-check. Docker/testnet v11 permanecem fora de escopo porque ainda
não há provider/runtime v11 compilável.
