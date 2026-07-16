package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/google/go-github/v43/github"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func pathsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "paths",
		Aliases: []string{"pth"},
		Short:   "Manage path configurations",
		Long: `
A path represents the "full path" or "link" for communication between two chains. 
This includes the client, connection, and channel ids from both the source and destination chains as well as the strategy to use when relaying`,
	}

	cmd.AddCommand(
		pathsListCmd(a),
		pathsShowCmd(a),
		pathsAddCmd(a),
		pathsAddDirCmd(a),
		pathsNewCmd(a),
		pathsUpdateCmd(a),
		pathsFetchCmd(a),
		pathsDeleteCmd(a),
	)

	return cmd
}

func pathsDeleteCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete index",
		Aliases: []string{"d"},
		Short:   "Delete a path with a given index",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths delete demo-path
$ %s pth d path-name`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.performConfigLockingOperation(cmd.Context(), func() error {
				if _, err := a.config.Paths.Get(args[0]); err != nil {
					return err
				}
				delete(a.config.Paths, args[0])
				return nil
			})
		},
	}
	return cmd
}

func pathsListCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "Print out configured paths",
		Args:    withUsage(cobra.NoArgs),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths list --yaml
$ %s paths list --json
$ %s pth l`, appName, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPathsList(cmd, a)
		},
	}
	return yamlFlag(a.viper, jsonFlag(a.viper, cmd))
}

func runPathsList(cmd *cobra.Command, a *appState) error {
	jsn, _ := cmd.Flags().GetBool(flagJSON)
	yml, _ := cmd.Flags().GetBool(flagYAML)
	return writePathsList(cmd, a, jsn, yml)
}

func writePathsList(cmd *cobra.Command, a *appState, jsn, yml bool) error {
	switch {
	case yml && jsn:
		return errors.New("can't pass both --json and --yaml, must pick one")
	case yml:
		out, err := yaml.Marshal(a.config.Paths)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	case jsn:
		out, err := json.Marshal(a.config.Paths)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	default:
		return writePlainPathsList(cmd, a)
	}
}

func writePlainPathsList(cmd *cobra.Command, a *appState) error {
	i := 0
	for k, pth := range a.config.Paths {
		chains, err := a.config.Chains.Gets(pth.Src.ChainID, pth.Dst.ChainID)
		if err != nil {
			return err
		}
		stat := pth.QueryPathStatus(cmd.Context(), chains[pth.Src.ChainID], chains[pth.Dst.ChainID]).Status

		printPath(cmd.OutOrStdout(), i, k, pth, checkmark(stat.Chains), checkmark(stat.Clients),
			checkmark(stat.Connection))

		i++
	}
	return nil
}

func printPath(stdout io.Writer, i int, k string, pth *relayer.Path, chains, clients, connection string) {
	fmt.Fprintf(stdout, "%2d: %-20s -> chns(%s) clnts(%s) conn(%s) (%s<>%s)\n",
		i, k, chains, clients, connection, pth.Src.ChainID, pth.Dst.ChainID)
}

func checkmark(status bool) string {
	if status {
		return check
	}
	return xIcon
}

func pathsShowCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show path_name",
		Aliases: []string{"s"},
		Short:   "Show a path given its name",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths show demo-path --yaml
$ %s paths show demo-path --json
$ %s pth s path-name`, appName, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPathsShow(cmd, a, args[0])
		},
	}
	return yamlFlag(a.viper, jsonFlag(a.viper, cmd))
}

func runPathsShow(cmd *cobra.Command, a *appState, pathName string) error {
	p, err := a.config.Paths.Get(pathName)
	if err != nil {
		return err
	}
	chains, err := a.config.Chains.Gets(p.Src.ChainID, p.Dst.ChainID)
	if err != nil {
		return err
	}
	jsn, _ := cmd.Flags().GetBool(flagJSON)
	yml, _ := cmd.Flags().GetBool(flagYAML)
	pathWithStatus := p.QueryPathStatus(cmd.Context(), chains[p.Src.ChainID], chains[p.Dst.ChainID])
	return writePathWithStatus(cmd, pathName, pathWithStatus, jsn, yml)
}

func writePathWithStatus(cmd *cobra.Command, pathName string, pathWithStatus *relayer.PathWithStatus, jsn, yml bool) error {
	switch {
	case yml && jsn:
		return errors.New("can't pass both --json and --yaml, must pick one")
	case yml:
		out, err := yaml.Marshal(pathWithStatus)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	case jsn:
		out, err := json.Marshal(pathWithStatus)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	default:
		fmt.Fprintln(cmd.OutOrStdout(), pathWithStatus.PrintString(pathName))
		return nil
	}
}

func pathsAddCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add src_chain_id dst_chain_id path_name",
		Aliases: []string{"a"},
		Short:   "Add a path to the list of paths",
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths add ibc-0 ibc-1 demo-path
$ %s paths add ibc-0 ibc-1 demo-path --file paths/demo.json
$ %s pth a ibc-0 ibc-1 demo-path`, appName, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPathsAdd(cmd, a, args)
		},
	}
	return fileFlag(a.viper, cmd)
}

