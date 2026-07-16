package ethermint

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveEthSecp256k1Golden(t *testing.T) {
	const mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	key, err := EthSecp256k1.Derive()(mnemonic, "", "m/44'/60'/0'/0/0")

	require.NoError(t, err)
	require.Equal(t, "1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727", hex.EncodeToString(key))
}

func TestDeriveEthSecp256k1RejectsInvalidInput(t *testing.T) {
	tests := map[string]struct {
		mnemonic string
		path     string
	}{
		"path":     {mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", path: "not-a-path"},
		"mnemonic": {mnemonic: "not a mnemonic", path: "m/44'/60'/0'/0/0"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := EthSecp256k1.Derive()(tc.mnemonic, "", tc.path)
			require.Error(t, err)
		})
	}
}
