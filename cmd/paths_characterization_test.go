package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCliPathsCharacterizesCobraStructureAndDefaults(t *testing.T) {
	state := newCharacterizedConfigState(t)
	state.config = DefaultConfig("")

	list := pathsListCmd(state)
	require.Equal(t, "list", list.Use)
	require.Equal(t, []string{"l"}, list.Aliases)
	require.False(t, mustBoolFlag(t, list, flagJSON))
	require.False(t, mustBoolFlag(t, list, flagYAML))
	require.NoError(t, list.Args(list, nil))
	require.Error(t, list.Args(list, []string{"extra"}))

	show := pathsShowCmd(state)
	require.Equal(t, "show path_name", show.Use)
	require.Equal(t, []string{"s"}, show.Aliases)
	require.False(t, mustBoolFlag(t, show, flagJSON))
	require.False(t, mustBoolFlag(t, show, flagYAML))
	require.NoError(t, show.Args(show, []string{"demo"}))
	require.Error(t, show.Args(show, nil))

	add := pathsAddCmd(state)
	require.Equal(t, "add src_chain_id dst_chain_id path_name", add.Use)
	require.Equal(t, []string{"a"}, add.Aliases)
	require.Empty(t, mustStringFlag(t, add, flagFile))
	require.NoError(t, add.Args(add, []string{"a", "b", "demo"}))
	require.Error(t, add.Args(add, []string{"a", "b"}))

	update := pathsUpdateCmd(state)
	require.Equal(t, "update path_name", update.Use)
	require.Equal(t, []string{"n"}, update.Aliases)
	require.Equal(t, blankValue, mustStringFlag(t, update, flagFilterRule))
	require.Equal(t, blankValue, mustStringFlag(t, update, flagFilterChannels))
	for _, name := range []string{
		flagSrcChainID, flagDstChainID, flagSrcClientID, flagDstClientID, flagSrcConnID, flagDstConnID,
	} {
		require.Empty(t, mustStringFlag(t, update, name))
	}
	require.NoError(t, update.Args(update, []string{"demo"}))
	require.Error(t, update.Args(update, nil))

	fetch := pathsFetchCmd(state)
	require.Equal(t, "fetch", fetch.Use)
	require.Equal(t, []string{"fch"}, fetch.Aliases)
	require.False(t, mustBoolFlag(t, fetch, flagOverwriteConfig))
	require.False(t, mustBoolFlag(t, fetch, flagTestnet))
	require.NoError(t, fetch.Args(fetch, nil))
	require.NoError(t, fetch.Args(fetch, []string{"a"}))
	require.Error(t, fetch.Args(fetch, []string{"a", "b"}))
}

