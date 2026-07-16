package cmd

import (
	"errors"
	"net"
	"testing"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetAddInputsCharacterizesValuesAndConflicts(t *testing.T) {
	tests := []struct {
		name                   string
		file, url              string
		forceAdd, testnet      bool
		wantFile, wantURL      string
		wantForce, wantTestnet bool
		wantErr                error
	}{
		{name: "empty values"},
		{name: "file", file: "chain.json", wantFile: "chain.json"},
		{name: "url", url: "https://example.test/chain.json", wantURL: "https://example.test/chain.json"},
		{name: "force add", forceAdd: true, wantForce: true},
		{name: "testnet", testnet: true, wantTestnet: true},
		{
			name: "file and url conflict clears every result",
			file: "chain.json", url: "https://example.test/chain.json", forceAdd: true,
			wantErr: errMultipleAddFlags,
		},
		{
			name: "file and testnet conflict clears every result",
			file: "chain.json", forceAdd: true, testnet: true,
			wantErr: errInvalidTestnetFlag,
		},
		{
			name: "url and testnet conflict clears every result",
			url:  "https://example.test/chain.json", forceAdd: true, testnet: true,
			wantErr: errInvalidTestnetFlag,
		},
		{
			name: "file and url conflict has precedence over testnet conflict",
			file: "chain.json", url: "https://example.test/chain.json", forceAdd: true, testnet: true,
			wantErr: errMultipleAddFlags,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := characterizedAddInputsCommand(true, true, true, true)
			require.NoError(t, cmd.Flags().Set(flagFile, tt.file))
			require.NoError(t, cmd.Flags().Set(flagURL, tt.url))
			require.NoError(t, cmd.Flags().Set(flagForceAdd, boolString(tt.forceAdd)))
			require.NoError(t, cmd.Flags().Set(flagTestnet, boolString(tt.testnet)))

			file, url, forceAdd, testnet, err := getAddInputs(cmd)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantFile, file)
			require.Equal(t, tt.wantURL, url)
			require.Equal(t, tt.wantForce, forceAdd)
			require.Equal(t, tt.wantTestnet, testnet)
		})
	}
}

func TestGetAddInputsCharacterizesAccessorOrderAndPartialResults(t *testing.T) {
	tests := []struct {
		name                            string
		withFile, withURL, withForceAdd bool
		wantFile, wantURL               string
		wantForce                       bool
		wantErr                         string
	}{
		{name: "file is read first", wantErr: "flag accessed but not defined: file"},
		{
			name: "url is read second", withFile: true, wantFile: "chain.json",
			wantErr: "flag accessed but not defined: url",
		},
		{
			name: "force-add is read third", withFile: true, withURL: true,
			wantFile: "chain.json", wantURL: "https://example.test/chain.json",
			wantErr: "flag accessed but not defined: force-add",
		},
		{
			name: "testnet is read fourth", withFile: true, withURL: true, withForceAdd: true,
			wantFile: "chain.json", wantURL: "https://example.test/chain.json", wantForce: true,
			wantErr: "flag accessed but not defined: testnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := characterizedAddInputsCommand(tt.withFile, tt.withURL, tt.withForceAdd, false)
			file, url, forceAdd, testnet, err := getAddInputs(cmd)
			require.EqualError(t, err, tt.wantErr)
			require.Equal(t, tt.wantFile, file)
			require.Equal(t, tt.wantURL, url)
			require.Equal(t, tt.wantForce, forceAdd)
			require.False(t, testnet)
		})
	}
}

func TestSetupDebugServerCharacterizesAccessorOrder(t *testing.T) {
	tests := []struct {
		name                                           string
		withDeprecatedAddr, withListenAddr, withEnable bool
		wantErr                                        string
	}{
		{name: "deprecated address is read first", wantErr: "flag accessed but not defined: debug-addr"},
		{
			name: "listen address is read second", withDeprecatedAddr: true,
			wantErr: "flag accessed but not defined: debug-listen-addr",
		},
		{
			name: "enable switch is read third", withDeprecatedAddr: true, withListenAddr: true,
			wantErr: "flag accessed but not defined: enable-debug-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := characterizedDebugCommand(tt.withDeprecatedAddr, tt.withListenAddr, tt.withEnable)
			state, _ := characterizedDebugState(GlobalConfig{})
			err := setupDebugServer(cmd, state, errors.New("incoming error"))
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestSetupDebugServerCharacterizesPrecedenceLogsAndErrors(t *testing.T) {
	t.Run("deprecated config warning precedes disabled message and incoming error is ignored", func(t *testing.T) {
		cmd := characterizedDebugCommand(true, true, true)
		state, logs := characterizedDebugState(GlobalConfig{ApiListenPort: "127.0.0.1:1"})

		err := setupDebugServer(cmd, state, errors.New("incoming error"))

		require.NoError(t, err)
		require.Equal(t, []string{
			"DEPRECATED: api-listen-addr config setting is deprecated use debug-listen-addr instead",
			"Debug server is disabled you can enable it using --enable-debug-server flag",
		}, characterizedLogMessages(logs))
	})

	t.Run("enabled server without any address is disabled with one warning", func(t *testing.T) {
		cmd := characterizedDebugCommand(true, true, true)
		require.NoError(t, cmd.Flags().Set(flagEnableDebugServer, "true"))
		state, logs := characterizedDebugState(GlobalConfig{})

		err := setupDebugServer(cmd, state, nil)

		require.NoError(t, err)
		require.Equal(t, []string{
			"Disabled debug server due to missing debug-listen-addr setting in config file or --debug-listen-addr flag",
		}, characterizedLogMessages(logs))
	})

	t.Run("new listen flag overrides deprecated flag and config", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, listener.Close()) })
		occupiedAddr := listener.Addr().String()

		cmd := characterizedDebugCommand(true, true, true)
		require.NoError(t, cmd.Flags().Set(flagDebugAddr, "127.0.0.1:1"))
		require.NoError(t, cmd.Flags().Set(flagDebugListenAddr, occupiedAddr))
		state, logs := characterizedDebugState(GlobalConfig{DebugListenPort: "127.0.0.1:2"})

		err = setupDebugServer(cmd, state, nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to listen on debug address \""+occupiedAddr+"\":")
		require.Equal(t, []string{
			"DEPRECATED: --debug-addr flag is deprecated use --enable-debug-server and --debug-listen-addr instead",
			"Debug server is enabled",
			"SECURITY WARNING! Debug server should only be run with caution and proper security in place",
			"Failed to start debug server you can change the address and port using debug-listen-addr config settingh or --debug-listen-flag",
		}, characterizedLogMessages(logs))
	})
}

