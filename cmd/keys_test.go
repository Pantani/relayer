package cmd_test

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/relayer/v2/cmd"
	"github.com/cosmos/relayer/v2/internal/relayertest"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type concurrentKeysCase struct {
	chainName string
	prefix    string
	coinType  uint32
}

type concurrentKeysResult struct {
	testCase  concurrentKeysCase
	restore   relayertest.RunResult
	show      relayertest.RunResult
	list      relayertest.RunResult
	export    relayertest.RunResult
	deleteKey relayertest.RunResult
	emptyList relayertest.RunResult
}

func runConcurrentKeysWorkflow(sys *relayertest.System, testCase concurrentKeysCase) concurrentKeysResult {
	result := concurrentKeysResult{testCase: testCase}
	coinType := strconv.FormatUint(uint64(testCase.coinType), 10)

	result.restore = sys.Run(zap.NewNop(), "keys", "restore", testCase.chainName, "default", relayertest.ZeroMnemonic, "--coin-type", coinType)
	if result.restore.Err != nil {
		return result
	}
	result.show = sys.Run(zap.NewNop(), "keys", "show", testCase.chainName, "default")
	if result.show.Err != nil {
		return result
	}
	result.list = sys.Run(zap.NewNop(), "keys", "list", testCase.chainName)
	if result.list.Err != nil {
		return result
	}
	result.export = sys.Run(zap.NewNop(), "keys", "export", testCase.chainName, "default")
	if result.export.Err != nil {
		return result
	}
	result.deleteKey = sys.Run(zap.NewNop(), "keys", "delete", testCase.chainName, "default", "-y")
	if result.deleteKey.Err != nil {
		return result
	}
	result.emptyList = sys.Run(zap.NewNop(), "keys", "list", testCase.chainName)
	return result
}

func requireConcurrentKeysResult(t *testing.T, result concurrentKeysResult) string {
	t.Helper()
	require.NoError(t, result.restore.Err)
	require.NoError(t, result.show.Err)
	require.NoError(t, result.list.Err)
	require.NoError(t, result.export.Err)
	require.NoError(t, result.deleteKey.Err)
	require.NoError(t, result.emptyList.Err)

	addressString := strings.TrimSpace(result.restore.Stdout.String())
	require.True(t, strings.HasPrefix(addressString, result.testCase.prefix+"1"))
	require.Empty(t, result.restore.Stderr.String())
	require.Equal(t, addressString+"\n", result.show.Stdout.String())
	require.Empty(t, result.show.Stderr.String())
	require.Equal(t, "key(default) -> "+addressString+"\n", result.list.Stdout.String())
	require.Empty(t, result.list.Stderr.String())
	require.Contains(t, result.export.Stdout.String(), "BEGIN TENDERMINT PRIVATE KEY")
	require.Empty(t, result.export.Stderr.String())
	require.Empty(t, result.deleteKey.Stdout.String())
	require.Equal(t, "key default deleted\n", result.deleteKey.Stderr.String())
	require.Empty(t, result.emptyList.Stdout.String())
	require.Contains(t, result.emptyList.Stderr.String(), "no keys found for chain "+result.testCase.chainName)

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	kr := keyring.NewInMemory(codec.NewProtoCodec(registry))
	require.NoError(t, kr.ImportPrivKey("imported", result.export.Stdout.String(), keys.DefaultKeyPass))
	info, err := kr.Key("imported")
	require.NoError(t, err)
	importedAddress, err := info.GetAddress()
	require.NoError(t, err)
	encodedAddress, err := address.NewBech32Codec(result.testCase.prefix).BytesToString(importedAddress)
	require.NoError(t, err)
	require.Equal(t, addressString, encodedAddress)

	return hex.EncodeToString(importedAddress)
}

