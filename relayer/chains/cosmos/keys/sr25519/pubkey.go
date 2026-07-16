package sr25519

import (
	"bytes"
	"fmt"

	"github.com/cometbft/cometbft/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	curvesr25519 "github.com/oasisprotocol/curve25519-voi/primitives/sr25519"
)

const (
	PubKeyName    = "tendermint/PubKeySr25519"
	pubKeySize    = 32
	signatureSize = 64
)

func (key *PubKey) Equals(other cryptotypes.PubKey) bool {
	otherKey, ok := other.(*PubKey)
	return ok && keyEqual(key.Key, otherKey.Key)
}

func (key *PubKey) Address() crypto.Address {
	if len(key.Key) != pubKeySize {
		panic(fmt.Sprintf("sr25519: public key must be %d bytes", pubKeySize))
	}
	return crypto.AddressHash(key.Key)
}

func (key *PubKey) Bytes() []byte { return append([]byte(nil), key.Key...) }

func (key *PubKey) String() string { return fmt.Sprintf("PubKeySr25519{%X}", key.Key) }

func (*PubKey) Type() string { return keyType }

func (key *PubKey) VerifySignature(message, signatureBytes []byte) bool {
	if len(key.Key) != pubKeySize || len(signatureBytes) != signatureSize {
		return false
	}
	var publicKey curvesr25519.PublicKey
	if err := publicKey.UnmarshalBinary(key.Key); err != nil {
		return false
	}
	var signature curvesr25519.Signature
	if err := signature.UnmarshalBinary(signatureBytes); err != nil {
		return false
	}
	return publicKey.Verify(signingContext.NewTranscriptBytes(message), &signature)
}

func keyEqual(left, right []byte) bool { return bytes.Equal(left, right) }
