package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCliChainsCharacterizesCobraStructureAndDefaults(t *testing.T) {
	state := newCharacterizedConfigState(t)
	state.config = DefaultConfig("")

	list := chainsListCmd(state)
	require.Equal(t, "list", list.Use)
	require.Equal(t, []string{"l"}, list.Aliases)
	require.Equal(t, "Returns chain configuration data", list.Short)
	require.False(t, mustBoolFlag(t, list, flagJSON))
	require.False(t, mustBoolFlag(t, list, flagYAML))

	registryList := chainsRegistryList(state)
	require.Equal(t, "registry-list", registryList.Use)
	require.Equal(t, []string{"rl"}, registryList.Aliases)
	require.Equal(t, "List chains available for configuration from the registry", registryList.Short)
	require.False(t, mustBoolFlag(t, registryList, flagJSON))
	require.False(t, mustBoolFlag(t, registryList, flagYAML))

	show := chainsShowCmd(state)
	require.Equal(t, "show chain_name", show.Use)
	require.Equal(t, []string{"s"}, show.Aliases)
	require.Equal(t, "Returns a chain's configuration data", show.Short)
	require.False(t, mustBoolFlag(t, show, flagJSON))

	add := chainsAddCmd(state)
	require.Equal(t, "add [chain-name...]", add.Use)
	require.Equal(t, []string{"a"}, add.Aliases)
	require.Empty(t, mustStringFlag(t, add, flagFile))
	require.Empty(t, mustStringFlag(t, add, flagURL))
	require.False(t, mustBoolFlag(t, add, flagForceAdd))
	require.False(t, mustBoolFlag(t, add, flagTestnet))

	require.NoError(t, list.Args(list, nil))
	require.Error(t, list.Args(list, []string{"extra"}))
	require.NoError(t, registryList.Args(registryList, nil))
	require.Error(t, registryList.Args(registryList, []string{"extra"}))
	require.NoError(t, show.Args(show, []string{"alpha"}))
	require.Error(t, show.Args(show, nil))
	require.Error(t, show.Args(show, []string{"alpha", "beta"}))
	require.NoError(t, add.Args(add, nil))
	require.NoError(t, add.Args(add, []string{"alpha", "beta"}))
}

func TestCliChainsCharacterizesFlagGetterOrderAndPreconditions(t *testing.T) {
	state, _ := characterizedChainsStateWithCosmos(t, "alpha", "alpha-1")

	t.Run("list reads json then yaml and preserves empty output", func(t *testing.T) {
		target := chainsListCmd(state)
		cmd := &cobra.Command{Use: "list"}
		var stdout, stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		err := target.RunE(cmd, nil)
		require.EqualError(t, err, "flag accessed but not defined: json")
		require.Empty(t, stdout.String())
		require.Empty(t, stderr.String())

		cmd.Flags().Bool(flagJSON, false, "")
		err = target.RunE(cmd, nil)
		require.EqualError(t, err, "flag accessed but not defined: yaml")
		require.Empty(t, stdout.String())
		require.Empty(t, stderr.String())
	})

	t.Run("registry-list reads both flags before network", func(t *testing.T) {
		var requests int
		withCharacterizedChainsTransport(t, func(*http.Request) (*http.Response, error) {
			requests++
			return nil, errors.New("network must not be reached")
		})

		target := chainsRegistryList(state)
		cmd := &cobra.Command{Use: "registry-list"}
		err := target.RunE(cmd, nil)
		require.EqualError(t, err, "flag accessed but not defined: json")

		cmd.Flags().Bool(flagJSON, false, "")
		err = target.RunE(cmd, nil)
		require.EqualError(t, err, "flag accessed but not defined: yaml")
		require.Zero(t, requests)
	})

	t.Run("show checks chain before json and then reads json", func(t *testing.T) {
		target := chainsShowCmd(state)
		cmd := &cobra.Command{Use: "show"}

		err := target.RunE(cmd, []string{"missing"})
		require.EqualError(t, err, "chain with name \"missing\" not found in config. consider running `rly chains add missing`")

		err = target.RunE(cmd, []string{"alpha"})
		require.EqualError(t, err, "flag accessed but not defined: json")
	})

	t.Run("add reads input flags before config initialization", func(t *testing.T) {
		nilState := &appState{log: zap.NewNop(), viper: state.viper}
		target := chainsAddCmd(nilState)
		cmd := &cobra.Command{Use: "add"}

		err := target.RunE(cmd, nil)
		require.EqualError(t, err, "flag accessed but not defined: file")

		cmd = characterizedAddInputsCommand(true, true, true, true)
		require.NoError(t, cmd.Flags().Set(flagFile, ""))
		require.NoError(t, cmd.Flags().Set(flagURL, ""))
		err = target.RunE(cmd, nil)
		require.EqualError(t, err, "config not initialized, consider running `rly config init`")

		require.NoError(t, cmd.Flags().Set(flagFile, "chain.json"))
		require.NoError(t, cmd.Flags().Set(flagURL, "https://example.test/chain.json"))
		err = target.RunE(cmd, nil)
		require.ErrorIs(t, err, errMultipleAddFlags)
	})
}

