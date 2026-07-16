# Matriz de compatibilidade IBC Classic e política de depreciação

Base normativa deste documento: `fb5ecd8a2c310fd06e8466fc7f0e98dc1bc87b96`.

## Critério da matriz

Cada linha é uma célula `{linha IBC-Go, capacidade}`. O campo **Estado** usa:

- **S — suportado:** compilação, codec e runtime E2E passaram na linha e na janela declaradas;
- **R — compatibilidade apenas de leitura:** o relayer pode decodificar ou consultar o formato, mas não cria, atualiza nem promete broadcast;
- **NV — não verificado:** falta ao menos um gate obrigatório, normalmente runtime E2E na linha exata;
- **D — removido/deprecated:** não recebe novas funcionalidades; qualquer código retido existe somente para transição ou leitura legada.

O campo **C/K/E** separa as evidências: **C** = compilação contra o major indicado; **K** = compatibilidade de codec/wire coberta por fixture do major indicado; **E** = runtime E2E contra chain desse major. `✓` significa verificado, `NV` significa não verificado e `—` significa removido ou não aplicável. Compilar o relayer com `github.com/cosmos/ibc-go/v11` não comprova codec de v8/v10; um golden protobuf não comprova runtime; E2E em outro major não promove a célula para **S**.

As janelas abaixo são propostas para runtimes de chain, não para múltiplos imports simultâneos no binário:

| Linha | Janela mínima/máxima proposta | Situação da linha |
|---|---|---|
| IBC-Go v8 | `>=8.8.0,<9.0.0` | legado em depreciação; upstream já não mantém a linha |
| IBC-Go v10 | `>=10.7.0,<11.0.0` | manutenção candidata; ainda sem qualificação E2E neste repositório |
| IBC-Go v11 | `>=11.2.0,<12.0.0` | linha principal candidata e major de compilação atual; ainda sem runtime E2E v11.2 |

Não há célula **S** no snapshot: os E2E atuais usam IBC-Go v4/v7, e os testes escritos para `simd` v8 não constituem uma execução verde em v8.8.0. Isso não invalida compilação/codec v11 já cobertos; apenas impede declarar suporte de runtime para v8, v10 ou v11.

## IBC-Go v8

| Capacidade | Estado | C/K/E | Teste atual | Teste necessário | Versão mínima/máxima proposta |
|---|---:|---|---|---|---|
| Canal ordered | NV | `C:NV K:NV E:NV` | [T1] cobre conversão do enum; [T4] e a variante localhost ICA de [T3] usam fixtures anteriores ou beta, não v8.8 | E2E v8.8 com canal ordered, Send/Recv/Ack e timeout-on-close | `>=8.8.0,<9.0.0` |
| Canal unordered | NV | `C:NV K:NV E:NV` | [T1] cobre parse/enum; [T2] é E2E em v4/v7; [T3] aponta para `simd` v8.0.0, sem qualificação v8.8 | E2E v8.8 bidirecional no processor events | `>=8.8.0,<9.0.0` |
| Send | NV | `C:NV K:NV E:NV` | [T1] cobre evento/adapter Classic; [T2]/[T3] não qualificam v8.8 | fixture wire v8.8 e E2E de emissão/observação | `>=8.8.0,<9.0.0` |
| Recv | NV | `C:NV K:NV E:NV` | [T1] cobre evento/adapter Classic; [T2]/[T3] não qualificam v8.8 | fixture wire v8.8 e E2E com prova de commitment | `>=8.8.0,<9.0.0` |
| Ack | NV | `C:NV K:NV E:NV` | [T1] cobre evento, ack e proof round-trip; [T2]/[T3] não qualificam v8.8 | fixture wire v8.8 e E2E WriteAck/Acknowledgement | `>=8.8.0,<9.0.0` |
| Timeout e refund | NV | `C:NV K:NV E:NV` | [T1] cobre proof kinds; [T2] valida timeout/refund somente em v4/v7 | E2E v8.8 de timeout por timestamp e height, saldo restituído e ausência no destino | `>=8.8.0,<9.0.0` |
| Close | NV | `C:NV K:NV E:NV` | [T4] cobre close e timeout-on-close em `icad` v0.5.0/v7 RC | E2E v8.8 de CloseInit/CloseConfirm e packet pendente | `>=8.8.0,<9.0.0` |
| Channel upgrade | D | `C:— K:— E:—` | nenhum teste ou builder atual; a superfície foi removida no salto upstream para v10 [U5] | nenhum enquanto a política não for explicitamente revertida | — |
| ICA | NV | `C:NV K:NV E:NV` | [T4] usa `icad` v0.5.0/v7 RC; [T3] contém fixture v8.0.0-beta.1, não v8.8 qualificado | E2E v8.8 de registro, packet, ack, timeout e close ordered | `>=8.8.0,<9.0.0` |
| ICS-29 | D | `C:NV K:✓ E:NV` | golden wire local e E2E Juno v13/v7 em [T5]; o módulo ainda existia em v8 [U6] | somente canary v8.8 enquanto o codec legado for distribuído; sem nova funcionalidade | `>=8.8.0,<9.0.0`, somente legado |
| Localhost stateful legado | R | `C:NV K:✓ E:NV` | decoder read-only [T6]; [T3] contém `simd` v8.0.0, abaixo da janela | E2E v8.8 de query/decode e relay sem criar/atualizar client | `>=8.8.0,<9.0.0`, somente leitura do client |
| Localhost stateless | D | `C:— K:— E:—` | inexistente na linha v8; v8 mantém `ClientState` stateful [U4] | nenhum | — |
| sr25519 | NV | `C:NV K:✓ E:NV` | vetores CometBFT 0.38, sign/verify, Any e restore em [T7] | E2E v8.8 com assinatura, broadcast e restart do provider | `>=8.8.0,<9.0.0` |
| Penumbra | NV | `C:NV K:NV E:NV` | apenas unitários defensivos/log e rejeição explícita de ICS-29 em [T8] | E2E Penumbra↔chain IBC-Go v8.8 de Send/Recv/Ack/Timeout | `>=8.8.0,<9.0.0` |
| Processor events | NV | `C:NV K:NV E:NV` | parser/adapter em [T1]/[T9]; conformance [T2] usa v4/v7 | E2E v8.8 completo, incluindo replay/restart e atributos hex | `>=8.8.0,<9.0.0` |
| Processor legacy | D | `C:NV K:NV E:NV` | conformance existe em [T2], mas usa v4/v7 | canary v8.8 somente enquanto o processor ainda for distribuído | `>=8.8.0,<9.0.0`, até remoção |

