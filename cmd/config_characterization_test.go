package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestConfigShowCmdCharacterization(t *testing.T) {
	state := newCharacterizedConfigState(t)
	state.config = DefaultConfig("characterized memo")

	jsonOutput, err := json.Marshal(state.config.Wrapped())
	require.NoError(t, err)
	yamlOutput, err := yaml.Marshal(state.config.Wrapped())
	require.NoError(t, err)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "yaml is the default", want: string(yamlOutput) + "\n"},
		{name: "explicit yaml", args: []string{"--yaml"}, want: string(yamlOutput) + "\n"},
		{name: "json", args: []string{"--json"}, want: string(jsonOutput) + "\n"},
		{name: "json shorthand", args: []string{"-j"}, want: string(jsonOutput) + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := executeCharacterizedConfigCommand(
				t, configShowCmd(state), state.homePath, tt.args...,
			)
			require.NoError(t, err)
			require.Equal(t, tt.want, stdout)
		})
	}
}

func TestConfigShowCmdErrorsAndMetadata(t *testing.T) {
	state := &appState{viper: viper.New(), config: DefaultConfig("")}
	cmd := configShowCmd(state)
	require.Equal(t, "show", cmd.Use)
	require.Equal(t, []string{"s", "list", "l"}, cmd.Aliases)
	require.Equal(t, "Prints current configuration", cmd.Short)
	require.NotNil(t, cmd.Args)
	require.NotNil(t, cmd.Flags().Lookup(flagJSON))
	require.NotNil(t, cmd.Flags().Lookup(flagYAML))

	missingHome := filepath.Join(t.TempDir(), "missing")
	_, _, err := executeCharacterizedConfigCommand(t, configShowCmd(state), missingHome)
	require.EqualError(t, err, "home path does not exist: "+missingHome)

	homeWithoutConfig := t.TempDir()
	_, _, err = executeCharacterizedConfigCommand(t, configShowCmd(state), homeWithoutConfig)
	require.EqualError(t, err, "config does not exist: "+filepath.Join(homeWithoutConfig, "config", "config.yaml"))

	homeWithConfig := newCharacterizedConfigState(t).homePath
	_, _, err = executeCharacterizedConfigCommand(t, configShowCmd(state), homeWithConfig, "--json", "--yaml")
	require.EqualError(t, err, "can't pass both --json and --yaml, must pick one")
}

func TestConfigShowCmdIgnoresOutputWriterErrors(t *testing.T) {
	state := newCharacterizedConfigState(t)
	state.config = DefaultConfig("")
	cmd := configShowCmd(state)
	root := characterizedConfigRoot(state.homePath, cmd)
	root.SetOut(characterizedConfigFailingWriter{})
	root.SetArgs([]string{"show", "--json"})
	require.NoError(t, root.ExecuteContext(context.Background()))
}

func TestConfigInitCmdCreatesDefaultConfig(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "new-home")
	state := &appState{viper: viper.New()}
	cmd := configInitCmd(state)
	require.Equal(t, "init", cmd.Use)
	require.Equal(t, []string{"i"}, cmd.Aliases)
	require.Equal(t, "Creates a default home directory at path defined by --home", cmd.Short)
	require.NotNil(t, cmd.Args)
	require.NotNil(t, cmd.Flags().Lookup(flagMemo))

	stdout, stderr, err := executeCharacterizedConfigCommand(t, cmd, home, "--memo", "campaign memo")
	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)

	configBytes, err := os.ReadFile(filepath.Join(home, "config", "config.yaml"))
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML("campaign memo"), configBytes)
}

func TestConfigInitCmdExistingConfigAndMissingParentErrors(t *testing.T) {
	home := t.TempDir()
	state := &appState{viper: viper.New()}
	_, _, err := executeCharacterizedConfigCommand(t, configInitCmd(state), home)
	require.NoError(t, err)

	configPath := filepath.Join(home, "config", "config.yaml")
	_, _, err = executeCharacterizedConfigCommand(t, configInitCmd(state), home)
	require.EqualError(t, err, "config already exists: "+configPath)

	missingParentHome := filepath.Join(t.TempDir(), "missing-parent", "home")
	_, _, err = executeCharacterizedConfigCommand(t, configInitCmd(state), missingParentHome)
	require.Error(t, err)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Equal(t, missingParentHome, pathErr.Path)
	require.NoDirExists(t, missingParentHome)
}

