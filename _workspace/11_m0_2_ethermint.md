# M0.2 — Ethermint codecs

## Scope and contract

- Audited base SHA: `bef2e868f157659b403fe1303ee121fb69fec9e6`.
- Exclusive scope: `relayer/codecs/ethermint/**`.
- Strict limit: every handwritten function must score below 10 in both cyclomatic and cognitive complexity.
- Pinned tools: `gocyclo v0.6.0` and `gocognit v1.2.1`.
- Canonically generated files excluded: `account.pb.go`, `keys.pb.go`, and `web3.pb.go`.

## Complexity before and after

Scores are keyed by `file:function`, not by mutable line number.

| Function | Before cyclomatic | Before cognitive | After cyclomatic | After cognitive |
|---|---:|---:|---:|---:|
| `encoding.go:traverseFields` | 33 | 82 | 4 | 4 |
| `encoding.go:typToEth` | 31 | 13 | 5 | 2 |
| `eip712.go:decodeProtobufSignDoc` | 16 | 15 | 6 | 5 |
| `eip712.go:decodeAminoSignDoc` | 11 | 11 | 8 | 7 |
| `eip712.go:validatePayloadMessages` | 10 | 16 | 5 | 5 |
| `algorithm.go:(ethSecp256k1Algo).Derive` | 7 | 13 | 1 | 0 |
| `algorithm.go:deriveEthSecp256k1` | — | — | 6 | 5 |

After the refactor, the maximum in the complete handwritten Ethermint scope is 8 cyclomatic and 7 cognitive. No function or test reaches 10.

## Refactoring

- Split reflection traversal into field preparation, collection extraction, pointer/interface dereferencing, Ethereum type classification, type-map insertion, and recursive traversal.
- Replaced the primitive type switch and long convertibility expressions with explicit classification tables and small conversion helpers.
- Split Amino and Protobuf EIP-712 decoding into decode, envelope validation, message unpacking, signer extraction, payload validation, and typed-data construction.
- Split HD derivation into the input/seed/private-key pipeline and path traversal after the full-scope audit found its inherited cognitive score of 13.
- Preserved field ordering, zero-value omission, first-element collection inference, `Any` unpacking order, EIP-712 output, and existing error strings.

## Characterization tests

- `encoding_test.go`
  - golden type map for primitives, collections, nested structs, zero values, and unsupported maps;
  - pointer-to-struct traversal;
  - existing early-return behavior for an already complete type definition;
  - table for every supported primitive, array/slice, special Cosmos/Ethereum type, and unsupported type.
- `eip712_test.go`
  - golden Protobuf sign-doc to EIP-712 message/domain conversion;
  - table for same/different message types, same/different signers, and multiple signers;
  - exact dual Amino/Protobuf codec-initialization error;
  - payload completeness table.
- `algorithm_test.go`
  - golden Ethereum BIP-44 private key for the canonical BIP-39 mnemonic;
  - invalid mnemonic/path table.

## Verification

Passed:

```text
GOTOOLCHAIN=go1.25.9 go test -mod=readonly ./relayer/codecs/ethermint
GOTOOLCHAIN=go1.25.9 go test -mod=readonly -race ./relayer/codecs/ethermint
GOTOOLCHAIN=go1.25.9 go build -mod=readonly ./relayer/codecs/ethermint
make lint
go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 <all handwritten Ethermint files>
go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -test <all handwritten Ethermint files>
```

The race build emitted a macOS linker `LC_DYSYMTAB` warning, but the test completed successfully.
CodeRabbit CLI `0.6.5` was authenticated, but the final uncommitted review was not executed because the public-repository quota was rate-limited for 29 minutes.

## Preserved edge behavior and limitations

- `common.Hash` and `common.Address` remain `uint8[]`, because reflection classifies their underlying definitions as arrays before the special struct conversion branch. Changing this would alter EIP-712 output and was intentionally left outside a behavior-preserving complexity lot.
- Collection type inference still inspects only the first item and still omits empty collections.
- Codec state remains package-global. Tests install and restore isolated Cosmos SDK codecs sequentially.
- The error text refers to initialization through `SetEncodingConfig`, but no initializer assigning `protoCodec` and `aminoCodec` exists in this repository. That is a pre-existing functional integration gap and needs a separate API/lifecycle decision.
- No Docker or live Ethermint chain scenario was run in this slice.