## IBC-Go v10

| Capacidade | Estado | C/K/E | Teste atual | Teste necessário | Versão mínima/máxima proposta |
|---|---:|---|---|---|---|
| Canal ordered | NV | `C:NV K:NV E:NV` | [T1] compila apenas com v11; nenhum fixture/runtime v10 | build isolado v10.7, fixture wire e E2E ordered | `>=10.7.0,<11.0.0` |
| Canal unordered | NV | `C:NV K:NV E:NV` | [T1] compila apenas com v11; nenhum fixture/runtime v10 | build isolado v10.7, fixture wire e E2E unordered | `>=10.7.0,<11.0.0` |
| Send | NV | `C:NV K:NV E:NV` | [T1] sem fixture v10 | fixture v10.7 e E2E de emissão/observação | `>=10.7.0,<11.0.0` |
| Recv | NV | `C:NV K:NV E:NV` | [T1] sem fixture v10 | fixture v10.7 e E2E com prova de commitment | `>=10.7.0,<11.0.0` |
| Ack | NV | `C:NV K:NV E:NV` | [T1] sem fixture v10 | fixture v10.7 e E2E WriteAck/Acknowledgement | `>=10.7.0,<11.0.0` |
| Timeout e refund | NV | `C:NV K:NV E:NV` | [T2] cobre semântica apenas em v4/v7 | E2E v10.7 de timeout por timestamp e height com refund de saldo | `>=10.7.0,<11.0.0` |
| Close | NV | `C:NV K:NV E:NV` | [T4] cobre apenas v7 RC | E2E v10.7 CloseInit/CloseConfirm e timeout-on-close | `>=10.7.0,<11.0.0` |
| Channel upgrade | D | `C:— K:— E:—` | removido do IBC-Go v10 [U5]; nenhum código/teste local | nenhum | — |
| ICA | NV | `C:NV K:NV E:NV` | [T4] usa v7 RC; v10 mudou o default de novos canais ICA para unordered [U5] | E2E v10.7 de ICA unordered e caso ordered explicitamente negociado | `>=10.7.0,<11.0.0` |
| ICS-29 | D | `C:— K:— E:—` | removido do IBC-Go v10 [U5]; [T5] cobre apenas wire legado e runtime v7 | nenhum runtime v10; preservar somente o codec legado fora desta célula | — |
| Localhost stateful legado | D | `C:— K:✓(R) E:—` | [T6] decodifica o type URL legado sem criar/atualizar | manter golden read-only até o sunset; nenhum E2E stateful v10 | sem faixa ativa; somente leitura de estado migrado |
| Localhost stateless | NV | `C:NV K:NV E:NV` | implementação upstream é stateless em v10.7 [U3], mas não há build/fixture/runtime local v10 | build isolado e E2E localhost v10.7 de transfer e ICA | `>=10.7.0,<11.0.0` |
| sr25519 | NV | `C:NV K:✓ E:NV` | vetores e persistência locais em [T7], sem chain v10.7 | E2E v10.7 com assinatura, broadcast e restart | `>=10.7.0,<11.0.0` |
| Penumbra | NV | `C:NV K:NV E:NV` | [T8] não fixa contraparte v10 | E2E Penumbra↔chain IBC-Go v10.7 | `>=10.7.0,<11.0.0` |
| Processor events | NV | `C:NV K:NV E:NV` | [T1]/[T9] compilam com v11; [T2] usa v4/v7 | E2E v10.7 completo com atributos atuais e restart | `>=10.7.0,<11.0.0` |
| Processor legacy | D | `C:NV K:NV E:NV` | [T2] usa v4/v7 | canary v10.7 apenas se necessário durante a janela de retirada | `>=10.7.0,<11.0.0`, até remoção |