func TestAddChainsFromDirectoryContinuesAfterEntryFailures(t *testing.T) {
	state := newCharacterizedConfigState(t)
	inputDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(inputDir, "00-directory"), 0o755))
	require.NoError(t, os.Symlink(filepath.Join(inputDir, "missing-target"), filepath.Join(inputDir, "01-broken.json")))
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "02-malformed.json"), []byte("{"), 0o600))
	writeCharacterizedJSON(t, filepath.Join(inputDir, "03-invalid-provider.json"), characterizedCosmosProvider("chain-invalid", "invalid"))
	writeCharacterizedJSON(t, filepath.Join(inputDir, "04-good.json"), characterizedCosmosProvider("chain-A", "10s"))
	writeCharacterizedJSON(t, filepath.Join(inputDir, "05-duplicate.json"), characterizedCosmosProvider("chain-A", "10s"))

	var stderr bytes.Buffer
	err := addChainsFromDirectory(context.Background(), &stderr, state, inputDir)
	require.NoError(t, err)
	require.Len(t, state.config.Chains, 1)
	require.Contains(t, state.config.Chains, "04-good")

	assertCharacterizedFragmentsInOrder(t, stderr.String(), []string{
		"directory at " + filepath.Join(inputDir, "00-directory") + ", skipping...",
		"failed to read file " + filepath.Join(inputDir, "01-broken.json"),
		"failed to unmarshal file " + filepath.Join(inputDir, "02-malformed.json"),
		"failed to build ChainProvider for " + filepath.Join(inputDir, "03-invalid-provider.json"),
		"added chain chain-A...",
		"failed to add chain " + filepath.Join(inputDir, "05-duplicate.json") + ": chain with ID chain-A already exists in config",
	})

	diskConfig := readCharacterizedDiskConfig(t, state.homePath)
	require.Contains(t, diskConfig.ProviderConfigs, "04-good")
}

func TestAddChainsFromDirectoryReturnsDirectoryReadError(t *testing.T) {
	state := newCharacterizedConfigState(t)
	missingDir := filepath.Join(t.TempDir(), "missing")
	err := addChainsFromDirectory(context.Background(), io.Discard, state, missingDir)
	require.Error(t, err)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Equal(t, missingDir, pathErr.Path)
}

func TestAddPathsFromDirectorySuccessAndNaming(t *testing.T) {
	state := newCharacterizedConfigState(t)
	inputDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(inputDir, "00-directory"), 0o755))
	writeCharacterizedJSON(t, filepath.Join(inputDir, "alpha.extra.json"), characterizedPath())

	var stderr bytes.Buffer
	err := addPathsFromDirectory(context.Background(), &stderr, state, inputDir)
	require.NoError(t, err)
	require.Contains(t, state.config.Paths, "alpha")
	require.Equal(t, strings.Join([]string{
		"directory at " + filepath.Join(inputDir, "00-directory") + ", skipping...",
		"Chain chain-a is not currently configured.",
		"Chain chain-b is not currently configured.",
		"added path alpha...",
		"",
		"",
	}, "\n"), stderr.String())

	diskConfig := readCharacterizedDiskConfig(t, state.homePath)
	require.Contains(t, diskConfig.Paths, "alpha")
}

func TestAddPathsFromDirectoryStopsAfterFirstFailureWithPartialMemoryOnly(t *testing.T) {
	state := newCharacterizedConfigState(t)
	inputDir := t.TempDir()
	writeCharacterizedJSON(t, filepath.Join(inputDir, "01-first.json"), characterizedPath())
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "02-malformed.json"), []byte("{"), 0o600))
	writeCharacterizedJSON(t, filepath.Join(inputDir, "03-never-read.json"), characterizedPath())

	var stderr bytes.Buffer
	err := addPathsFromDirectory(context.Background(), &stderr, state, inputDir)
	require.EqualError(t, err, "failed to unmarshal file "+filepath.Join(inputDir, "02-malformed.json")+": unexpected end of JSON input")
	require.Contains(t, state.config.Paths, "01-first")
	require.NotContains(t, state.config.Paths, "03-never-read")
	require.Contains(t, stderr.String(), "added path 01-first...")

	diskConfig := readCharacterizedDiskConfig(t, state.homePath)
	require.Empty(t, diskConfig.Paths, "the failed locking operation must not persist its partial mutation")
}

func TestAddPathsFromDirectoryWrapsReadAndValidationErrors(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		state := newCharacterizedConfigState(t)
		inputDir := t.TempDir()
		brokenPath := filepath.Join(inputDir, "broken.json")
		require.NoError(t, os.Symlink(filepath.Join(inputDir, "missing-target"), brokenPath))

		err := addPathsFromDirectory(context.Background(), io.Discard, state, inputDir)
		require.ErrorContains(t, err, "failed to read file "+brokenPath+":")
	})

	t.Run("validation", func(t *testing.T) {
		state := newCharacterizedConfigState(t)
		inputDir := t.TempDir()
		invalidPath := filepath.Join(inputDir, "invalid.json")
		writeCharacterizedJSON(t, invalidPath, map[string]any{"src": nil, "dst": map[string]any{"chain-id": "chain-b"}})

		err := addPathsFromDirectory(context.Background(), io.Discard, state, inputDir)
		require.EqualError(t, err, "failed to validate path "+invalidPath+": path source is nil")
	})
}

