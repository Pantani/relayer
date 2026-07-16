# M0.3 — revisão de código

Data: 2026-07-15  
Ferramenta: CodeRabbit CLI 0.6.5, revisão `uncommitted` autenticada

## Resultado final

**APROVADO: zero findings no segundo passe.** O passe final revisou todos os arquivos não commitados, incluindo o core protocol-neutral, adaptadores, configuração, guardas de runtime, testes e documentação.

O primeiro passe encontrou cinco findings exclusivamente em `interchaintest/feegrant_test.go`. A comparação com o SHA-base confirmou que todos já existiam antes dos lotes M0.2/M0.3:

| finding | classificação | decisão |
|---|---|---|
| `t.Parallel` com collector global | risco real preexistente | hardening FeeGrant separado; não alterar M0.3 |
| mnemonic efêmero impresso | exposição preexistente e documentada | remover em lote de segurança próprio |
| signer assertions validam cardinalidade, não identidade | fraqueza preexistente | comparar conjuntos exatos em lote FeeGrant |
| iteração só por chains observadas | fraqueza preexistente | iterar chains esperadas em lote FeeGrant |
| `gaiaUser` restaurada pela counterparty | provável typo preexistente | corrigir apenas com nova rodada Docker |

Nenhum desses findings foi introduzido pelo diff M0.3. Eles não foram corrigidos aqui porque mudariam uma fatia já validada e exigiriam reexecução dos quatro cenários Docker; permanecem dívida explícita, não aprovação silenciosa.

## Self-review complementar

- `MsgSendPacket` v2 foi modelada como request pré-sequência: não inventa `destination_client` ou `sequence`, que são definidos pelo keeper.
- `Config.validateConfig` ganhou label nil-safe, evitando panic ao reportar path/end malformado.
- O adaptador Classic rejeita eventos desconhecidos e preserva nil/empty e cópias defensivas de data, ack e proof.
- O core neutro não importa Cosmos SDK, ibc-go, provider ou processor; apenas o adaptador Classic importa provider e v8.
- Parser, cache, state machine, proof selection e broadcast Classic não receberam branches v2.

## Comandos

```text
coderabbit review --agent -t uncommitted --dir <repo>  # primeiro passe: 5 findings preexistentes
coderabbit review --agent -t uncommitted --dir <repo>  # passe final: 0 findings
```