func TestCliChainsCharacterizesListAndShowFormatting(t *testing.T) {
	t.Run("empty list warns before rejecting conflicting formats", func(t *testing.T) {
		state := newCharacterizedConfigState(t)
		state.config = DefaultConfig("")
		cmd := chainsListCmd(state)
		require.NoError(t, cmd.Flags().Set(flagJSON, "true"))
		require.NoError(t, cmd.Flags().Set(flagYAML, "true"))

		stdout, stderr, err := executeCharacterizedChainsCommand(t, cmd)

		require.EqualError(t, err, "can't pass both --json and --yaml, must pick one")
		require.Empty(t, stdout)
		require.Equal(t, "warning: no chains found (do you need to run 'rly chains add'?)\n", stderr)
	})

	t.Run("plain list uses fixed columns and path marker", func(t *testing.T) {
		state, _ := characterizedChainsStateWithCosmos(t, "alpha", "alpha-1")
		state.config.Paths["demo"] = &relayer.Path{
			Protocol: protocol.ProtocolClassic,
			Src:      &relayer.PathEnd{ChainID: "alpha-1"},
			Dst:      &relayer.PathEnd{ChainID: "beta-1"},
		}

		stdout, stderr, err := executeCharacterizedChainsCommand(t, chainsListCmd(state))

		require.NoError(t, err)
		require.Equal(t, " 1: alpha-1              -> type(cosmos) key(✘) bal(✘) path(✔)\n", stdout)
		require.Empty(t, stderr)
	})

	t.Run("show json and yaml retain exact envelope and newlines", func(t *testing.T) {
		state, _ := characterizedChainsStateWithCosmos(t, "alpha", "alpha-1")

		jsonCmd := chainsShowCmd(state)
		require.NoError(t, jsonCmd.Flags().Set(flagJSON, "true"))
		jsonOut, jsonErrOut, err := executeCharacterizedChainsCommand(t, jsonCmd, "alpha")
		require.NoError(t, err)
		require.Empty(t, jsonErrOut)
		require.Equal(t, characterizedChainsShowJSON, strings.ReplaceAll(jsonOut, state.homePath, "<HOME>"))

		yamlOut, yamlErrOut, err := executeCharacterizedChainsCommand(t, chainsShowCmd(state), "alpha")
		require.NoError(t, err)
		require.Empty(t, yamlErrOut)
		require.Equal(t, characterizedChainsShowYAML, strings.ReplaceAll(yamlOut, state.homePath, "<HOME>"))
	})

	t.Run("list json and yaml retain map envelope and newlines", func(t *testing.T) {
		state, _ := characterizedChainsStateWithCosmos(t, "alpha", "alpha-1")

		jsonCmd := chainsListCmd(state)
		require.NoError(t, jsonCmd.Flags().Set(flagJSON, "true"))
		jsonOut, jsonErrOut, err := executeCharacterizedChainsCommand(t, jsonCmd)
		require.NoError(t, err)
		require.Empty(t, jsonErrOut)
		wantJSON := "{\"alpha\":" + strings.TrimSuffix(characterizedChainsShowJSON, "\n") + "}\n"
		require.Equal(t, wantJSON, strings.ReplaceAll(jsonOut, state.homePath, "<HOME>"))

		yamlCmd := chainsListCmd(state)
		require.NoError(t, yamlCmd.Flags().Set(flagYAML, "true"))
		yamlOut, yamlErrOut, err := executeCharacterizedChainsCommand(t, yamlCmd)
		require.NoError(t, err)
		require.Empty(t, yamlErrOut)
		require.Equal(t, characterizedChainsListYAML, strings.ReplaceAll(yamlOut, state.homePath, "<HOME>"))
	})
}