func TestKeysConcurrentPrefixes(t *testing.T) {
	t.Parallel()

	testCases := []concurrentKeysCase{
		{chainName: "cosmosChain", prefix: "cosmos", coinType: 118},
		{chainName: "osmoChain", prefix: "osmo", coinType: 330},
	}
	type concurrentKeysRun struct {
		testCase concurrentKeysCase
		sys      *relayertest.System
	}
	runs := make([]concurrentKeysRun, 0, len(testCases))

	for _, testCase := range testCases {
		sys := relayertest.NewSystem(t)
		_ = sys.MustRun(t, "config", "init")
		slip44 := 118
		sys.MustAddChain(t, testCase.chainName, cmd.ProviderConfigWrapper{
			Type: "cosmos",
			Value: cosmos.CosmosProviderConfig{
				AccountPrefix:  testCase.prefix,
				ChainID:        "test-" + testCase.chainName,
				KeyringBackend: "test",
				Timeout:        "10s",
				Slip44:         &slip44,
			},
		})
		runs = append(runs, concurrentKeysRun{testCase: testCase, sys: sys})
	}

	start := make(chan struct{})
	results := make(chan concurrentKeysResult, len(runs))
	for _, run := range runs {
		go func() {
			<-start
			results <- runConcurrentKeysWorkflow(run.sys, run.testCase)
		}()
	}
	close(start)

	derivedAddresses := make([]string, 0, len(runs))
	for range runs {
		derivedAddresses = append(derivedAddresses, requireConcurrentKeysResult(t, <-results))
	}
	require.NotEqual(t, derivedAddresses[0], derivedAddresses[1])
}

func TestKeysList_Empty(t *testing.T) {
	t.Parallel()

	sys := relayertest.NewSystem(t)

	_ = sys.MustRun(t, "config", "init")

	sys.MustAddChain(t, "testChain", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			ChainID:        "testcosmos",
			KeyringBackend: "test",
			Timeout:        "10s",
		},
	})

	res := sys.MustRun(t, "keys", "list", "testChain")
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "no keys found for chain testChain")
}

func TestKeysRestore_Delete(t *testing.T) {
	t.Parallel()

	sys := relayertest.NewSystem(t)

	_ = sys.MustRun(t, "config", "init")

	slip44 := 118

	sys.MustAddChain(t, "testChain", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos",
			KeyringBackend: "test",
			Timeout:        "10s",
			Slip44:         &slip44,
		},
	})

	// Restore a key with mnemonic to the chain.
	res := sys.MustRun(t, "keys", "restore", "testChain", "default", relayertest.ZeroMnemonic)
	require.Equal(t, res.Stdout.String(), relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Restored key must show up in list.
	res = sys.MustRun(t, "keys", "list", "testChain")
	require.Equal(t, res.Stdout.String(), "key(default) -> "+relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Deleting the key must succeed.
	res = sys.MustRun(t, "keys", "delete", "testChain", "default", "-y")
	require.Empty(t, res.Stdout.String())
	require.Equal(t, res.Stderr.String(), "key default deleted\n")

	// Listing the keys again gives the no keys warning.
	res = sys.MustRun(t, "keys", "list", "testChain")
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "no keys found for chain testChain")
}

func TestKeysExport(t *testing.T) {
	t.Parallel()

	sys := relayertest.NewSystem(t)

	_ = sys.MustRun(t, "config", "init")

	slip44 := 118

	sys.MustAddChain(t, "testChain", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos",
			KeyringBackend: "test",
			Timeout:        "10s",
			Slip44:         &slip44,
		},
	})

	// Restore a key with mnemonic to the chain.
	res := sys.MustRun(t, "keys", "restore", "testChain", "default", relayertest.ZeroMnemonic)
	require.Equal(t, res.Stdout.String(), relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Export the key.
	res = sys.MustRun(t, "keys", "export", "testChain", "default")
	armorOut := res.Stdout.String()
	require.Contains(t, armorOut, "BEGIN TENDERMINT PRIVATE KEY")
	require.Empty(t, res.Stderr.String())

	// Import the key to a temporary keyring.
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kr := keyring.NewInMemory(cdc)
	require.NoError(t, kr.ImportPrivKey("temp", armorOut, keys.DefaultKeyPass))

	// Retrieve the imported key's address and confirm it matches the expected address
	info, err := kr.Key("temp")
	require.NoError(t, err, "failed to retrieve imported key")
	addr, err := info.GetAddress()
	require.NoError(t, err, "failed to get address from imported key")
	require.Equal(t, relayertest.ZeroCosmosAddr, addr.String(), "imported address does not match expected address")
}