func TestStartCmdCharacterizesStructureDefaultsAndInitialSelection(t *testing.T) {
	state := &appState{
		log:    zap.NewNop(),
		viper:  viper.New(),
		config: DefaultConfig("configured memo"),
	}
	cmd := startCmd(state)

	require.Equal(t, "start path_name", cmd.Use)
	require.Equal(t, []string{"st"}, cmd.Aliases)
	require.Equal(t, "Start the listening relayer on a given path", cmd.Short)
	require.Equal(t, uint64(relayer.DefaultMaxMsgLength), mustUint64Flag(t, cmd, flagMaxMsgLength))
	require.Equal(t, relayer.ProcessorEvents, mustStringFlag(t, cmd, flagProcessor))
	require.Equal(t, uint64(20), mustUint64Flag(t, cmd, flagInitialBlockHistory))
	require.False(t, mustBoolFlag(t, cmd, flagEnableDebugServer))
	require.Empty(t, mustStringFlag(t, cmd, flagDebugAddr))
	require.Empty(t, mustStringFlag(t, cmd, flagDebugListenAddr))
	require.False(t, mustBoolFlag(t, cmd, flagEnableMetricsServer))
	require.Empty(t, mustStringFlag(t, cmd, flagMetricsListenAddr))
	require.Empty(t, mustStringFlag(t, cmd, flagMemo))
	require.Empty(t, mustStringFlag(t, cmd, flagStuckPacketChainID))
	require.Zero(t, mustUint64Flag(t, cmd, flagStuckPacketHeightStart))
	require.Zero(t, mustUint64Flag(t, cmd, flagStuckPacketHeightEnd))

	path := &relayer.Path{
		Protocol: protocol.ProtocolClassic,
		Src:      &relayer.PathEnd{ChainID: "chain-a"},
		Dst:      &relayer.PathEnd{ChainID: "chain-a"},
	}
	state.config.Paths["demo"] = path

	require.EqualError(t, cmd.RunE(cmd, []string{"demo"}), "chain with ID chain-a is not configured")
	require.EqualError(t, cmd.RunE(cmd, nil), "chain with ID chain-a is not configured")
	require.PanicsWithError(t, "path with name missing does not exist", func() {
		_ = cmd.RunE(cmd, []string{"missing"})
	})
}

func characterizedAddInputsCommand(withFile, withURL, withForceAdd, withTestnet bool) *cobra.Command {
	cmd := &cobra.Command{Use: "characterize-add-inputs"}
	if withFile {
		cmd.Flags().String(flagFile, "chain.json", "")
	}
	if withURL {
		cmd.Flags().String(flagURL, "https://example.test/chain.json", "")
	}
	if withForceAdd {
		cmd.Flags().Bool(flagForceAdd, true, "")
	}
	if withTestnet {
		cmd.Flags().Bool(flagTestnet, false, "")
	}
	return cmd
}

func characterizedDebugCommand(withDeprecatedAddr, withListenAddr, withEnable bool) *cobra.Command {
	cmd := &cobra.Command{Use: "characterize-debug"}
	if withDeprecatedAddr {
		cmd.Flags().String(flagDebugAddr, "", "")
	}
	if withListenAddr {
		cmd.Flags().String(flagDebugListenAddr, "", "")
	}
	if withEnable {
		cmd.Flags().Bool(flagEnableDebugServer, false, "")
	}
	return cmd
}

func characterizedDebugState(global GlobalConfig) (*appState, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return &appState{log: zap.New(core), config: &Config{Global: global}}, logs
}

func characterizedLogMessages(logs *observer.ObservedLogs) []string {
	entries := logs.All()
	messages := make([]string, len(entries))
	for i, entry := range entries {
		messages[i] = entry.Message
	}
	return messages
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func mustStringFlag(t *testing.T, cmd *cobra.Command, name string) string {
	t.Helper()
	value, err := cmd.Flags().GetString(name)
	require.NoError(t, err)
	return value
}

func mustUint64Flag(t *testing.T, cmd *cobra.Command, name string) uint64 {
	t.Helper()
	value, err := cmd.Flags().GetUint64(name)
	require.NoError(t, err)
	return value
}

func mustBoolFlag(t *testing.T, cmd *cobra.Command, name string) bool {
	t.Helper()
	value, err := cmd.Flags().GetBool(name)
	require.NoError(t, err)
	return value
}