func TestCliPathsCharacterizesListAndShowOrderAndOutput(t *testing.T) {
	state := characterizedPathsStatusState()

	t.Run("list ignores absent format flags and prints exact plain status", func(t *testing.T) {
		target := pathsListCmd(state)
		cmd := characterizedBarePathsCommand("list")

		err := target.RunE(cmd, nil)

		require.NoError(t, err)
		require.Equal(t, " 0: demo                 -> chns(✘) clnts(✘) conn(✘) (alpha-1<>beta-1)\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})

	t.Run("list rejects formats before querying status", func(t *testing.T) {
		state, src, dst := characterizedPathsStatusStateWithProviders()
		cmd := pathsListCmd(state)
		require.NoError(t, cmd.Flags().Set(flagJSON, "true"))
		require.NoError(t, cmd.Flags().Set(flagYAML, "true"))

		stdout, stderr, err := executeCharacterizedChainsCommand(t, cmd)

		require.EqualError(t, err, "can't pass both --json and --yaml, must pick one")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Zero(t, src.heightQueries)
		require.Zero(t, dst.heightQueries)
	})

	t.Run("list json and yaml preserve exact bytes", func(t *testing.T) {
		jsonCmd := pathsListCmd(state)
		require.NoError(t, jsonCmd.Flags().Set(flagJSON, "true"))
		stdout, stderr, err := executeCharacterizedChainsCommand(t, jsonCmd)
		require.NoError(t, err)
		require.Empty(t, stderr)
		require.Equal(t, characterizedPathsListJSON, stdout)

		yamlCmd := pathsListCmd(state)
		require.NoError(t, yamlCmd.Flags().Set(flagYAML, "true"))
		stdout, stderr, err = executeCharacterizedChainsCommand(t, yamlCmd)
		require.NoError(t, err)
		require.Empty(t, stderr)
		require.Equal(t, characterizedPathsListYAML, stdout)
	})

	t.Run("show path then chains precede ignored format getters", func(t *testing.T) {
		target := pathsShowCmd(state)
		cmd := characterizedBarePathsCommand("show")

		err := target.RunE(cmd, []string{"missing"})
		require.EqualError(t, err, "path with name missing does not exist")
		require.Empty(t, cmd.OutOrStdout().(*bytes.Buffer).String())

		missingChains := characterizedPathsStatusState()
		missingChains.config.Chains = make(relayer.Chains)
		target = pathsShowCmd(missingChains)
		err = target.RunE(cmd, []string{"demo"})
		require.EqualError(t, err, "chain with ID alpha-1 is not configured")
	})

	t.Run("show plain json and yaml preserve exact bytes", func(t *testing.T) {
		stdout, stderr, err := executeCharacterizedChainsCommand(t, pathsShowCmd(state), "demo")
		require.NoError(t, err)
		require.Empty(t, stderr)
		require.Equal(t, characterizedPathsShowPlain, stdout)

		jsonCmd := pathsShowCmd(state)
		require.NoError(t, jsonCmd.Flags().Set(flagJSON, "true"))
		stdout, stderr, err = executeCharacterizedChainsCommand(t, jsonCmd, "demo")
		require.NoError(t, err)
		require.Empty(t, stderr)
		require.Equal(t, characterizedPathsShowJSON, stdout)

		yamlCmd := pathsShowCmd(state)
		require.NoError(t, yamlCmd.Flags().Set(flagYAML, "true"))
		stdout, stderr, err = executeCharacterizedChainsCommand(t, yamlCmd, "demo")
		require.NoError(t, err)
		require.Empty(t, stderr)
		require.Equal(t, characterizedPathsShowYAML, stdout)
	})
}

func TestCliPathsCharacterizesAddValidationMutationAndSave(t *testing.T) {
	t.Run("chain lookup precedes file getter", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		target := pathsAddCmd(state)
		cmd := characterizedBarePathsCommand("add")

		err := target.RunE(cmd, []string{"missing", "beta-1", "demo"})
		require.EqualError(t, err, "chains need to be configured before paths to them can be added: chain with ID missing is not configured")

		err = target.RunE(cmd, []string{"alpha-1", "beta-1", "demo"})
		require.EqualError(t, err, "flag accessed but not defined: file")
		require.Empty(t, state.config.Paths)
	})

	t.Run("file path fields win over positional chain IDs and are saved", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		filename := filepath.Join(t.TempDir(), "path.json")
		writeCharacterizedPathJSON(t, filename, characterizedClassicPath("beta-1", "alpha-1"))

		cmd := pathsAddCmd(state)
		stdout, stderr, err := executeCharacterizedChainsCommand(
			t, cmd, "alpha-1", "beta-1", "reverse", "--file", filename,
		)

		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Equal(t, "beta-1", state.config.Paths["reverse"].Src.ChainID)
		require.Equal(t, "alpha-1", state.config.Paths["reverse"].Dst.ChainID)
		disk := readCharacterizedDiskConfig(t, state.homePath)
		require.Equal(t, "beta-1", disk.Paths["reverse"].Src.ChainID)
		require.Equal(t, "alpha-1", disk.Paths["reverse"].Dst.ChainID)
	})

	t.Run("invalid file path reports validation and does not save", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		filename := filepath.Join(t.TempDir(), "path.json")
		invalid := characterizedClassicPath("alpha-1", "beta-1")
		invalid.Src.ClientID = "invalid!"
		writeCharacterizedPathJSON(t, filename, invalid)

		stdout, stderr, err := executeCharacterizedChainsCommand(
			t, pathsAddCmd(state), "alpha-1", "beta-1", "demo", "--file", filename,
		)

		require.ErrorContains(t, err, "invalid identifier")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Empty(t, state.config.Paths)
		require.Empty(t, readCharacterizedDiskConfig(t, state.homePath).Paths)
	})
}