## IBC-Go v11

| Capacidade | Estado | C/K/E | Teste atual | Teste necessário | Versão mínima/máxima proposta |
|---|---:|---|---|---|---|
| Canal ordered | NV | `C:✓ K:✓ E:NV` | enum/path e builders compilam em [T1]; sem runtime v11.2 | E2E v11.2 ordered com Send/Recv/Ack e timeout-on-close | `>=11.2.0,<12.0.0` |
| Canal unordered | NV | `C:✓ K:✓ E:NV` | parse/enum e builders em [T1]; sem runtime v11.2 | E2E v11.2 bidirecional no processor events | `>=11.2.0,<12.0.0` |
| Send | NV | `C:✓ K:✓ E:NV` | evento e adapter round-trip em [T1], usando tipos v11 [U2] | E2E v11.2 de emissão, observação e commitment | `>=11.2.0,<12.0.0` |
| Recv | NV | `C:✓ K:✓ E:NV` | evento, packet e proof adapter em [T1] | E2E v11.2 com prova e confirmação on-chain | `>=11.2.0,<12.0.0` |
| Ack | NV | `C:✓ K:✓ E:NV` | WriteAck/Acknowledge e proof round-trip em [T1] | E2E v11.2 WriteAck/Acknowledgement, incluindo async ack Classic | `>=11.2.0,<12.0.0` |
| Timeout e refund | NV | `C:✓ K:✓ E:NV` | proof kinds e builders compilam em [T1]; [T2] valida refund apenas em v4/v7 | E2E v11.2 de timeout height/timestamp, ordered/unordered e saldos | `>=11.2.0,<12.0.0` |
| Close | NV | `C:✓ K:NV E:NV` | código compila; [T4] executa somente em v7 RC | fixture v11.2 e E2E CloseInit/CloseConfirm com packet pendente | `>=11.2.0,<12.0.0` |
| Channel upgrade | D | `C:— K:— E:—` | removido upstream desde v10 [U5] e ausente localmente | nenhum | — |
| ICA | NV | `C:✓ K:NV E:NV` | imports/build v11 e cenários [T4], mas runtime dos cenários é v7 RC; ICA permanece no v11 [U7] | E2E v11.2 de ICA unordered, ordered explícito, ack, timeout e close | `>=11.2.0,<12.0.0` |
| ICS-29 | D | `C:✓(local) K:✓(legado) E:—` | codec protobuf local e golden em [T5]; módulo removido upstream [U5] e padrão ICS-29 deprecated [U8] | nenhum runtime v11; somente regressão do golden legado enquanto retido | — |
| Localhost stateful legado | R | `C:✓ K:✓ E:—` | decoder read-only e interface unpack em [T6] | manter golden de decode; é proibido criar/atualizar/broadcast como client stateful | sem faixa ativa; somente leitura |
| Localhost stateless | NV | `C:✓ K:✓ E:NV` | cálculo de height/timeout local em [T6]; implementação upstream stateless [U1] | E2E v11.2 de transfer e ICA localhost, incluindo restart | `>=11.2.0,<12.0.0` |
| sr25519 | NV | `C:✓ K:✓ E:NV` | vetores, sign/verify, Any, restore e restart do provider em [T7] | E2E v11.2 com conta sr25519, broadcast e client update | `>=11.2.0,<12.0.0` |
| Penumbra | NV | `C:✓ K:NV E:NV` | provider compila e testes defensivos/log passam em [T8]; não há fixture de packet completa nem E2E | E2E Penumbra↔chain IBC-Go v11.2 de Send/Recv/Ack/Timeout | `>=11.2.0,<12.0.0` |
| Processor events | NV | `C:✓ K:✓ E:NV` | classificação/parsing/adapter em [T1]/[T9]; conformance [T2] usa v4/v7 | E2E v11.2 completo, incluindo catch-up, restart e atributos hex-only | `>=11.2.0,<12.0.0` |
| Processor legacy | D | `C:✓ K:NV E:NV` | processor compila; conformance [T2] usa v4/v7 | canary v11.2 somente durante a retirada; não bloqueia remoção depois dos gates abaixo | `>=11.2.0,<12.0.0`, até remoção |