func TestCliChainsCharacterizesRegistryListFormatsOrderAndErrors(t *testing.T) {
	var mu sync.Mutex
	var requestedPaths []string
	withCharacterizedChainsTransport(t, func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		requestedPaths = append(requestedPaths, req.URL.String())
		mu.Unlock()
		return characterizedHTTPResponse(http.StatusOK, `{
			"sha":"root",
			"tree":[
				{"path":"cosmoshub","mode":"040000","type":"tree","sha":"1"},
				{"path":".github","mode":"040000","type":"tree","sha":"2"},
				{"path":"README.md","mode":"100644","type":"blob","sha":"3"},
				{"path":"osmosis","mode":"040000","type":"tree","sha":"4"}
			]
		}`), nil
	})

	newState := func(t *testing.T) *appState {
		state := newCharacterizedConfigState(t)
		state.config = DefaultConfig("")
		return state
	}

	stdout, stderr, err := executeCharacterizedChainsCommand(t, chainsRegistryList(newState(t)))
	require.NoError(t, err)
	require.Equal(t, "cosmoshub\nosmosis\n", stdout)
	require.Empty(t, stderr)

	jsonCmd := chainsRegistryList(newState(t))
	require.NoError(t, jsonCmd.Flags().Set(flagJSON, "true"))
	stdout, stderr, err = executeCharacterizedChainsCommand(t, jsonCmd)
	require.NoError(t, err)
	require.Equal(t, "[\"cosmoshub\",\"osmosis\"]\n", stdout)
	require.Empty(t, stderr)

	yamlCmd := chainsRegistryList(newState(t))
	require.NoError(t, yamlCmd.Flags().Set(flagYAML, "true"))
	stdout, stderr, err = executeCharacterizedChainsCommand(t, yamlCmd)
	require.NoError(t, err)
	require.Equal(t, "- cosmoshub\n- osmosis\n\n", stdout)
	require.Empty(t, stderr)

	bothCmd := chainsRegistryList(newState(t))
	require.NoError(t, bothCmd.Flags().Set(flagJSON, "true"))
	require.NoError(t, bothCmd.Flags().Set(flagYAML, "true"))
	stdout, stderr, err = executeCharacterizedChainsCommand(t, bothCmd)
	require.EqualError(t, err, "can't pass both --json and --yaml, must pick one")
	require.Empty(t, stdout)
	require.Empty(t, stderr)

	mu.Lock()
	paths := append([]string(nil), requestedPaths...)
	mu.Unlock()
	require.Len(t, paths, 4)
	for _, requested := range paths {
		require.Equal(t, "https://api.github.com/repos/cosmos/chain-registry/git/trees/master", requested)
	}

	t.Run("transport errors are returned with request context", func(t *testing.T) {
		withCharacterizedChainsTransport(t, func(*http.Request) (*http.Response, error) {
			return nil, errors.New("registry offline")
		})
		state := newState(t)

		stdout, stderr, err := executeCharacterizedChainsCommand(t, chainsRegistryList(state))

		require.ErrorContains(t, err, "registry offline")
		require.ErrorContains(t, err, "Get \"https://api.github.com/repos/cosmos/chain-registry/git/trees/master\"")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}

func TestCliChainsCharacterizesAddFileURLMutationSaveAndPrecedence(t *testing.T) {
	t.Run("file without name derives basename and saves runtime config", func(t *testing.T) {
		state := newCharacterizedConfigState(t)
		state.config = DefaultConfig("")
		filename := filepath.Join(t.TempDir(), "alpha.json")
		writeCharacterizedChainJSON(t, filename, "alpha-1")

		cmd := chainsAddCmd(state)
		stdout, stderr, err := executeCharacterizedChainsCommand(t, cmd, "--file", filename)

		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Contains(t, state.config.Chains, "alpha")
		disk := readCharacterizedDiskConfig(t, state.homePath)
		require.Contains(t, disk.ProviderConfigs, "alpha")
		require.Equal(t, "alpha-1", disk.ProviderConfigs["alpha"].Value.(*cosmos.CosmosProviderConfig).ChainID)
	})

	t.Run("file rejects more than one explicit name before mutation", func(t *testing.T) {
		state := newCharacterizedConfigState(t)
		state.config = DefaultConfig("")
		filename := filepath.Join(t.TempDir(), "alpha.json")
		writeCharacterizedChainJSON(t, filename, "alpha-1")

		stdout, stderr, err := executeCharacterizedChainsCommand(
			t, chainsAddCmd(state), "--file", filename, "alpha", "beta",
		)

		require.EqualError(t, err, "one chain name is required")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Empty(t, state.config.Chains)
		require.Empty(t, readCharacterizedDiskConfig(t, state.homePath).ProviderConfigs)
	})

	t.Run("url requires one name before issuing request then saves success", func(t *testing.T) {
		var requests int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			requests++
			require.Equal(t, "/alpha.json", req.URL.Path)
			_, err := io.WriteString(w, characterizedCosmosProviderJSON("alpha-1"))
			require.NoError(t, err)
		}))
		defer server.Close()

		state := newCharacterizedConfigState(t)
		state.config = DefaultConfig("")
		stdout, stderr, err := executeCharacterizedChainsCommand(
			t, chainsAddCmd(state), "--url", server.URL+"/alpha.json",
		)
		require.EqualError(t, err, "one chain name is required")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Zero(t, requests)

		stdout, stderr, err = executeCharacterizedChainsCommand(
			t, chainsAddCmd(state), "--url", server.URL+"/alpha.json", "alpha",
		)
		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Equal(t, 1, requests)
		require.Contains(t, state.config.Chains, "alpha")
		require.Contains(t, readCharacterizedDiskConfig(t, state.homePath).ProviderConfigs, "alpha")
	})

	t.Run("file url conflict wins before config and IO", func(t *testing.T) {
		state := &appState{log: zap.NewNop(), viper: newCharacterizedConfigState(t).viper}
		cmd := chainsAddCmd(state)
		stdout, stderr, err := executeCharacterizedChainsCommand(
			t, cmd, "--file", "/does/not/exist", "--url", "https://example.test/chain.json",
		)
		require.ErrorIs(t, err, errMultipleAddFlags)
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}

func TestCliChainsCharacterizesRegistryBatchContinuationForceAndTestnet(t *testing.T) {
	var mu sync.Mutex
	var requested []string
	withCharacterizedChainsTransport(t, func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		requested = append(requested, req.URL.Path)
		mu.Unlock()
		switch {
		case strings.Contains(req.URL.Path, "/missing/"):
			return characterizedHTTPResponse(http.StatusNotFound, "missing"), nil
		case strings.Contains(req.URL.Path, "/good/"):
			return characterizedHTTPResponse(http.StatusOK, characterizedRegistryChainJSON("good-canonical", "good-1")), nil
		case strings.Contains(req.URL.Path, "/no-rpc/"):
			return characterizedHTTPResponse(http.StatusOK, characterizedRegistryChainJSON("no-rpc-canonical", "no-rpc-1")), nil
		default:
			return characterizedHTTPResponse(http.StatusInternalServerError, "unexpected request"), nil
		}
	})

	t.Run("duplicate is skipped, failures continue, testnet path and canonical name are preserved", func(t *testing.T) {
		state, duplicate := characterizedChainsStateWithCosmos(t, "duplicate", "duplicate-1")
		logs := replaceCharacterizedChainsLogger(state)

		err := addChainsFromRegistry(context.Background(), state, true, true, []string{"duplicate", "missing", "good"})

		require.NoError(t, err)
		require.Same(t, duplicate, state.config.Chains["duplicate"])
		require.Contains(t, state.config.Chains, "good-canonical")
		require.NotContains(t, state.config.Chains, "good")
		require.Equal(t, []string{
			"Chain already exists",
			"Error retrieving chain",
			"Endpoints queried",
			"Config update status",
		}, characterizedLogMessages(logs))
		status := logs.FilterMessage("Config update status").All()[0].ContextMap()
		require.Equal(t, []any{"good"}, status["added"])
		require.Equal(t, []any{"missing"}, status["failed"])
		require.Equal(t, []any{"duplicate"}, status["already existed"])

		mu.Lock()
		paths := append([]string(nil), requested...)
		mu.Unlock()
		require.Equal(t, []string{
			"/cosmos/chain-registry/master/testnets/missing/chain.json",
			"/cosmos/chain-registry/master/testnets/good/chain.json",
		}, paths)
	})

	t.Run("force-add changes zero-RPC result from logged failure to addition", func(t *testing.T) {
		state := newCharacterizedConfigState(t)
		state.config = DefaultConfig("")
		logs := replaceCharacterizedChainsLogger(state)

		err := addChainsFromRegistry(context.Background(), state, false, false, []string{"no-rpc"})
		require.NoError(t, err)
		require.Empty(t, state.config.Chains)
		require.Equal(t, []string{
			"Endpoints queried",
			"Error generating chain config",
			"Config update status",
		}, characterizedLogMessages(logs))
		failureStatus := logs.FilterMessage("Config update status").All()[0].ContextMap()
		require.Equal(t, []any{"no-rpc"}, failureStatus["failed"])

		logs.TakeAll()
		err = addChainsFromRegistry(context.Background(), state, true, false, []string{"no-rpc"})
		require.NoError(t, err)
		require.Contains(t, state.config.Chains, "no-rpc-canonical")
		require.Equal(t, []string{"Endpoints queried", "Config update status"}, characterizedLogMessages(logs))
		successStatus := logs.FilterMessage("Config update status").All()[0].ContextMap()
		require.Equal(t, []any{"no-rpc"}, successStatus["added"])
	})
}