func TestCliPathsCharacterizesUpdateFieldSemanticsAndPersistence(t *testing.T) {
	t.Run("no action and invalid filter fail before mutation", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		state.config.Paths["demo"] = characterizedClassicPath("alpha-1", "beta-1")
		writeCharacterizedRuntimeConfig(t, state.homePath, state.config)

		stdout, stderr, err := executeCharacterizedChainsCommand(t, pathsUpdateCmd(state), "demo")
		require.EqualError(t, err, "at least one flag must be provided")
		require.Empty(t, stdout)
		require.Empty(t, stderr)

		cmd := pathsUpdateCmd(state)
		require.NoError(t, cmd.Flags().Set(flagFilterRule, "invalid"))
		require.NoError(t, cmd.Flags().Set(flagSrcChainID, "changed-1"))
		stdout, stderr, err = executeCharacterizedChainsCommand(t, cmd, "demo")
		require.EqualError(t, err, `invalid filter rule : "invalid". valid rules: ("", "allowlist", "denylist")`)
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Equal(t, "alpha-1", state.config.Paths["demo"].Src.ChainID)
	})

	t.Run("all fields update in getter order and save even with unconfigured chain IDs", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		state.config.Paths["demo"] = characterizedClassicPath("alpha-1", "beta-1")
		writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
		cmd := pathsUpdateCmd(state)
		values := map[string]string{
			flagFilterRule:     processor.RuleAllowList,
			flagFilterChannels: "channel-2,channel-3",
			flagSrcChainID:     "new-alpha-1",
			flagDstChainID:     "new-beta-1",
			flagSrcClientID:    "07-tendermint-2",
			flagDstClientID:    "07-tendermint-3",
			flagSrcConnID:      "connection-2",
			flagDstConnID:      "connection-3",
		}
		for name, value := range values {
			require.NoError(t, cmd.Flags().Set(name, value))
		}

		stdout, stderr, err := executeCharacterizedChainsCommand(t, cmd, "demo")

		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		want := &relayer.Path{
			Protocol: protocol.ProtocolClassic,
			Src:      &relayer.PathEnd{ChainID: "new-alpha-1", ClientID: "07-tendermint-2", ConnectionID: "connection-2"},
			Dst:      &relayer.PathEnd{ChainID: "new-beta-1", ClientID: "07-tendermint-3", ConnectionID: "connection-3"},
			Filter: relayer.ChannelFilter{
				Rule: processor.RuleAllowList, ChannelList: []string{"channel-2", "channel-3"},
			},
		}
		require.Equal(t, want, state.config.Paths["demo"])
		require.Equal(t, want, readCharacterizedDiskConfig(t, state.homePath).Paths["demo"])
	})

	t.Run("explicit empty filters clear rule and turn channels into nil", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		path := characterizedClassicPath("alpha-1", "beta-1")
		path.Filter = relayer.ChannelFilter{Rule: processor.RuleDenyList, ChannelList: []string{"channel-7"}}
		state.config.Paths["demo"] = path
		writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
		cmd := pathsUpdateCmd(state)
		require.NoError(t, cmd.Flags().Set(flagFilterRule, ""))
		require.NoError(t, cmd.Flags().Set(flagFilterChannels, ""))

		_, _, err := executeCharacterizedChainsCommand(t, cmd, "demo")

		require.NoError(t, err)
		require.Empty(t, state.config.Paths["demo"].Filter.Rule)
		require.Nil(t, state.config.Paths["demo"].Filter.ChannelList)
		require.Empty(t, readCharacterizedDiskConfig(t, state.homePath).Paths["demo"].Filter.ChannelList)
	})

	t.Run("update does not validate identifier syntax and saves invalid value", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		state.config.Paths["demo"] = characterizedClassicPath("alpha-1", "beta-1")
		writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
		cmd := pathsUpdateCmd(state)
		require.NoError(t, cmd.Flags().Set(flagSrcClientID, "invalid!"))

		_, _, err := executeCharacterizedChainsCommand(t, cmd, "demo")

		require.NoError(t, err)
		require.Equal(t, "invalid!", state.config.Paths["demo"].Src.ClientID)
		require.Equal(t, "invalid!", readCharacterizedDiskConfig(t, state.homePath).Paths["demo"].Src.ClientID)
	})

	t.Run("missing path panics through MustGet after config reload", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		cmd := pathsUpdateCmd(state)
		require.NoError(t, cmd.Flags().Set(flagSrcChainID, "changed"))

		require.PanicsWithError(t, "path with name missing does not exist", func() {
			_, _, _ = executeCharacterizedChainsCommand(t, cmd, "missing")
		})
	})

	t.Run("v2 structural violation mutates memory but is rejected before save", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "alpha", "beta")
		state.config.Paths["v2"] = validConfigV2Path()
		writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
		cmd := pathsUpdateCmd(state)
		require.NoError(t, cmd.Flags().Set(flagSrcConnID, "connection-0"))

		stdout, stderr, err := executeCharacterizedChainsCommand(t, cmd, "v2")

		require.ErrorContains(t, err, "error parsing chain config")
		require.ErrorContains(t, err, "path protocol v2 cannot set source connection-id")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Equal(t, "connection-0", state.config.Paths["v2"].Src.ConnectionID)
		require.Empty(t, readCharacterizedDiskConfig(t, state.homePath).Paths["v2"].Src.ConnectionID)
	})
}