## Política de depreciação

| Item | Política proposta | Gate para avançar | Remoção |
|---|---|---|---|
| Declaração de suporte por célula | **S** exige `C:✓ K:✓ E:✓` na mesma linha e dentro da janela; resultado em outro major é apenas evidência auxiliar | build isolado, golden/fixture do major e E2E real pinado por tag e digest | rebaixa para **NV** quando qualquer gate deixa de existir ou de rodar regularmente |
| IBC-Go v11 | linha principal candidata `>=11.2.0,<12.0.0`; sem promessa de runtime até o primeiro E2E v11.2 verde | events processor: ordered/unordered, Send/Recv/Ack, timeout/refund, close, ICA e localhost stateless | ao abrir v12, manter v11 por pelo menos duas releases minor do relayer após v12 ficar verde |
| IBC-Go v10 | linha de manutenção candidata `>=10.7.0,<11.0.0`; não declarar suporte antes da qualificação | mesma matriz Classic do v11, exceto itens removidos upstream | anunciar depreciação somente após v11 estar **S** por duas releases minor; remover após aviso mínimo de 90 dias e duas releases minor adicionais |
| IBC-Go v8 | deprecated imediatamente; aceitar somente correções de segurança/regressão na faixa `>=8.8.0,<9.0.0`, sem features | canary v8.8 do processor events enquanto a linha estiver distribuída | remover após v10 e v11 ficarem **S** por duas releases minor e aviso mínimo de 90 dias |
| Processor legacy | deprecated em todas as linhas; sem novas capacidades, formatos ou otimizações | processor events cobre todas as células suportadas, catch-up/restart e dois ciclos minor sem regressão crítica | remover na primeira major do relayer após os gates; antes disso, manter apenas canary, flag explícita e aviso de startup |
| Channel upgrade Classic | removido; não reintroduzir implicitamente por compatibilidade de tipos antigos | nova decisão explícita, matriz própria e E2E v8 se houver exceção | nenhum código de compatibilidade deve sobreviver sem decisão explícita |
| ICS-29 | deprecated/removido para v10/v11; o codec local não transforma a feature em suporte | golden wire e decode permanecem verdes enquanto houver suporte v8 legado | remover junto com v8, salvo compromisso externo documentado que exija leitura do wire histórico |
| Localhost stateful | somente leitura; nunca criar, atualizar ou usar como prova de suporte runtime v10/v11 | decode do type URL legado, sem panic, e ausência de caminhos de criação/update | remover quando nenhuma linha suportada puder retornar o estado legado e após uma release minor com aviso |
| Localhost stateless | único localhost elegível para suporte v10/v11 | transfer e ICA E2E na linha, incluindo timeout, height e restart | segue a janela normal da respectiva linha IBC-Go |
| sr25519 | independente do major IBC-Go, mas não é “suportado” numa célula sem runtime correspondente | vetores, Any/keyring, restart e broadcast E2E no major | deprecar apenas com alternativa/migração de chaves e aviso de duas releases minor |
| Penumbra | independente do import IBC-Go; cada contraparte precisa de qualificação própria | packet lifecycle completo e restart contra cada linha declarada | não remover por simples troca de major; usar aviso de duas releases minor e evidência de ausência de uso/suporte |

## Fontes primárias e testes inventariados

### Upstream