func runPathsAdd(cmd *cobra.Command, a *appState, args []string) error {
	src, dst := args[0], args[1]
	return a.performConfigLockingOperation(cmd.Context(), func() error {
		return addPath(cmd, a, src, dst, args[2])
	})
}

func addPath(cmd *cobra.Command, a *appState, src, dst, pathName string) error {
	_, err := a.config.Chains.Gets(src, dst)
	if err != nil {
		return fmt.Errorf("chains need to be configured before paths to them can be added: %w", err)
	}

	file, err := cmd.Flags().GetString(flagFile)
	if err != nil {
		return err
	}

	if file != "" {
		return a.addPathFromFile(cmd.Context(), cmd.ErrOrStderr(), file, pathName)
	}
	return a.addPathFromUserInput(cmd.Context(), cmd.InOrStdin(), cmd.ErrOrStderr(), src, dst, pathName)
}

func pathsAddDirCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-dir dir",
		Args:  withUsage(cobra.ExactArgs(1)),
		Short: `Add path configuration data in bulk from a directory. Example dir: 'configs/demo/paths'`,
		Long: `Add path configuration data in bulk from a directory housing individual path config files. This is useful for spinning up testnets.
		
		See 'examples/demo/configs/paths' for an example of individual path config files.`,
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s config add-paths examples/demo/configs/paths`, appName)),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return addPathsFromDirectory(cmd.Context(), cmd.ErrOrStderr(), a, args[0])
		},
	}

	return cmd
}

func pathsNewCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new src_chain_id dst_chain_id path_name",
		Aliases: []string{"n"},
		Short:   "Create a new blank path to be used in generating a new path (connection & client) between two chains",
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths new ibc-0 ibc-1 demo-path
$ %s pth n ibc-0 ibc-1 demo-path`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, dst := args[0], args[1]

			return a.performConfigLockingOperation(cmd.Context(), func() error {
				_, err := a.config.Chains.Gets(src, dst)
				if err != nil {
					return fmt.Errorf("chains need to be configured before paths to them can be added: %w", err)
				}

				p := &relayer.Path{
					Src: &relayer.PathEnd{ChainID: src},
					Dst: &relayer.PathEnd{ChainID: dst},
				}

				name := args[2]
				if err = a.config.AddPath(name, p); err != nil {
					return err
				}
				return nil
			})
		},
	}
	return channelParameterFlags(a.viper, cmd)
}

func pathsUpdateCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update path_name",
		Aliases: []string{"n"},
		Short:   `Update a path such as the filter rule ("allowlist", "denylist", or "" for no filtering), filter channels, and src/dst chain, client, or connection IDs`,
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths update demo-path --filter-rule allowlist --filter-channels channel-0,channel-1
$ %s paths update demo-path --filter-rule denylist --filter-channels channel-0,channel-1
$ %s paths update demo-path --src-chain-id chain-1 --dst-chain-id chain-2
$ %s paths update demo-path --src-client-id 07-tendermint-02 --dst-client-id 07-tendermint-04
$ %s paths update demo-path --src-connection-id connection-02 --dst-connection-id connection-04`,
			appName, appName, appName, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPathsUpdate(cmd, a, args[0])
		},
	}
	cmd = pathFilterFlags(a.viper, cmd)
	return cmd
}

func runPathsUpdate(cmd *cobra.Command, a *appState, pathName string) error {
	flags := cmd.Flags()
	return a.performConfigLockingOperation(cmd.Context(), func() error {
		p := a.config.Paths.MustGet(pathName)
		return updatePathFromFlags(p, flags)
	})
}

func updatePathFromFlags(p *relayer.Path, flags *pflag.FlagSet) error {
	actionTaken, err := updatePathFilterRule(p, flags)
	if err != nil {
		return err
	}

	if updatePathFilterChannels(p, flags) {
		actionTaken = true
	}
	if updatePathEndField(p, flags, flagSrcChainID) {
		actionTaken = true
	}
	if updatePathEndField(p, flags, flagDstChainID) {
		actionTaken = true
	}
	if updatePathEndField(p, flags, flagSrcClientID) {
		actionTaken = true
	}
	if updatePathEndField(p, flags, flagDstClientID) {
		actionTaken = true
	}
	if updatePathEndField(p, flags, flagSrcConnID) {
		actionTaken = true
	}
	if updatePathEndField(p, flags, flagDstConnID) {
		actionTaken = true
	}

	if !actionTaken {
		return errors.New("at least one flag must be provided")
	}
	return nil
}

func updatePathFilterRule(p *relayer.Path, flags *pflag.FlagSet) (bool, error) {
	filterRule, _ := flags.GetString(flagFilterRule)
	if filterRule == blankValue {
		return false, nil
	}
	if filterRule != "" && filterRule != processor.RuleAllowList && filterRule != processor.RuleDenyList {
		return false, fmt.Errorf(
			`invalid filter rule : "%s". valid rules: ("", "%s", "%s")`,
			filterRule, processor.RuleAllowList, processor.RuleDenyList)
	}
	p.Filter.Rule = filterRule
	return true, nil
}

func updatePathFilterChannels(p *relayer.Path, flags *pflag.FlagSet) bool {
	filterChannels, _ := flags.GetString(flagFilterChannels)
	if filterChannels == blankValue {
		return false
	}

	var channelList []string
	if filterChannels != "" {
		channelList = strings.Split(filterChannels, ",")
	}
	p.Filter.ChannelList = channelList
	return true
}

func updatePathEndField(p *relayer.Path, flags *pflag.FlagSet, flagName string) bool {
	value, _ := flags.GetString(flagName)
	if value == "" {
		return false
	}

	switch flagName {
	case flagSrcChainID:
		p.Src.ChainID = value
	case flagDstChainID:
		p.Dst.ChainID = value
	case flagSrcClientID:
		p.Src.ClientID = value
	case flagDstClientID:
		p.Dst.ClientID = value
	case flagSrcConnID:
		p.Src.ConnectionID = value
	case flagDstConnID:
		p.Dst.ConnectionID = value
	}
	return true
}

// pathsFetchCmd attempts to fetch the json files containing the path metadata, for each configured chain, from GitHub
func pathsFetchCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fetch",
		Aliases: []string{"fch"},
		Short:   "Fetches the json files necessary to setup the paths for the configured chains. Passing a chain name will only fetch paths for that chain",
		Args:    withUsage(cobra.RangeArgs(0, 1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s paths fetch --home %s
$ %s paths fetch --testnet
$ %s pth fch`, appName, defaultHome, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPathsFetch(cmd, a, args)
		},
	}
	OverwriteConfigFlag(a.viper, cmd)
	testnetFlag(a.viper, cmd)
	return cmd
}

func runPathsFetch(cmd *cobra.Command, a *appState, args []string) error {
	overwrite, _ := cmd.Flags().GetBool(flagOverwriteConfig)
	testnet, _ := cmd.Flags().GetBool(flagTestnet)

	chainReq, err := requestedPathChain(a, args)
	if err != nil {
		return err
	}

	return a.performConfigLockingOperation(cmd.Context(), func() error {
		return fetchPaths(cmd, a, chainReq, overwrite, testnet)
	})
}

func requestedPathChain(a *appState, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	chainReq := args[0]
	if _, exist := a.config.Chains[chainReq]; !exist {
		return "", fmt.Errorf("chain %s not found in config", chainReq)
	}
	return chainReq, nil
}