func TestUpdatePathConfigRejectsEmptyAndMissingNames(t *testing.T) {
	state := &appState{}
	err := state.updatePathConfig(context.Background(), "", "", "", "", "")
	require.EqualError(t, err, "empty path name not allowed")
	require.Nil(t, state.config, "empty names return before config loading or mutation")

	state = newCharacterizedConfigState(t)
	err = state.updatePathConfig(context.Background(), "missing", "", "", "", "")
	require.EqualError(t, err, "config does not exist for that path: missing")
	require.Empty(t, state.config.Paths)
}

func TestUpdatePathConfigMutatesOnlyNonEmptyFieldsAndPersists(t *testing.T) {
	state := newCharacterizedConfigState(t)
	cfg := DefaultConfig("")
	cfg.Paths["demo"] = characterizedPath()
	writeCharacterizedRuntimeConfig(t, state.homePath, cfg)

	err := state.updatePathConfig(
		context.Background(), "demo",
		"07-tendermint-9", "", "", "connection-9",
	)
	require.NoError(t, err)

	updated := state.config.Paths["demo"]
	require.Equal(t, "07-tendermint-9", updated.Src.ClientID)
	require.Equal(t, "connection-0", updated.Src.ConnectionID)
	require.Equal(t, "07-tendermint-1", updated.Dst.ClientID)
	require.Equal(t, "connection-9", updated.Dst.ConnectionID)

	diskConfig := readCharacterizedDiskConfig(t, state.homePath)
	require.Equal(t, updated, diskConfig.Paths["demo"])
}

func TestAddPathFromUserInputSuccess(t *testing.T) {
	state := &appState{config: DefaultConfig("")}
	input := characterizedConfigInput("07-tendermint-0\nconnection-0\n07-tendermint-1\nconnection-1\n")
	var stderr bytes.Buffer

	err := state.addPathFromUserInput(context.Background(), input, &stderr, "chain-a", "chain-b", "demo")
	require.NoError(t, err)
	require.Equal(t, characterizedPath(), state.config.Paths["demo"])
	require.Equal(t, strings.Join([]string{
		"enter src(chain-a) client-id...",
		"enter src(chain-a) connection-id...",
		"enter dst(chain-b) client-id...",
		"enter dst(chain-b) connection-id...",
		"Chain chain-a is not currently configured.",
		"Chain chain-b is not currently configured.",
		"",
	}, "\n"), stderr.String())
}

func TestAddPathFromUserInputReturnsReadErrorsAtEveryPrompt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "source client", input: "", want: "enter src(chain-a) client-id...\n"},
		{name: "source connection", input: "07-tendermint-0\n", want: "enter src(chain-a) client-id...\nenter src(chain-a) connection-id...\n"},
		{name: "destination client", input: "07-tendermint-0\nconnection-0\n", want: "enter src(chain-a) client-id...\nenter src(chain-a) connection-id...\nenter dst(chain-b) client-id...\n"},
		{name: "destination connection", input: "07-tendermint-0\nconnection-0\n07-tendermint-1\n", want: "enter src(chain-a) client-id...\nenter src(chain-a) connection-id...\nenter dst(chain-b) client-id...\nenter dst(chain-b) connection-id...\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &appState{config: DefaultConfig("")}
			var stderr bytes.Buffer
			err := state.addPathFromUserInput(
				context.Background(), characterizedConfigInput(tt.input), &stderr,
				"chain-a", "chain-b", "demo",
			)
			require.ErrorIs(t, err, io.EOF)
			require.Equal(t, tt.want, stderr.String())
			require.Empty(t, state.config.Paths)
		})
	}
}

func TestAddPathFromUserInputBulkReaderStopsAfterFirstBufferedLine(t *testing.T) {
	state := &appState{config: DefaultConfig("")}
	input := strings.NewReader("07-tendermint-0\nconnection-0\n07-tendermint-1\nconnection-1\n")
	var stderr bytes.Buffer

	err := state.addPathFromUserInput(context.Background(), input, &stderr, "chain-a", "chain-b", "demo")
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, "enter src(chain-a) client-id...\nenter src(chain-a) connection-id...\n", stderr.String())
	require.Empty(t, state.config.Paths)
}

func TestAddPathFromUserInputValidatesEachIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrompt string
	}{
		{name: "source client", input: "!\n", wantPrompt: "enter src(chain-a) client-id..."},
		{name: "source connection", input: "07-tendermint-0\n!\n", wantPrompt: "enter src(chain-a) connection-id..."},
		{name: "destination client", input: "07-tendermint-0\nconnection-0\n!\n", wantPrompt: "enter dst(chain-b) client-id..."},
		{name: "destination connection", input: "07-tendermint-0\nconnection-0\n07-tendermint-1\n!\n", wantPrompt: "enter dst(chain-b) connection-id..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &appState{config: DefaultConfig("")}
			var stderr bytes.Buffer
			err := state.addPathFromUserInput(
				context.Background(), characterizedConfigInput(tt.input), &stderr,
				"chain-a", "chain-b", "demo",
			)
			require.ErrorContains(t, err, "invalid identifier")
			require.Contains(t, stderr.String(), tt.wantPrompt)
			require.Empty(t, state.config.Paths)
		})
	}
}

func TestAddPathFromUserInputPreservesAddPathConflict(t *testing.T) {
	state := &appState{config: DefaultConfig("")}
	existing := characterizedPath()
	state.config.Paths["demo"] = existing
	input := characterizedConfigInput("07-tendermint-9\nconnection-0\n07-tendermint-1\nconnection-1\n")

	err := state.addPathFromUserInput(context.Background(), input, io.Discard, "chain-a", "chain-b", "demo")
	require.EqualError(t, err, "path with ID demo and conflicting source client ID (07-tendermint-0) already exists")
	require.Same(t, existing, state.config.Paths["demo"])
	require.Equal(t, "07-tendermint-0", existing.Src.ClientID)
}

func newCharacterizedConfigState(t *testing.T) *appState {
	t.Helper()
	home := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(home, "config"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "config.yaml"), defaultConfigYAML(""), 0o600))
	return &appState{homePath: home, viper: viper.New(), log: zap.NewNop()}
}

func characterizedConfigRoot(home string, child *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "rly", SilenceUsage: true}
	root.PersistentFlags().String(flagHome, home, "")
	root.AddCommand(child)
	return root
}

func executeCharacterizedConfigCommand(
	t *testing.T,
	child *cobra.Command,
	home string,
	args ...string,
) (stdout, stderr string, err error) {
	t.Helper()
	root := characterizedConfigRoot(home, child)
	var stdoutBuffer, stderrBuffer bytes.Buffer
	root.SetOut(&stdoutBuffer)
	root.SetErr(&stderrBuffer)
	root.SetArgs(append([]string{child.Name()}, args...))
	err = root.ExecuteContext(context.Background())
	return stdoutBuffer.String(), stderrBuffer.String(), err
}

func characterizedCosmosProvider(chainID, timeout string) map[string]any {
	return map[string]any{
		"type": "cosmos",
		"value": map[string]any{
			"chain-id":        chainID,
			"keyring-backend": "test",
			"timeout":         timeout,
		},
	}
}

func characterizedPath() *relayer.Path {
	return &relayer.Path{
		Src: &relayer.PathEnd{
			ChainID:      "chain-a",
			ClientID:     "07-tendermint-0",
			ConnectionID: "connection-0",
		},
		Dst: &relayer.PathEnd{
			ChainID:      "chain-b",
			ClientID:     "07-tendermint-1",
			ConnectionID: "connection-1",
		},
	}
}

func writeCharacterizedJSON(t *testing.T, filename string, value any) {
	t.Helper()
	contents, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filename, contents, 0o600))
}

func writeCharacterizedRuntimeConfig(t *testing.T, home string, config *Config) {
	t.Helper()
	contents, err := yaml.Marshal(config.Wrapped())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "config.yaml"), contents, 0o600))
}

func readCharacterizedDiskConfig(t *testing.T, home string) ConfigInputWrapper {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(home, "config", "config.yaml"))
	require.NoError(t, err)
	var config ConfigInputWrapper
	require.NoError(t, yaml.Unmarshal(contents, &config))
	return config
}

func assertCharacterizedFragmentsInOrder(t *testing.T, output string, fragments []string) {
	t.Helper()
	position := 0
	for _, fragment := range fragments {
		index := strings.Index(output[position:], fragment)
		require.NotEqual(t, -1, index, "missing ordered fragment %q in %q", fragment, output)
		position += index + len(fragment)
	}
}

type characterizedConfigFailingWriter struct{}

func (characterizedConfigFailingWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("characterized writer failure")
}

type characterizedConfigStreamingReader struct {
	reader *strings.Reader
}

func characterizedConfigInput(value string) *characterizedConfigStreamingReader {
	return &characterizedConfigStreamingReader{reader: strings.NewReader(value)}
}

func (r *characterizedConfigStreamingReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.reader.Read(p)
}
