package cclient

import (
	"bytes"
	"fmt"

	"github.com/cometbft/cometbft/crypto"
	curvesr25519 "github.com/oasisprotocol/curve25519-voi/primitives/sr25519"
)

const (
	sr25519PubKeySize    = 32
	sr25519SignatureSize = 64
)

var sr25519SigningContext = curvesr25519.NewSigningContext([]byte{})

type sr25519PubKey []byte

func newSR25519PubKey(key []byte) sr25519PubKey {
	return append(sr25519PubKey(nil), key...)
}

func (key sr25519PubKey) Address() crypto.Address {
	if len(key) != sr25519PubKeySize {
		panic(fmt.Sprintf("sr25519: public key must be %d bytes", sr25519PubKeySize))
	}
	return crypto.AddressHash(key)
}

func (key sr25519PubKey) Bytes() []byte { return append([]byte(nil), key...) }

func (key sr25519PubKey) VerifySignature(message, signatureBytes []byte) bool {
	if len(key) != sr25519PubKeySize || len(signatureBytes) != sr25519SignatureSize {
		return false
	}
	var publicKey curvesr25519.PublicKey
	if err := publicKey.UnmarshalBinary(key); err != nil {
		return false
	}
	var signature curvesr25519.Signature
	if err := signature.UnmarshalBinary(signatureBytes); err != nil {
		return false
	}
	return publicKey.Verify(sr25519SigningContext.NewTranscriptBytes(message), &signature)
}

func (key sr25519PubKey) Equals(other crypto.PubKey) bool {
	otherKey, ok := other.(sr25519PubKey)
	return ok && bytes.Equal(key, otherKey)
}

func (sr25519PubKey) Type() string { return "sr25519" }