func characterizedChainsStateWithCosmos(t *testing.T, name, chainID string) (*appState, *relayer.Chain) {
	t.Helper()
	state := newCharacterizedConfigState(t)
	state.config = DefaultConfig("")
	config := cosmos.CosmosProviderConfig{
		Key:            "default",
		ChainID:        chainID,
		AccountPrefix:  "cosmos",
		KeyringBackend: "test",
		Timeout:        "10s",
	}
	provider, err := config.NewProvider(state.log, state.homePath, false, name)
	require.NoError(t, err)
	require.NoError(t, provider.Init(context.Background()))
	chain := relayer.NewChain(state.log, provider, false)
	state.config.Chains[name] = chain
	return state, chain
}

func executeCharacterizedChainsCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

func writeCharacterizedChainJSON(t *testing.T, filename, chainID string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filename, []byte(characterizedCosmosProviderJSON(chainID)), 0o600))
}

func characterizedCosmosProviderJSON(chainID string) string {
	wrapper := ProviderConfigWrapper{
		Type: "cosmos",
		Value: cosmos.CosmosProviderConfig{
			Key:            "default",
			ChainID:        chainID,
			AccountPrefix:  "cosmos",
			KeyringBackend: "test",
			Timeout:        "10s",
		},
	}
	contents, err := json.Marshal(wrapper)
	if err != nil {
		panic(err)
	}
	return string(contents)
}

