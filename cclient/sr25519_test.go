package cclient

import (
	"encoding/hex"
	"testing"

	relayersr25519 "github.com/cosmos/relayer/v2/relayer/chains/cosmos/keys/sr25519"
	"github.com/stretchr/testify/require"
)

func TestSR25519PubKeyCompatibility(t *testing.T) {
	publicKeyBytes, err := hex.DecodeString("c059e4478e70444f3f4546826967d1463abba81e3a068d1fec9fa9fa158e0423")
	require.NoError(t, err)
	publicKey := newSR25519PubKey(publicKeyBytes)
	require.Equal(t, "9B3FBE142C74F496E09B6F5A79B2A585E7EB07DE", publicKey.Address().String())

	privateKey := relayersr25519.GenPrivKey()
	message := []byte("Comet client payload")
	signature, err := privateKey.Sign(message)
	require.NoError(t, err)
	relayerPublicKey := newSR25519PubKey(privateKey.PubKey().Bytes())
	require.True(t, relayerPublicKey.VerifySignature(message, signature))
	require.False(t, relayerPublicKey.VerifySignature([]byte("modified"), signature))
}
