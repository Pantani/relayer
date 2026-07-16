# M1.1b-d — progresso do harness de integração

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
Base preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Resultado

Etapa intermediária concluída: **framework migrated; runtime v11.2 not
verified**. O módulo `interchaintest` deixou de carregar a fronteira Store v1,
compila tanto no workspace quanto isoladamente e voltou a exercitar o relayer
local em um cenário Classic real. Isso não encerra IBC v2 e não é evidência de
uma aplicação chain v11.2.

## Framework e grafo

- framework oficial, sem fork:
  `github.com/cosmos/interchaintest/v11@v11.0.0-20260507171724-1a8c536981a8`;
- `go.work` contém `.` e `./interchaintest`;
- módulo isolado passa `go mod verify`, compile readonly e build readonly;
- pacotes carregados usam SDK 0.54.3, CometBFT 0.39.3, IBC-Go v11.2.0,
  Store v2 e Log v2;
- `go mod why` confirma que Store/Log v1 e IBC-Go v10 não são necessários;
- o gate `interchaintest-contract` valida workspace, isolamento, build e os
  unit tests do adapter antes das lanes Docker.

## Adaptações implementadas

- interface `ibc.Relayer` v11 completa, com novos argumentos para keys,
  clients, path update e link;
- `SigningAlgorithm` propagado para config e restore de keys;
- migração para as APIs Docker/Moby atuais;
- imagens Gaia dos cenários pinadas em tags heighliner existentes e
  multi-arquitetura;
- tipo read-only local para decodificar o client localhost stateful legado
  removido do IBC-Go v11; o relayer não cria nem atualiza esse tipo;
- matcher ignora client types não suportados antes da comparação Tendermint;
- `local.Dockerfile` copia o binário produzido em `/go/bin/rly`;
- métodos v11 ainda não suportados retornam erro explícito, sem panic;
- jobs Docker dependem do contrato do módulo; a lane localhost foi removida
  porque `simd:v8.0.0-beta.1` não existe.

## Evidência executável

```text
make interchaintest-contract                         PASS
focused adapter tests with -race                    PASS
root legacy-localhost/matcher tests with -race      PASS
isolated go mod verify/test-compile/build            PASS
loaded legacy Store/Log/IBC v10 packages             0
go test -mod=readonly -p 1 ./...                     PASS (387 / 52 packages)
make lint                                             PASS (0 issues)
actionlint interchaintest workflow                    PASS
local.Dockerfile complete image build                PASS
container /bin/rly version smoke                      PASS (SDK 0.54.3, Go 1.25.9)
git diff --check                                    PASS
```

O teste Docker focal abaixo executou o relayer local com Gaia v14.1.0 e
Osmosis v22.0.0:

```text
TestRelayerInProcess/.../relayer_setup
PASS: 9 subtests
```

O fluxo chegou a clients, connections e channels. Durante o diagnóstico, o
harness também revelou e corrigiu: tag Gaia upstream inexistente, imagem Gaia
amd64 inadequada ao host arm64, flag `--client-tp` vazia e decode do localhost
stateful legado.

## Complexidade

Toda função criada ou alterada neste sublote fica abaixo de 10 nas duas
métricas. O inventário global melhorou para:

```text
cyclomatic violations: 83  (max 48)
cognitive violations:  134 (max 99)
union:                  138
```

O `make complexity` permanece vermelho de propósito: o objetivo final do
programa ainda exige eliminar as 138 funções herdadas.

## Revisão

A revisão focal paralela identificou quatro findings. Todos foram tratados:

1. caminho `/go/bin/rly` corrigido no Dockerfile;
2. job localhost com imagem inexistente removido;
3. signing algorithm propagado e testado;
4. unit tests do adapter incluídos no contrato executado pela CI.

A imagem local resultante foi construída e executada em `linux/arm64`; seu
manifest local é
`sha256:62dc65ac3d3ffe4426030bf6afeb5fb5885fe7abcb6e3868e091b580926fe124`.

CodeRabbit CLI não revisou o diff acumulado porque a árvore possui 193 arquivos
e o modo OSS aceita no máximo 150. A cobertura substituta foi revisão focal
independente, testes com race, E2E Docker, compile/build readonly e gates de
grafo/complexidade.

## Limites e próximo lote

- não existe imagem publicada `ghcr.io/cosmos/ibc-go-simd:v11.2.0`;
- os cenários atuais ainda provam compatibilidade Classic com chains antigas;
- falta construir `simd` do source oficial v11.2.0, registrar SHA/digest e
  executar Classic events, legacy, timeout, localhost e misbehaviour;
- depois dessa baseline runtime, abrir a primeira fatia vertical v2:
  CounterpartyInfo/config, proof queries e builders/broadcast com resultado
  NOOP/SUCCESS/FAILURE.

Nenhum commit ou push foi criado neste sublote.
