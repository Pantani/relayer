# M0.3 — desenho de configuração Classic/v2

Data: 2026-07-15  
Escopo: desenho somente; nenhum código de produção alterado por esta análise.

## Decisão

O menor schema retrocompatível é um discriminador por `Path`, mantendo os campos Classic existentes e acrescentando o prefixo Merkle em cada ponta:

```go
// relayer/path.go
type Path struct {
	Protocol protocol.Protocol `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	Src      *PathEnd           `yaml:"src" json:"src"`
	Dst      *PathEnd           `yaml:"dst" json:"dst"`
	Filter   ChannelFilter      `yaml:"src-channel-filter" json:"src-channel-filter"`
}

// relayer/pathEnd.go
type MerklePrefix []string

type PathEnd struct {
	ChainID      string       `yaml:"chain-id,omitempty" json:"chain-id,omitempty"`
	ClientID     string       `yaml:"client-id,omitempty" json:"client-id,omitempty"`
	ConnectionID string       `yaml:"connection-id,omitempty" json:"connection-id,omitempty"`
	MerklePrefix MerklePrefix `yaml:"merkle-prefix,omitempty" json:"merkle-prefix,omitempty"`
}
```

`MerklePrefix` representa, em ordem, os segmentos UTF-8 do prefixo de compromisso da **chain da própria ponta**. Assim, o registro no client de `src` deve apontar para `dst.ClientID` e usar `dst.MerklePrefix`, e vice-versa. A conversão para o contrato `[][]byte` do client/v2 é explícita (`MerklePrefix.Bytes()`), sem colocar tipos de `ibc-go/v11` no schema enquanto o módulo continua pinado em v8. O escopo M0.3/Cosmos usa prefixos UTF-8; suporte futuro a prefixo binário deve ganhar codificação explícita, nunca conversão implícita.

O pacote protocol-neutral deve expor exatamente estes valores:

```go
const (
	protocol.Unspecified protocol.Protocol = ""
	protocol.Classic     protocol.Protocol = "classic"
	protocol.V2          protocol.Protocol = "v2"
)
```

Não adicionar `schema-version` global neste lote: isso reescreveria todos os arquivos sem existir ainda um migrador global. O discriminador `path.protocol` versiona somente a parte que de fato ganhou duas semânticas.

## Default e estabilidade

- Campo ausente significa Classic por compatibilidade, mas o valor armazenado continua `""`; não preencher `classic` durante unmarshal.
- `(*Path).EffectiveProtocol() protocol.Protocol` retorna `protocol.Classic` para `Unspecified` e o valor explícito nos demais casos.
- Aceitar somente `"classic"` e `"v2"` em minúsculas. Não aparar, mudar caixa nem converter valor desconhecido.
- `omitempty` impede que um path legado passe a emitir `protocol:` ou `merkle-prefix:`. O YAML legado continua com o mesmo shape canônico produzido hoje.
- `protocol: classic` explícito permanece explícito após round-trip; não colapsar para campo ausente.
- Manter a tag atual, sem `omitempty`, de `src-channel-filter`. Alterá-la agora mudaria a saída legada. Em v2 o objeto vazio pode continuar serializado, mas qualquer `rule` ou item em `channel-list` é proibido. Essa presença vazia não é uma capacidade Classic ativa.

## Validação estrutural agora (M0.3)

Adicionar métodos pequenos, todos com ciclomatica e cognitiva menores que 10:

```go
func (p *Path) EffectiveProtocol() protocol.Protocol
func (p *Path) Validate() error
func (p *Path) validateClassic() error
func (p *Path) validateV2() error
func (pe *PathEnd) ValidateMerklePrefix() error
func (p MerklePrefix) Bytes() [][]byte
func (cf ChannelFilter) Empty() bool
```

`Path.Validate()` é offline e deve rodar antes de qualquer acesso a `Src`/`Dst` ou RPC:

1. rejeitar `Path` nulo e `Src`/`Dst` nulos com erro de configuração, sem panic;
2. validar o discriminador e preservar `ValidateChannelFilterRule()` para Classic;
3. Classic (inclusive ausente): permitir IDs de conexão vazios ou preenchidos e filtro atual; rejeitar `merkle-prefix` em qualquer ponta para não ignorar silenciosamente configuração v2;
4. v2: exigir `chain-id`, `client-id` e `merkle-prefix` não vazios nas duas pontas;
5. v2: rejeitar `connection-id` em qualquer ponta;
6. v2: exigir `Filter.Empty()` (`Rule == "" && len(ChannelList) == 0`);
7. rejeitar segmento vazio em `merkle-prefix`; preservar ordem e duplicatas, pois o valor é um key path, não um conjunto.

Mensagens devem nomear protocolo, direção e campo. Exemplos estáveis:

```text
path protocol "v3" is not supported
path protocol v2 requires source client-id
path protocol v2 requires destination merkle-prefix
path protocol v2 cannot set source connection-id
path protocol v2 cannot set src-channel-filter
path protocol classic cannot set source merkle-prefix
source merkle-prefix contains an empty segment at index 1
```

Integração mínima:

- `Config.validateConfig()` troca a chamada isolada de `ValidateChannelFilterRule()` por `Path.Validate()` e conserva o wrapper atual `error initializing the relayer config for path ...`.
- `Config.ValidatePath(...)` chama `p.Validate()` primeiro e somente então `ValidatePathEnd` para cada ponta.
- `PathEnd.ValidateFull()` pode validar a forma do prefixo, mas não decide se ele é permitido: a exclusão Classic/v2 depende do `Path` completo.

## Validação online e limite de prontidão

M0.3 valida somente o que o provider v8 já sabe consultar:

- Classic mantém a validação atual de client e, quando presente, connection/client correspondente.
- v2 consulta a existência dos dois clients por `QueryClientStateResponse`; como `connection-id` é proibido, nunca chama `QueryConnection`.
- Chain ausente continua sendo warning, como hoje, para não quebrar importação de paths antes de cadastrar chains.

Isso **não** prova que o relay v2 está pronto. A validação de prontidão fica para M1, quando o provider tiver `CounterpartyInfo` e capabilities. Ela deve verificar bilateralmente client contraparte e bytes de prefixo, allowlist/signer e endpoints/proofs. Até lá, start/flush/status/handshakes Classic precisam rejeitar path v2 com erro explícito de capability; não podem cair em queries com connection vazia.

## Conflitos e atualização

`Config.AddPath` hoje permite preencher campo antigo vazio, mas proíbe substituir campo antigo preenchido. Preservar essa regra e acrescentar:

```go
func checkPathProtocolConflict(pathID string, oldPath, newPath *relayer.Path) error
func checkMerklePrefixConflict(pathID, direction string, old, new relayer.MerklePrefix) error
```

- Comparar `EffectiveProtocol()`: `classic -> v2` e `v2 -> classic` são conflito mesmo se o Classic antigo usava campo ausente. Troca de protocolo exige fluxo de migração explícito futuro; `AddPath` nunca converte.
- Ausente Classic e `protocol: classic` são semanticamente compatíveis.
- Se o path antigo tinha `protocol: classic` explícito e o novo usa campo ausente, manter o valor explícito ao copiar; não apagar intenção textual.
- Prefixo antigo vazio pode ser preenchido; prefixo preenchido só aceita valor idêntico (`slices.Equal`). Alteração ou remoção é conflito, igual aos IDs atuais.
- Executar conflito de protocolo antes dos conflitos de ponta para produzir o erro causal correto.
- Fazer cópia rasa do novo `Path` para preservar protocolo explícito; não mutar o argumento do chamador.

Não adicionar `--protocol` a `paths update` em M0.3. O comando atual também não consegue limpar connection IDs, e limpar automaticamente seria uma migração silenciosa. Um comando futuro deve exigir flags explícitas de limpeza e mostrar o diff antes de trocar o protocolo.

## Consumidores que precisam de guarda

O schema sozinho torna estes fluxos perigosos para v2, pois assumem connection/channel: `Path.QueryPathStatus`, `Config.ChainsFromPath`, `StartRelayer`/`relayerStartLegacy`, `flush`, transfer com descoberta de channel, create/link connection/channel, `paths fetch` do chain-registry e `updatePathConfig` dos handshakes. M0.3 deve manter o dispatch Classic por default e retornar `v2 not operational in this build` antes dessas operações; não fabricar connection/channel a partir do client pair.

## Testes de aceitação

### `relayer/path_protocol_test.go`

1. `TestPathEffectiveProtocol`: campo ausente e `classic` explícito resultam Classic; `v2` resulta v2.
2. `TestLegacyPathYAMLRoundTrip`: fixture canônica atual faz unmarshal/marshal sem `protocol` ou `merkle-prefix` e mantém `src-channel-filter`.
3. `TestExplicitProtocolRoundTrip`: YAML e JSON preservam `protocol: classic`; v2 preserva protocolo e os prefixos ordenados.
4. `TestPathValidate`: tabela com unknown protocol, nil ends, Classic válido, Classic+prefix, v2 válido, client/prefix ausente por direção, connection por direção, regra/lista de channel e segmento vazio.
5. `TestMerklePrefixBytes`: preserva ordem, duplicatas e bytes UTF-8.
6. `TestGenPathRemainsImplicitClassic`: `GenPath` não passa a emitir o novo campo.

### `cmd/config_path_validation_test.go`

1. `TestValidatePathRejectsStructureBeforeRPC`: combinação v2 inválida produz zero queries.
2. `TestValidateV2PathQueriesClientsOnly`: duas heights/client queries e zero connection queries.
3. Regressões existentes de Classic permanecem com mesmas contagens e mensagens.
4. `TestValidateConfigRejectsInvalidProtocolCombination`: load offline falha mesmo sem chains configuradas.

### `cmd/config_path_conflict_test.go`

1. implicit Classic versus v2 e v2 versus Classic falham;
2. implicit/explicit Classic são compatíveis e preservam o explícito antigo;
3. prefixo vazio pode ser preenchido em v2;
4. prefixo preenchido diferente ou removido falha;
5. re-add idêntico de v2 passa.

## Ordem de implementação

1. Aguardar o tipo `protocol.Protocol` do modelo neutro.
2. Implementar tipos/métodos no pacote `relayer` e testes de round-trip/validação.
3. Integrar validação em `cmd/config.go` e conflitos/testes.
4. Adicionar as guardas de operação no boundary/adapter, sem habilitar runtime v2.
5. Rodar testes focados, `go test ./relayer ./cmd`, build, lint e métricas estritas das funções tocadas.

