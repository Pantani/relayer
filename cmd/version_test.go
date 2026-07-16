package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type characterizedVersionInfo struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	CosmosSDK string `json:"cosmos-sdk" yaml:"cosmos-sdk"`
	Go        string `json:"go" yaml:"go"`
}

func TestGetVersionCmdMetadata(t *testing.T) {
	state := &appState{viper: viper.New()}
	cmd := getVersionCmd(state)

	require.Equal(t, "version", cmd.Use)
	require.Equal(t, []string{"v"}, cmd.Aliases)
	require.Equal(t, "Print the relayer version info", cmd.Short)
	require.Equal(t, "$ rly version --json\n$ rly v", cmd.Example)
	require.NotNil(t, cmd.Args)
	require.NotNil(t, cmd.RunE)
	require.NoError(t, cmd.ValidateArgs(nil))

	jsonFlag := cmd.Flags().Lookup(flagJSON)
	require.NotNil(t, jsonFlag)
	require.Equal(t, "j", jsonFlag.Shorthand)
	require.Equal(t, "false", jsonFlag.DefValue)
	require.Equal(t, "true", jsonFlag.NoOptDefVal)
	require.Equal(t, "bool", jsonFlag.Value.Type())
	require.Equal(t, "returns the response in json format", jsonFlag.Usage)
	require.Nil(t, cmd.Flags().Lookup(flagYAML), "YAML is the default output, not an accepted flag")

	require.NoError(t, cmd.Flags().Set(flagJSON, "true"))
	require.True(t, state.viper.GetBool(flagJSON), "the JSON flag must remain bound to appState.viper")
}

func TestGetVersionCmdSerializesVersionInfo(t *testing.T) {
	setCharacterizedVersionGlobals(t, "v9.8.7", "abc123", "0")

	tests := []struct {
		name       string
		args       []string
		jsonOutput bool
	}{
		{
			name: "default YAML",
		},
		{
			name:       "long JSON flag",
			args:       []string{"--json"},
			jsonOutput: true,
		},
		{
			name:       "short JSON flag",
			args:       []string{"-j"},
			jsonOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCharacterizedVersionCommand(tt.args...)
			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Equal(t, expectedCharacterizedVersionOutput(t, tt.jsonOutput, "abc123"), stdout)
		})
	}
}

func TestGetVersionCmdCommitDirtyMarker(t *testing.T) {
	tests := []struct {
		name       string
		dirty      string
		wantCommit string
	}{
		{
			name:       "literal zero is clean",
			dirty:      "0",
			wantCommit: "abc123",
		},
		{
			name:       "empty value is dirty",
			dirty:      "",
			wantCommit: "abc123 (dirty)",
		},
		{
			name:       "one is dirty",
			dirty:      "1",
			wantCommit: "abc123 (dirty)",
		},
		{
			name:       "other values are dirty",
			dirty:      "false",
			wantCommit: "abc123 (dirty)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCharacterizedVersionGlobals(t, "v9.8.7", "abc123", tt.dirty)

			stdout, stderr, err := executeCharacterizedVersionCommand("--json")
			require.NoError(t, err)
			require.Empty(t, stderr)

			var got characterizedVersionInfo
			require.NoError(t, json.Unmarshal([]byte(stdout), &got))
			require.Equal(t, tt.wantCommit, got.Commit)
		})
	}
}

func TestGetVersionCmdAliasExecutes(t *testing.T) {
	setCharacterizedVersionGlobals(t, "v9.8.7", "abc123", "0")

	state := &appState{viper: viper.New()}
	versionCmd := getVersionCmd(state)
	rootCmd := &cobra.Command{
		Use:           "rly",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetArgs([]string{"v", "--json"})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	require.NoError(t, rootCmd.Execute())
	require.Empty(t, stderr.String())
	require.Equal(t, "v", versionCmd.CalledAs())
	require.Equal(t, expectedCharacterizedVersionOutput(t, true, "abc123"), stdout.String())
}

func TestGetVersionCmdRejectsArgumentsWithUsage(t *testing.T) {
	state := &appState{viper: viper.New()}
	versionCmd := getVersionCmd(state)
	rootCmd := &cobra.Command{
		Use:          "rly",
		SilenceUsage: true,
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetArgs([]string{"version", "unexpected"})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	require.EqualError(t, err, `unknown command "unexpected" for "rly version"`)
	require.Equal(t, "Error: unknown command \"unexpected\" for \"rly version\"\n", stderr.String())
	require.Contains(t, stdout.String(), "Usage:\n  rly version [flags]")
	require.NotContains(t, stdout.String(), "cosmos-sdk:")
	require.False(t, rootCmd.SilenceUsage)
	require.False(t, versionCmd.SilenceUsage)
}

func TestGetVersionCmdRejectsYAMLFlag(t *testing.T) {
	stdout, stderr, err := executeCharacterizedVersionCommand("--yaml")

	require.EqualError(t, err, "unknown flag: --yaml")
	require.Empty(t, stdout)
	require.Empty(t, stderr)
}

func TestGetVersionCmdIgnoresOutputWriterErrors(t *testing.T) {
	setCharacterizedVersionGlobals(t, "v9.8.7", "abc123", "0")

	cmd := getVersionCmd(&appState{viper: viper.New()})
	cmd.SetArgs(make([]string, 0))
	cmd.SetOut(characterizedFailingWriter{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	require.NoError(t, cmd.Execute())
}

type characterizedFailingWriter struct{}

func (characterizedFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("characterized output failure")
}

func executeCharacterizedVersionCommand(args ...string) (stdout, stderr string, err error) {
	cmd := getVersionCmd(&appState{viper: viper.New()})
	commandArgs := make([]string, len(args))
	copy(commandArgs, args)
	cmd.SetArgs(commandArgs)

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.SetOut(&stdoutBuffer)
	cmd.SetErr(&stderrBuffer)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err = cmd.Execute()
	return stdoutBuffer.String(), stderrBuffer.String(), err
}

func setCharacterizedVersionGlobals(t *testing.T, version, commit, dirty string) {
	t.Helper()

	previousVersion, previousCommit, previousDirty := Version, Commit, Dirty
	Version, Commit, Dirty = version, commit, dirty
	t.Cleanup(func() {
		Version, Commit, Dirty = previousVersion, previousCommit, previousDirty
	})
}

func expectedCharacterizedVersionOutput(t *testing.T, jsonOutput bool, commit string) string {
	t.Helper()

	info := characterizedVersionInfo{
		Version:   Version,
		Commit:    commit,
		CosmosSDK: characterizedCosmosSDKVersion(),
		Go:        runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH,
	}

	var (
		output []byte
		err    error
	)
	if jsonOutput {
		output, err = json.Marshal(info)
	} else {
		output, err = yaml.Marshal(info)
	}
	require.NoError(t, err)

	return string(output) + "\n"
}

func characterizedCosmosSDKVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unable to determine)"
	}

	for _, dependency := range buildInfo.Deps {
		if dependency.Path == "github.com/cosmos/cosmos-sdk" {
			return dependency.Version
		}
	}

	return "(unable to determine)"
}