func TestCliPathsCharacterizesFetchSelectionNetworkAndOverwrite(t *testing.T) {
	t.Run("missing requested chain fails before lock and network", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "a", "b")
		requests := 0
		withCharacterizedChainsTransport(t, func(*http.Request) (*http.Response, error) {
			requests++
			return nil, errors.New("unexpected network")
		})

		stdout, stderr, err := executeCharacterizedChainsCommand(t, pathsFetchCmd(state), "missing")

		require.EqualError(t, err, "chain missing not found in config")
		require.Empty(t, stdout)
		require.Empty(t, stderr)
		require.Zero(t, requests)
	})

	t.Run("existing path skips request unless overwrite and overwrite clears filters", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "a", "b")
		existing := characterizedClassicPath("a-1", "b-1")
		existing.Src.ClientID = "07-tendermint-0"
		existing.Src.ConnectionID = "connection-0"
		existing.Dst.ClientID = "07-tendermint-1"
		existing.Dst.ConnectionID = "connection-1"
		existing.Filter = relayer.ChannelFilter{Rule: processor.RuleAllowList, ChannelList: []string{"channel-7"}}
		state.config.Paths["a-b"] = existing
		writeCharacterizedRuntimeConfig(t, state.homePath, state.config)

		requests := 0
		withCharacterizedChainsTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "api.github.com" {
				return characterizedGithubDirectoryResponse(req, "a-b"), nil
			}
			requests++
			return characterizedGithubDownloadResponse(req, characterizedIBCData("a", "b")), nil
		})

		stdout, stderr, err := executeCharacterizedChainsCommand(t, pathsFetchCmd(state))
		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Equal(t, "skipping:  a-b already exists in config, use -o to overwrite (clears filters)\n", stderr)
		require.Zero(t, requests)
		require.Equal(t, []string{"channel-7"}, state.config.Paths["a-b"].Filter.ChannelList)

		overwrite := pathsFetchCmd(state)
		require.NoError(t, overwrite.Flags().Set(flagOverwriteConfig, "true"))
		stdout, stderr, err = executeCharacterizedChainsCommand(t, overwrite)
		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Equal(t, "added:  a-b\n", stderr)
		require.Equal(t, 1, requests)
		require.Empty(t, state.config.Paths["a-b"].Filter)
		diskFilter := readCharacterizedDiskConfig(t, state.homePath).Paths["a-b"].Filter
		require.Empty(t, diskFilter.Rule)
		require.Empty(t, diskFilter.ChannelList)
	})

	t.Run("testnet changes contents path and saves fetched fields", func(t *testing.T) {
		state := characterizedPathsDiskState(t, "a", "b")
		var requestedDirectory string
		withCharacterizedChainsTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "api.github.com" {
				requestedDirectory = req.URL.Path
				return characterizedGithubDirectoryResponse(req, "a-b"), nil
			}
			return characterizedGithubDownloadResponse(req, characterizedIBCData("a", "b")), nil
		})
		cmd := pathsFetchCmd(state)
		require.NoError(t, cmd.Flags().Set(flagTestnet, "true"))

		stdout, stderr, err := executeCharacterizedChainsCommand(t, cmd, "a")

		require.NoError(t, err)
		require.Empty(t, stdout)
		require.Equal(t, "added:  a-b\n", stderr)
		require.Equal(t, "/repos/cosmos/chain-registry/contents/testnets/_IBC", requestedDirectory)
		path := state.config.Paths["a-b"]
		require.Equal(t, "a-1", path.Src.ChainID)
		require.Equal(t, "07-tendermint-0", path.Src.ClientID)
		require.Equal(t, "connection-0", path.Src.ConnectionID)
		require.Equal(t, "b-1", path.Dst.ChainID)
		require.Contains(t, readCharacterizedDiskConfig(t, state.homePath).Paths, "a-b")
	})
}

