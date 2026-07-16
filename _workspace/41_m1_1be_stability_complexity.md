# M1.1b-e — estabilidade do mock e primeiro lote incremental de complexidade

Data: 2026-07-15  
Branch: `Pantani/cx/m0-baseline`  
Base preservada: `bef2e868f157659b403fe1303ee121fb69fec9e6`

## Resultado

O timeout atribuído anteriormente a um flake concorrente era um falso positivo
do comando de stress. `TestMockChainAndPathProcessors` aguardava sempre o
deadline local de 20 segundos; vinte repetições exigiam no mínimo 400 segundos,
mas o comando usava timeout global de 120 segundos. O dump mostrava os três
processors aguardando em `select`, sem goroutine bloqueada em enqueue.

O teste agora encerra quando os seis contadores esperados de Send, Recv e Ack
foram observados nas duas chains. O timeout de dez segundos é somente uma guarda.
O stress corrigido passou vinte repetições com race dentro do limite de 90
segundos.

## Complexidade

Os três parsers públicos de identificadores em `relayer/events.go` agora usam
um helper comum, e a duplicação privada de client ID em `relayer/client.go` foi
reduzida a um wrapper trivial. Isso preserva sem alteração textual o corpo
herdado de `CreateClient`, que continua fora deste lote. Foram preservados:

- tipos de evento aceitos para client, connection e channel;
- ordem de seleção do primeiro evento aplicável;
- atributo presente com valor vazio;
- mensagens de erro públicas exatas.

Testes novos cobrem create-client, connection init/try, channel init/try,
eventos e atributos irrelevantes, atributo ausente, valor vazio e erros exatos.

```text
antes:  cyclomatic 83, cognitive 134, union 138, max 48/99
depois: cyclomatic 83, cognitive 130, union 134, max 48/99
delta:   0 / -4 / -4
```

Nenhuma função tocada ou criada aparece em `gocyclo -over 9` ou
`gocognit -test -over 9`.

## Evidência executável

```text
focused parser tests with -race                    PASS (7)
go test -mod=readonly -count=1 ./relayer           PASS (59)
mock stress -race -count=20 -timeout=90s           PASS (20)
root race, -parallel=1                              PASS (394 / 52 packages)
root race, paralelismo padrão                       FAIL (392 PASS / 2 data races em cmd)
go build -mod=readonly ./...                       PASS
make lint                                          PASS (0 issues)
make interchaintest-contract                       PASS
git diff --check                                   PASS
focused complexity for touched/new functions      PASS
global make complexity                             EXPECTED FAIL (134 remaining)
```

## Risco separado identificado

O timeout original não reproduziu deadlock. Entretanto, existe um risco de
shutdown sob backpressure: `PathProcessor.HandleNewData` faz enqueue bloqueante
em buffers de 100 itens e não recebe contexto. Se o consumer sair e o buffer
encher enquanto um chain processor ainda está em `queryCycle`, o produtor pode
não alcançar a próxima verificação de cancelamento.

A reexecução final também reproduziu uma data race herdada entre testes
paralelos de `cmd` e a configuração Bech32 global do Cosmos SDK. A escrita
ocorre em `SetSDKConfigContext`, chamada durante `MakeCodecConfig`; os testes
`TestKeysDefaultCoinType` e `TestKeysRestoreAll_Delete` usam `t.Parallel`. Os
arquivos e caminhos envolvidos não foram alterados por este sublote, e a suíte
com `-parallel=1` passa 394 testes, mas a lane race com paralelismo padrão não
pode ser considerada verde até a configuração global ser serializada ou
eliminada.

O próximo lote de estabilidade deve tratar a data race Bech32 e adicionar uma
variante context-aware para os produtores internos, com testes determinísticos
de cancelamento e fila saturada. Em paralelo, a próxima redução de complexidade
deve continuar em uma fronteira pequena, sem entrar ainda nos hotspots que
concentram a state machine Classic/v2.

Nenhum commit, push ou PR foi criado neste lote.