- **[U1]** IBC-Go v11.2.0: [localhost stateless](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/light-clients/09-localhost/light_client_module.go) e [módulo pinado](https://github.com/cosmos/ibc-go/blob/v11.2.0/go.mod).
- **[U2]** IBC-Go v11.2.0: [mensagens Classic de channel/packet](https://github.com/cosmos/ibc-go/blob/v11.2.0/modules/core/04-channel/types/msgs.go).
- **[U3]** IBC-Go v10.7.0: [localhost stateless](https://github.com/cosmos/ibc-go/blob/v10.7.0/modules/light-clients/09-localhost/light_client_module.go) e [módulo pinado](https://github.com/cosmos/ibc-go/blob/v10.7.0/go.mod).
- **[U4]** IBC-Go v8.8.0: [localhost stateful](https://github.com/cosmos/ibc-go/blob/v8.8.0/modules/light-clients/09-localhost/client_state.go) e [módulo pinado](https://github.com/cosmos/ibc-go/blob/v8.8.0/go.mod).
- **[U5]** IBC-Go v10.0.0: [changelog da quebra de API](https://github.com/cosmos/ibc-go/blob/v10.0.0/CHANGELOG.md#api-breaking), incluindo ICA unordered por padrão, localhost stateless e remoções de ICS-29/channel upgrade.
- **[U6]** IBC-Go v8.8.0: [módulo ICS-29](https://github.com/cosmos/ibc-go/tree/v8.8.0/modules/apps/29-fee).
- **[U7]** IBC-Go v11.2.0: [ICS-27 Interchain Accounts](https://github.com/cosmos/ibc-go/tree/v11.2.0/modules/apps/27-interchain-accounts).
- **[U8]** Especificação canônica: [ICS-29 marcado deprecated](https://github.com/cosmos/ibc#interchain-standards).
- Linhas mantidas upstream: [README oficial do IBC-Go](https://github.com/cosmos/ibc-go#releases), que lista v10 e v11; v8 permanece nesta política apenas como transição do relayer.
- Fixtures E2E atuais fora da matriz alvo: [Gaia v14.1.0 usa IBC-Go v4](https://github.com/cosmos/gaia/blob/v14.1.0/go.mod), [Osmosis v22.0.0 usa IBC-Go v7](https://github.com/osmosis-labs/osmosis/blob/v22.0.0/go.mod), [Juno v13.0.0 usa IBC-Go v7](https://github.com/CosmosContracts/juno/blob/v13.0.0/go.mod) e [`icad` v0.5.0 usa IBC-Go v7 RC](https://github.com/cosmos/interchain-accounts-demo/blob/v0.5.0/go.mod).

### Repositório

- **[T1]** Classic unit/codec: [`relayer/chains/parsing_test.go`](../relayer/chains/parsing_test.go), [`relayer/protocol/classic/adapter_test.go`](../relayer/protocol/classic/adapter_test.go) e [`relayer/pathEnd_test.go`](../relayer/pathEnd_test.go).
- **[T2]** Conformance events/legacy, unordered Send/Recv/Ack e timeout/refund: [`interchaintest/ibc_test.go`](../interchaintest/ibc_test.go); as chains pinadas são v4/v7, não v8/v10/v11.
- **[T3]** Localhost escrito para imagens v8.0.0/beta: [`interchaintest/localhost_client_test.go`](../interchaintest/localhost_client_test.go); não qualifica a janela v8.8 sem execução verde pinada.
- **[T4]** ICA e close em `icad` v0.5.0/v7 RC: [`interchaintest/interchain_accounts_test.go`](../interchaintest/interchain_accounts_test.go) e [`interchaintest/ica_channel_close_test.go`](../interchaintest/ica_channel_close_test.go).
- **[T5]** ICS-29 legado: [`relayer/codecs/ics29/codec_test.go`](../relayer/codecs/ics29/codec_test.go) e [`interchaintest/fee_middleware_test.go`](../interchaintest/fee_middleware_test.go).
- **[T6]** Localhost: [`relayer/chains/cosmos/module/legacy_localhost_test.go`](../relayer/chains/cosmos/module/legacy_localhost_test.go) e [`relayer/packet_tx_test.go`](../relayer/packet_tx_test.go).
- **[T7]** sr25519: [`cclient/sr25519_test.go`](../cclient/sr25519_test.go), [`relayer/chains/cosmos/keys/sr25519/sr25519_test.go`](../relayer/chains/cosmos/keys/sr25519/sr25519_test.go) e [`relayer/chains/cosmos/keys_test.go`](../relayer/chains/cosmos/keys_test.go).
- **[T8]** Penumbra: [`relayer/chains/penumbra/log_test.go`](../relayer/chains/penumbra/log_test.go).
- **[T9]** Processor/event compatibility: [`relayer/chains/v2_event_ingestion_test.go`](../relayer/chains/v2_event_ingestion_test.go), [`relayer/chains/parsing_test.go`](../relayer/chains/parsing_test.go) e [`relayer/chains/mock/mock_chain_processor_test.go`](../relayer/chains/mock/mock_chain_processor_test.go).