func TestCliPathsCharacterizesFetchBatchContinuationAndFatalPartialState(t *testing.T) {
	t.Run("retrieval failures continue and stderr follows request order", characterizeFetchRetrievalFailureContinuation)
	t.Run("fatal unmarshal stops batch after preserving memory-only prior addition", characterizeFetchFatalPartialState)
}

func characterizeFetchRetrievalFailureContinuation(t *testing.T) {
	state := characterizedPathsDiskState(t, "a", "b", "c")
	var downloaded []string
	directoryCalls := 0
	withCharacterizedChainsTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.github.com" {
			directoryCalls++
			if directoryCalls == 1 {
				return characterizedGithubDirectoryResponse(req), nil
			}
			return characterizedGithubDirectoryResponse(req, "a-b", "a-c", "b-c"), nil
		}
		pair := characterizedPairFromContentsPath(req.URL.Path)
		downloaded = append(downloaded, pair)
		left, right := pair[0:1], pair[2:3]
		return characterizedGithubDownloadResponse(req, characterizedIBCData(left, right)), nil
	})

	stdout, stderr, err := executeCharacterizedChainsCommand(t, pathsFetchCmd(state))

	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Equal(t, 3, directoryCalls)
	require.Len(t, downloaded, 2)
	lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
	require.Len(t, lines, 3)
	require.Contains(t, lines[0], "failure retrieving: ")
	require.Contains(t, lines[0], ": consider adding to cosmos/chain-registry: ERR:")
	require.Equal(t, "added:  "+downloaded[0], lines[1])
	require.Equal(t, "added:  "+downloaded[1], lines[2])
	failedPair := strings.Split(strings.TrimPrefix(lines[0], "failure retrieving: "), ":")[0]
	require.NotContains(t, state.config.Paths, failedPair)
	require.Contains(t, state.config.Paths, downloaded[0])
	require.Contains(t, state.config.Paths, downloaded[1])
}

func characterizeFetchFatalPartialState(t *testing.T) {
	state := characterizedPathsDiskState(t, "a", "b", "c")
	var downloaded []string
	withCharacterizedChainsTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.github.com" {
			return characterizedGithubDirectoryResponse(req, "a-b", "a-c", "b-c"), nil
		}
		pair := characterizedPairFromContentsPath(req.URL.Path)
		downloaded = append(downloaded, pair)
		if len(downloaded) == 2 {
			return characterizedGithubDownloadResponse(req, "not-json"), nil
		}
		left, right := pair[0:1], pair[2:3]
		return characterizedGithubDownloadResponse(req, characterizedIBCData(left, right)), nil
	})

	stdout, stderr, err := executeCharacterizedChainsCommand(t, pathsFetchCmd(state))

	require.ErrorContains(t, err, "failed to unmarshal")
	require.Empty(t, stdout)
	require.Len(t, downloaded, 2)
	require.Equal(t, "added:  "+downloaded[0]+"\n", stderr)
	require.Contains(t, state.config.Paths, downloaded[0])
	require.Empty(t, readCharacterizedDiskConfig(t, state.homePath).Paths)
}

func characterizedPathsStatusState() *appState {
	state, _, _ := characterizedPathsStatusStateWithProviders()
	return state
}

func characterizedPathsStatusStateWithProviders() (*appState, *pathValidationProvider, *pathValidationProvider) {
	src := &pathValidationProvider{chainID: "alpha-1", heightErr: errors.New("offline")}
	dst := &pathValidationProvider{chainID: "beta-1", heightErr: errors.New("offline")}
	state := &appState{
		log:   zap.NewNop(),
		viper: newCharacterizedConfigStateForViper(),
		config: &Config{
			Global: newDefaultGlobalConfig(""),
			Chains: relayer.Chains{
				"alpha": relayer.NewChain(zap.NewNop(), src, false),
				"beta":  relayer.NewChain(zap.NewNop(), dst, false),
			},
			Paths: relayer.Paths{"demo": characterizedStatusPath()},
		},
	}
	return state, src, dst
}

func newCharacterizedConfigStateForViper() *viper.Viper {
	return viper.New()
}

func characterizedStatusPath() *relayer.Path {
	path := characterizedClassicPath("alpha-1", "beta-1")
	path.Src.ClientID = "07-tendermint-0"
	path.Src.ConnectionID = "connection-0"
	path.Dst.ClientID = "07-tendermint-1"
	path.Dst.ConnectionID = "connection-1"
	return path
}