func characterizedRegistryChainJSON(chainName, chainID string) string {
	return `{
		"chain_name":` + quotedJSON(chainName) + `,
		"chain_id":` + quotedJSON(chainID) + `,
		"bech32_prefix":"cosmos",
		"slip44":118,
		"fees":{"fee_tokens":[{"denom":"uatom","low_gas_price":0.01}]},
		"apis":{"rpc":[]}
	}`
}

func quotedJSON(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

type characterizedChainsRoundTripper func(*http.Request) (*http.Response, error)

func (fn characterizedChainsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withCharacterizedChainsTransport(
	t *testing.T,
	fn func(*http.Request) (*http.Response, error),
) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = characterizedChainsRoundTripper(fn)
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func characterizedHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func replaceCharacterizedChainsLogger(state *appState) *observer.ObservedLogs {
	core, logs := observer.New(zap.DebugLevel)
	state.log = zap.New(core)
	return logs
}

const characterizedChainsShowJSON = `{"type":"cosmos","value":{"key-directory":"<HOME>/keys/alpha-1","key":"default","chain-id":"alpha-1","rpc-addr":"","backup-rpc-addrs":null,"account-prefix":"cosmos","keyring-backend":"test","dynamic-gas-price":false,"gas-adjustment":0,"gas-prices":"","min-gas-amount":0,"max-gas-amount":0,"debug":false,"timeout":"10s","block-timeout":"","output-format":"","sign-mode":"","extra-codecs":null,"coin-type":null,"signing-algorithm":"","broadcast-mode":"batch","min-loop-duration":0,"extension-options":null,"feegrants":null}}
`

const characterizedChainsShowYAML = `type: cosmos
value:
    key-directory: <HOME>/keys/alpha-1
    key: default
    chain-id: alpha-1
    rpc-addr: ""
    backup-rpc-addrs: []
    account-prefix: cosmos
    keyring-backend: test
    dynamic-gas-price: false
    gas-adjustment: 0
    gas-prices: ""
    min-gas-amount: 0
    max-gas-amount: 0
    debug: false
    timeout: 10s
    block-timeout: ""
    output-format: ""
    sign-mode: ""
    extra-codecs: []
    coin-type: null
    signing-algorithm: ""
    broadcast-mode: batch
    min-loop-duration: 0s
    extension-options: []
    feegrants: null

`

const characterizedChainsListYAML = `alpha:
    type: cosmos
    value:
        key-directory: <HOME>/keys/alpha-1
        key: default
        chain-id: alpha-1
        rpc-addr: ""
        backup-rpc-addrs: []
        account-prefix: cosmos
        keyring-backend: test
        dynamic-gas-price: false
        gas-adjustment: 0
        gas-prices: ""
        min-gas-amount: 0
        max-gas-amount: 0
        debug: false
        timeout: 10s
        block-timeout: ""
        output-format: ""
        sign-mode: ""
        extra-codecs: []
        coin-type: null
        signing-algorithm: ""
        broadcast-mode: batch
        min-loop-duration: 0s
        extension-options: []
        feegrants: null

`
