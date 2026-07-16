package sr25519

import (
	"encoding/hex"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestCometBFT038CompatibilityVector(t *testing.T) {
	key := newPrivKeyFromSecret([]byte("relayer-sr25519-golden"))
	require.Equal(t, "45da45b9e16e307d10770d43e21184bac966257a5a339160ba6ce51b9c9678fc", hex.EncodeToString(key.Bytes()))

	publicKey := key.PubKey().(*PubKey)
	require.Equal(t, "c059e4478e70444f3f4546826967d1463abba81e3a068d1fec9fa9fa158e0423", hex.EncodeToString(publicKey.Bytes()))
	require.Equal(t, "9B3FBE142C74F496E09B6F5A79B2A585E7EB07DE", publicKey.Address().String())

	wire, err := proto.Marshal(publicKey)
	require.NoError(t, err)
	require.Equal(t, "0a20c059e4478e70444f3f4546826967d1463abba81e3a068d1fec9fa9fa158e0423", hex.EncodeToString(wire))
}

func TestSignAndVerify(t *testing.T) {
	key := newPrivKeyFromSecret([]byte("relayer-sr25519-signing"))
	message := []byte("IBC relayer payload")
	signature, err := key.Sign(message)
	require.NoError(t, err)
	require.True(t, key.PubKey().VerifySignature(message, signature))
	require.False(t, key.PubKey().VerifySignature([]byte("modified"), signature))
	require.False(t, key.PubKey().VerifySignature(message, signature[:len(signature)-1]))
}

func TestInvalidPrivateKeyReturnsError(t *testing.T) {
	key := &PrivKey{Key: []byte("short")}
	_, err := key.Sign([]byte("message"))
	require.ErrorContains(t, err, "private key must be 32 bytes")
}

func TestPrivateKeyEquality(t *testing.T) {
	key := newPrivKeyFromSecret([]byte("relayer-sr25519-equality"))
	require.True(t, key.Equals(&PrivKey{Key: key.Bytes()}))
	require.False(t, key.Equals(&PrivKey{Key: append(key.Bytes(), 0)}))
	require.False(t, key.Equals(&PrivKey{Key: []byte("short")}))
}

func TestAnyRoundTrip(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)

	privateKey := newPrivKeyFromSecret([]byte("persisted-sr25519-key"))
	privateAny, err := codectypes.NewAnyWithValue(privateKey)
	require.NoError(t, err)
	require.Equal(t, "/cosmos.crypto.sr25519.PrivKey", privateAny.TypeUrl)
	var decodedPrivateKey cryptotypes.PrivKey
	require.NoError(t, registry.UnpackAny(privateAny, &decodedPrivateKey))
	require.True(t, privateKey.Equals(decodedPrivateKey))

	publicKey := privateKey.PubKey()
	publicAny, err := codectypes.NewAnyWithValue(publicKey)
	require.NoError(t, err)
	require.Equal(t, "/cosmos.crypto.sr25519.PubKey", publicAny.TypeUrl)
	var decodedPublicKey cryptotypes.PubKey
	require.NoError(t, registry.UnpackAny(publicAny, &decodedPublicKey))
	require.True(t, publicKey.Equals(decodedPublicKey))
}