func fetchPaths(cmd *cobra.Command, a *appState, chainReq string, overwrite, testnet bool) error {
	chainCombinations := configuredChainCombinations(a, chainReq)
	githubClient := github.NewClient(nil)

	var deferredClosers []io.ReadCloser
	defer func() {
		closePathReaders(deferredClosers)
	}()

	for pthName := range chainCombinations {
		if skipFetchedPath(cmd, a, pthName, overwrite) {
			continue
		}

		reader, result := downloadRegistryPath(cmd, githubClient, pthName, testnet)
		switch result {
		case pathDownloadStop:
			return nil
		case pathDownloadFailed:
			continue
		case pathDownloadReady:
			deferredClosers = append(deferredClosers, reader)
		}

		if err := addDownloadedPath(cmd, a, pthName, reader); err != nil {
			return err
		}
	}
	return nil
}

func configuredChainCombinations(a *appState, chainReq string) map[string]bool {
	chains := make([]string, 0, len(a.config.Chains))
	for chainName := range a.config.Chains {
		chains = append(chains, chainName)
	}

	chainCombinations := make(map[string]bool)
	for _, chainA := range chains {
		for _, chainB := range chains {
			addChainCombination(chainCombinations, chainReq, chainA, chainB)
		}
	}
	return chainCombinations
}

func addChainCombination(combinations map[string]bool, chainReq, chainA, chainB string) {
	if chainA == chainB {
		return
	}

	pair := chainA + "-" + chainB
	if chainB < chainA {
		pair = chainB + "-" + chainA
	}
	if chainReq != "" && !strings.Contains(pair, chainReq) {
		return
	}
	combinations[pair] = true
}

func skipFetchedPath(cmd *cobra.Command, a *appState, pathName string, overwrite bool) bool {
	_, exist := a.config.Paths[pathName]
	if exist && !overwrite {
		fmt.Fprintf(cmd.ErrOrStderr(), "skipping:  %s already exists in config, use -o to overwrite (clears filters)\n", pathName)
		return true
	}
	return false
}

type pathDownloadResult uint8

const (
	pathDownloadReady pathDownloadResult = iota
	pathDownloadFailed
	pathDownloadStop
)

func downloadRegistryPath(
	cmd *cobra.Command,
	client *github.Client,
	pathName string,
	testnet bool,
) (io.ReadCloser, pathDownloadResult) {
	// TODO: Don't use github api. Potentially use http.get like GetChain() does to avoid rate limits
	regPath := registryPath(pathName, testnet)
	reader, _, err := client.Repositories.DownloadContents(cmd.Context(), "cosmos", "chain-registry", regPath, nil)
	if err != nil {
		if errors.As(err, new(*github.RateLimitError)) {
			fmt.Println("some paths failed: ", err)
			return nil, pathDownloadStop
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "failure retrieving: %s: consider adding to cosmos/chain-registry: ERR: %v\n", pathName, err)
		return nil, pathDownloadFailed
	}
	return reader, pathDownloadReady
}

func registryPath(pathName string, testnet bool) string {
	fileName := pathName + ".json"
	if testnet {
		return path.Join("testnets", "_IBC", fileName)
	}
	return path.Join("_IBC", fileName)
}

func addDownloadedPath(cmd *cobra.Command, a *appState, pathName string, reader io.ReadCloser) error {
	b, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	ibc := &relayer.IBCdata{}
	if err = json.Unmarshal(b, &ibc); err != nil {
		return fmt.Errorf("failed to unmarshal: %w ", err)
	}

	srcChainName := ibc.Chain1.ChainName
	dstChainName := ibc.Chain2.ChainName
	newPath := &relayer.Path{
		Src: &relayer.PathEnd{
			ChainID:      a.config.Chains[srcChainName].ChainID(),
			ClientID:     ibc.Chain1.ClientID,
			ConnectionID: ibc.Chain1.ConnectionID,
		},
		Dst: &relayer.PathEnd{
			ChainID:      a.config.Chains[dstChainName].ChainID(),
			ClientID:     ibc.Chain2.ClientID,
			ConnectionID: ibc.Chain2.ConnectionID,
		},
	}
	reader.Close()

	if err = a.config.AddPath(pathName, newPath); err != nil {
		return fmt.Errorf("failed to add path %s: %w", pathName, err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "added:  %s\n", pathName)
	return nil
}

func closePathReaders(readers []io.ReadCloser) {
	for i := len(readers) - 1; i >= 0; i-- {
		readers[i].Close()
	}
}
