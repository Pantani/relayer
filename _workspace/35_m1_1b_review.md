# M1.1b — revisão final focal

Data: 2026-07-15

## Veredito

Nenhum finding acionável P0–P3 permanece nas funções alteradas pelo lote.

## Correções incorporadas durante a revisão

- timeout localhost e offset temporal negativo;
- complexidade de `SendTransferMsg` e fee payer abaixo de 10;
- guards para Tx/AuthInfo/Fee/body/mensagens/signers malformados;
- conversão correta dos bytes de signer para bech32 no SDK 0.54;
- registry/AddressCodec e decode de Tx no provider Penumbra;
- sentinel ABCI v8 exato para counterparty payee vazio;
- persistência protobuf/Amino e armor sr25519;
- comparação de chave privada sr25519 em tempo constante;
- remoção de panics públicos em denom hash e ICS-29 Penumbra;
- verificação Injective compatível com Linux sem CGO.

## Provas focais

- `go test -race -count=1`: 108 testes aprovados em sete pacotes;
- `go vet`: limpo nos mesmos pacotes;
- `git diff --check`: limpo;
- `gocyclo`/`gocognit -over 9`: nenhuma função nova ou com corpo alterado;
- E2E real SDK 0.54/IBC v11 e Penumbra permanece pendente por causa do
  bloqueio documentado do framework de integração.

