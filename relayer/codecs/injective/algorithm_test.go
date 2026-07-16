package injective

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestEthSecp256k1DeriveGolden(t *testing.T) {
	key, err := EthSecp256k1.Derive()(testMnemonic, "", "m/44'/60'/0'/0/0")

	require.NoError(t, err)
	require.Equal(t, "1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727", hex.EncodeToString(key))
}

func TestEthSecp256k1DeriveRejectsInvalidInput(t *testing.T) {
	tests := map[string]struct {
		mnemonic string
		path     string
	}{
		"path":     {mnemonic: testMnemonic, path: "not-a-path"},
		"mnemonic": {mnemonic: "not a mnemonic", path: "m/44'/60'/0'/0/0"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			key, err := EthSecp256k1.Derive()(tc.mnemonic, "", tc.path)

			require.Error(t, err)
			require.Nil(t, key)
		})
	}
}

func TestEthSecp256k1DeriveValidatesPathBeforeMnemonic(t *testing.T) {
	_, pathErr := EthSecp256k1.Derive()(testMnemonic, "", "not-a-path")
	key, err := EthSecp256k1.Derive()("not a mnemonic", "", "not-a-path")

	require.EqualError(t, err, pathErr.Error())
	require.Nil(t, key)
}
