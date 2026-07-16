package sr25519

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/gogoproto/proto"
	curvesr25519 "github.com/oasisprotocol/curve25519-voi/primitives/sr25519"
)

const (
	PrivKeySize = 32
	PrivKeyName = "tendermint/PrivKeySr25519"
	keyType     = "sr25519"
)

var signingContext = curvesr25519.NewSigningContext([]byte{})

// PrivKey is a local compatibility implementation for chains that still use
// sr25519 after CometBFT removed the algorithm in v0.39.
type PrivKey struct {
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (key *PrivKey) Reset()         { key.Key = nil }
func (key *PrivKey) String() string { return fmt.Sprintf("PrivKeySr25519{%X}", key.Key) }
func (*PrivKey) ProtoMessage()      {}
func (*PrivKey) XXX_MessageName() string {
	return "cosmos.crypto.sr25519.PrivKey"
}

func init() {
	proto.RegisterType((*PrivKey)(nil), "cosmos.crypto.sr25519.PrivKey")
}

func (key *PrivKey) Bytes() []byte {
	return append([]byte(nil), key.Key...)
}

func (key *PrivKey) Sign(message []byte) ([]byte, error) {
	keyPair, err := key.keyPair()
	if err != nil {
		return nil, err
	}
	signature, err := keyPair.Sign(rand.Reader, signingContext.NewTranscriptBytes(message))
	if err != nil {
		return nil, fmt.Errorf("sr25519: sign message: %w", err)
	}
	encoded, err := signature.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("sr25519: encode signature: %w", err)
	}
	return encoded, nil
}

func (key *PrivKey) PubKey() cryptotypes.PubKey {
	keyPair, err := key.keyPair()
	if err != nil {
		panic(err)
	}
	encoded, err := keyPair.PublicKey().MarshalBinary()
	if err != nil {
		panic(fmt.Errorf("sr25519: encode public key: %w", err))
	}
	return &PubKey{Key: encoded}
}

func (key *PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	otherKey, ok := other.(*PrivKey)
	if !ok {
		return false
	}
	return privateKeyEqual(key.Key, otherKey.Key)
}

func privateKeyEqual(left, right []byte) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare(left, right) == 1
}

func (*PrivKey) Type() string { return keyType }

func GenPrivKey() *PrivKey {
	miniSecret, err := curvesr25519.GenerateMiniSecretKey(rand.Reader)
	if err != nil {
		panic(fmt.Errorf("sr25519: generate private key: %w", err))
	}
	return &PrivKey{Key: append([]byte(nil), miniSecret[:]...)}
}

func newPrivKeyFromSecret(secret []byte) *PrivKey {
	digest := sha256.Sum256(secret)
	return &PrivKey{Key: append([]byte(nil), digest[:]...)}
}

func (key *PrivKey) keyPair() (*curvesr25519.KeyPair, error) {
	if len(key.Key) != PrivKeySize {
		return nil, fmt.Errorf("sr25519: private key must be %d bytes", PrivKeySize)
	}
	miniSecret, err := curvesr25519.NewMiniSecretKeyFromBytes(key.Key)
	if err != nil {
		return nil, fmt.Errorf("sr25519: decode private key: %w", err)
	}
	return miniSecret.ExpandEd25519().KeyPair(), nil
}