func characterizedPathsDiskState(t *testing.T, names ...string) *appState {
	t.Helper()
	state := newCharacterizedConfigState(t)
	state.config = DefaultConfig("")
	for _, name := range names {
		config := cosmos.CosmosProviderConfig{
			Key: "default", ChainID: name + "-1", AccountPrefix: "cosmos", KeyringBackend: "test", Timeout: "10s",
		}
		provider, err := config.NewProvider(state.log, state.homePath, false, name)
		require.NoError(t, err)
		state.config.Chains[name] = relayer.NewChain(state.log, provider, false)
	}
	writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
	return state
}

func characterizedClassicPath(src, dst string) *relayer.Path {
	return &relayer.Path{
		Protocol: protocol.ProtocolClassic,
		Src:      &relayer.PathEnd{ChainID: src},
		Dst:      &relayer.PathEnd{ChainID: dst},
	}
}

func characterizedBarePathsCommand(use string) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(""))
	return cmd
}

func writeCharacterizedPathJSON(t *testing.T, filename string, path *relayer.Path) {
	t.Helper()
	contents, err := json.Marshal(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filename, contents, 0o600))
}

func characterizedIBCData(left, right string) string {
	return fmt.Sprintf(`{
		"chain_1":{"chain_name":%q,"client_id":"07-tendermint-0","connection_id":"connection-0"},
		"chain_2":{"chain_name":%q,"client_id":"07-tendermint-1","connection_id":"connection-1"},
		"channels":[]
	}`, left, right)
}

func characterizedGithubDirectoryResponse(request *http.Request, pairs ...string) *http.Response {
	entries := make([]map[string]string, 0, len(pairs))
	for _, pair := range pairs {
		entries = append(entries, map[string]string{
			"type": "file", "name": pair + ".json", "path": "_IBC/" + pair + ".json",
			"download_url": "https://download.test/" + pair + ".json",
		})
	}
	body, err := json.Marshal(entries)
	if err != nil {
		panic(err)
	}
	response := characterizedHTTPResponse(http.StatusOK, string(body))
	response.Request = request
	return response
}

func characterizedGithubDownloadResponse(request *http.Request, raw string) *http.Response {
	response := characterizedHTTPResponse(http.StatusOK, raw)
	response.Request = request
	return response
}

func characterizedPairFromContentsPath(requestPath string) string {
	return strings.TrimSuffix(filepath.Base(requestPath), ".json")
}

const characterizedPathsListJSON = `{"demo":{"protocol":"classic","src":{"chain-id":"alpha-1","client-id":"07-tendermint-0","connection-id":"connection-0"},"dst":{"chain-id":"beta-1","client-id":"07-tendermint-1","connection-id":"connection-1"},"src-channel-filter":{"rule":"","channel-list":null}}}
`
const characterizedPathsListYAML = `demo:
    protocol: classic
    src:
        chain-id: alpha-1
        client-id: 07-tendermint-0
        connection-id: connection-0
    dst:
        chain-id: beta-1
        client-id: 07-tendermint-1
        connection-id: connection-1
    src-channel-filter:
        rule: ""
        channel-list: []

`
const characterizedPathsShowPlain = `Path "demo":
  SRC(alpha-1)
    ClientID:     07-tendermint-0
    ConnectionID: connection-0
  DST(beta-1)
    ClientID:     07-tendermint-1
    ConnectionID: connection-1
  STATUS:
    Chains:       ✘
    Clients:      ✘
    Connection:   ✘
`
const characterizedPathsShowJSON = `{"chains":{"protocol":"classic","src":{"chain-id":"alpha-1","client-id":"07-tendermint-0","connection-id":"connection-0"},"dst":{"chain-id":"beta-1","client-id":"07-tendermint-1","connection-id":"connection-1"},"src-channel-filter":{"rule":"","channel-list":null}},"status":{"chains":false,"clients":false,"connection":false}}
`
const characterizedPathsShowYAML = `path:
    protocol: classic
    src:
        chain-id: alpha-1
        client-id: 07-tendermint-0
        connection-id: connection-0
    dst:
        chain-id: beta-1
        client-id: 07-tendermint-1
        connection-id: connection-1
    src-channel-filter:
        rule: ""
        channel-list: []
status:
    chains: false
    clients: false
    connection: false

`