func TestKeysDefaultCoinType(t *testing.T) {
	t.Parallel()

	sys := relayertest.NewSystem(t)

	_ = sys.MustRun(t, "config", "init")

	slip44 := 118

	sys.MustAddChain(t, "testChain", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos-1",
			KeyringBackend: "test",
			Timeout:        "10s",
			Slip44:         &slip44,
		},
	})

	sys.MustAddChain(t, "testChain2", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos-2",
			KeyringBackend: "test",
			Timeout:        "10s",
		},
	})

	// Restore a key with mnemonic to the chain.
	res := sys.MustRun(t, "keys", "restore", "testChain", "default", relayertest.ZeroMnemonic)
	require.Equal(t, res.Stdout.String(), relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Restore a key with mnemonic to the chain.
	res = sys.MustRun(t, "keys", "restore", "testChain2", "default", relayertest.ZeroMnemonic)
	require.Equal(t, res.Stdout.String(), relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Export the key.
	res = sys.MustRun(t, "keys", "export", "testChain", "default")
	armorOut := res.Stdout.String()
	require.Contains(t, armorOut, "BEGIN TENDERMINT PRIVATE KEY")
	require.Empty(t, res.Stderr.String())

	// Export the key.
	res = sys.MustRun(t, "keys", "export", "testChain2", "default")
	armorOut2 := res.Stdout.String()
	require.Contains(t, armorOut, "BEGIN TENDERMINT PRIVATE KEY")
	require.Empty(t, res.Stderr.String())

	// Import the key to a temporary keyring.
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kr := keyring.NewInMemory(cdc)
	require.NoError(t, kr.ImportPrivKey("temp", armorOut, keys.DefaultKeyPass))

	// This should fail due to same key
	err := kr.ImportPrivKey("temp", armorOut2, keys.DefaultKeyPass)
	require.Error(t, err, "same key was able to be imported twice")
	require.Contains(t, err.Error(), "cannot overwrite key")
}

func TestKeysRestoreAll_Delete(t *testing.T) {
	t.Parallel()

	sys := relayertest.NewSystem(t)

	_ = sys.MustRun(t, "config", "init")

	slip44 := 118

	sys.MustAddChain(t, "testChain", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos",
			KeyringBackend: "test",
			Timeout:        "10s",
			Slip44:         &slip44,
		},
	})
	sys.MustAddChain(t, "testChain2", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos-2",
			KeyringBackend: "test",
			Timeout:        "10s",
			Slip44:         &slip44,
		},
	})
	sys.MustAddChain(t, "testChain3", cmd.ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			AccountPrefix:  "cosmos",
			ChainID:        "testcosmos-3",
			KeyringBackend: "test",
			Timeout:        "10s",
			Slip44:         &slip44,
		},
	})

	// Restore keys for all configured chains with a single mnemonic.
	res := sys.MustRun(t, "keys", "restore", "default", relayertest.ZeroMnemonic, "--restore-all")
	require.Equal(t, res.Stdout.String(), relayertest.ZeroCosmosAddr+"\n"+relayertest.ZeroCosmosAddr+"\n"+relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Restored key must show up in list.
	res = sys.MustRun(t, "keys", "list", "testChain")
	require.Equal(t, res.Stdout.String(), "key(default) -> "+relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	res = sys.MustRun(t, "keys", "list", "testChain2")
	require.Equal(t, res.Stdout.String(), "key(default) -> "+relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	res = sys.MustRun(t, "keys", "list", "testChain3")
	require.Equal(t, res.Stdout.String(), "key(default) -> "+relayertest.ZeroCosmosAddr+"\n")
	require.Empty(t, res.Stderr.String())

	// Deleting the key must succeed.
	res = sys.MustRun(t, "keys", "delete", "testChain", "default", "-y")
	require.Empty(t, res.Stdout.String())
	require.Equal(t, res.Stderr.String(), "key default deleted\n")

	res = sys.MustRun(t, "keys", "delete", "testChain2", "default", "-y")
	require.Empty(t, res.Stdout.String())
	require.Equal(t, res.Stderr.String(), "key default deleted\n")

	res = sys.MustRun(t, "keys", "delete", "testChain3", "default", "-y")
	require.Empty(t, res.Stdout.String())
	require.Equal(t, res.Stderr.String(), "key default deleted\n")

	// Listing the keys again gives the no keys warning.
	res = sys.MustRun(t, "keys", "list", "testChain")
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "no keys found for chain testChain")

	res = sys.MustRun(t, "keys", "list", "testChain2")
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "no keys found for chain testChain2")

	res = sys.MustRun(t, "keys", "list", "testChain3")
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "no keys found for chain testChain3")
}
