package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/relayer/v2/cregistry"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	check = "✔"
	xIcon = "✘"
)

func chainsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chains",
		Aliases: []string{"ch"},
		Short:   "Manage chain configurations",
	}

	cmd.AddCommand(
		chainsListCmd(a),
		chainsRegistryList(a),
		chainsDeleteCmd(a),
		chainsAddCmd(a),
		chainsShowCmd(a),
		chainsAddrCmd(a),
		chainsAddDirCmd(a),
		cmdChainsConfigure(a),
		cmdChainsUseRpcAddr(a),
		cmdChainsUseBackupRpcAddr(a),
	)

	return cmd
}

func cmdChainsUseRpcAddr(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-rpc-addr chain_name valid_rpc_url",
		Aliases: []string{"rpc"},
		Short:   "Sets chain's rpc address",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains set-rpc-addr ibc-0 https://abc.xyz.com:443
$ %s ch set-rpc-addr ibc-0 https://abc.xyz.com:443`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainName := args[0]
			rpc_address := args[1]
			if !isValidURL(rpc_address) {
				return invalidRpcAddr(rpc_address)
			}

			return a.useRpcAddr(chainName, rpc_address)
		},
	}

	return cmd
}

func cmdChainsUseBackupRpcAddr(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-backup-rpc-addrs chain_name comma_separated_valid_rpc_urls",
		Aliases: []string{"set-backup-rpcs"},
		Short:   "Sets chain's backup rpc addresses",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains set-backup-rpc-addr ibc-0 https://abc.xyz.com:443,https://123.456.com:443
$ %s ch set-backup-rpc-addr ibc-0 https://abc.xyz.com:443,https://123.456.com:443`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainName := args[0]
			rpc_addresses := args[1]

			// split rpc_addresses by ','
			rpc_addresses_list := strings.Split(rpc_addresses, ",")

			// loop through and ensure valid
			for _, rpc_address := range rpc_addresses_list {
				rpc_address := rpc_address
				if !isValidURL(rpc_address) {
					return invalidRpcAddr(rpc_address)
				}
			}

			return a.useBackupRpcAddrs(chainName, rpc_addresses_list)
		},
	}

	return cmd
}

func chainsAddrCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "address chain_name",
		Aliases: []string{"addr"},
		Short:   "Returns a chain's configured key's address",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains address ibc-0
$ %s ch addr ibc-0`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			address, err := chain.ChainProvider.ShowAddress(chain.ChainProvider.Key())
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), address)
			return nil
		},
	}

	return cmd
}

func chainsShowCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show chain_name",
		Aliases: []string{"s"},
		Short:   "Returns a chain's configuration data",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains show ibc-0 --json
$ %s chains show ibc-0 --yaml
$ %s ch s ibc-0 --json
$ %s ch s ibc-0 --yaml`, appName, appName, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChainsShow(cmd, a, args[0])
		},
	}
	return jsonFlag(a.viper, cmd)
}

func runChainsShow(cmd *cobra.Command, a *appState, chainName string) error {
	c, ok := a.config.Chains[chainName]
	if !ok {
		return errChainNotFound(chainName)
	}

	jsn, err := cmd.Flags().GetBool(flagJSON)
	if err != nil {
		return err
	}

	pcfgw := &ProviderConfigWrapper{
		Type:  c.ChainProvider.Type(),
		Value: c.ChainProvider.ProviderConfig(),
	}
	out, err := marshalProviderConfig(pcfgw, jsn)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}

func marshalProviderConfig(config *ProviderConfigWrapper, jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		return json.Marshal(config)
	}
	return yaml.Marshal(config)
}

func chainsDeleteCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete chain_name",
		Aliases: []string{"d"},
		Short:   "Removes chain from config based off chain-id",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains delete ibc-0
$ %s ch d ibc-0`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain := args[0]
			return a.performConfigLockingOperation(cmd.Context(), func() error {
				_, ok := a.config.Chains[chain]
				if !ok {
					return errChainNotFound(chain)
				}
				a.config.DeleteChain(chain)
				return nil
			})
		},
	}
	return cmd
}

func cmdChainsConfigure(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "manage local chain configurations",
	}

	cmd.AddCommand(
		feegrantConfigureBaseCmd(a),
	)

	return cmd
}

func chainsRegistryList(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry-list",
		Args:    withUsage(cobra.NoArgs),
		Aliases: []string{"rl"},
		Short:   "List chains available for configuration from the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChainsRegistryList(cmd, a)
		},
	}
	return yamlFlag(a.viper, jsonFlag(a.viper, cmd))
}

func runChainsRegistryList(cmd *cobra.Command, a *appState) error {
	jsn, err := cmd.Flags().GetBool(flagJSON)
	if err != nil {
		return err
	}

	yml, err := cmd.Flags().GetBool(flagYAML)
	if err != nil {
		return err
	}

	chains, err := cregistry.DefaultChainRegistry(a.log).ListChains(cmd.Context())
	if err != nil {
		return err
	}

	return writeRegistryChains(cmd, chains, jsn, yml)
}

func writeRegistryChains(cmd *cobra.Command, chains []string, jsn, yml bool) error {
	switch {
	case yml && jsn:
		return errors.New("can't pass both --json and --yaml, must pick one")
	case yml:
		out, err := yaml.Marshal(chains)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	case jsn:
		out, err := json.Marshal(chains)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	default:
		for _, chain := range chains {
			fmt.Fprintln(cmd.OutOrStdout(), chain)
		}
	}
	return nil
}

func chainsListCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "Returns chain configuration data",
		Args:    withUsage(cobra.NoArgs),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains list
$ %s ch l`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChainsList(cmd, a)
		},
	}
	return yamlFlag(a.viper, jsonFlag(a.viper, cmd))
}

func runChainsList(cmd *cobra.Command, a *appState) error {
	jsn, err := cmd.Flags().GetBool(flagJSON)
	if err != nil {
		return err
	}

	yml, err := cmd.Flags().GetBool(flagYAML)
	if err != nil {
		return err
	}

	configs := a.config.Wrapped().ProviderConfigs
	if len(configs) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: no chains found (do you need to run 'rly chains add'?)")
	}

	return writeChainConfigs(cmd, a, configs, jsn, yml)
}

func writeChainConfigs(cmd *cobra.Command, a *appState, configs ProviderConfigs, jsn, yml bool) error {
	switch {
	case yml && jsn:
		return errors.New("can't pass both --json and --yaml, must pick one")
	case yml:
		out, err := yaml.Marshal(configs)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	case jsn:
		out, err := json.Marshal(configs)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	default:
		writePlainChainList(cmd, a)
		return nil
	}
}

func writePlainChainList(cmd *cobra.Command, a *appState) {
	i := 0
	for _, c := range a.config.Chains {
		key, bal, path := chainListStatus(cmd, a, c)
		i++
		fmt.Fprintf(cmd.OutOrStdout(), "%2d: %-20s -> type(%s) key(%s) bal(%s) path(%s)\n", i, c.ChainID(), c.ChainProvider.Type(), key, bal, path)
	}
}

func chainListStatus(cmd *cobra.Command, a *appState, c *relayer.Chain) (key, bal, path string) {
	key = xIcon
	bal = xIcon
	path = xIcon

	// check that the key from config.yaml is set in keychain
	if c.ChainProvider.KeyExists(c.ChainProvider.Key()) {
		key = check
	}

	coins, err := c.ChainProvider.QueryBalance(cmd.Context(), c.ChainProvider.Key())
	if err == nil && !coins.Empty() {
		bal = check
	}

	for _, pth := range a.config.Paths {
		if pth.Src.ChainID == c.ChainProvider.ChainId() || pth.Dst.ChainID == c.ChainID() {
			path = check
		}
	}
	return key, bal, path
}

func chainsAddCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [chain-name...]",
		Aliases: []string{"a"},
		Short: "Add a new chain to the configuration file by fetching chain metadata from \n" +
			"                the chain-registry or passing a file (-f) or url (-u)",
		Args: withUsage(cobra.MinimumNArgs(0)),
		Example: fmt.Sprintf(` $ %s chains add cosmoshub
 $ %s chains add cosmoshub osmosis
 $ %s chains add cosmoshubtestnet --testnet
 $ %s chains add --file chains/ibc0.json ibc0
 $ %s chains add --url https://relayer.com/ibc0.json ibc0`, appName, appName, appName, appName, appName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChainsAdd(cmd, a, args)
		},
	}

	return chainsAddFlags(a.viper, cmd)
}

func runChainsAdd(cmd *cobra.Command, a *appState, args []string) error {
	file, rawURL, forceAdd, testnet, err := getAddInputs(cmd)
	if err != nil {
		return err
	}

	if ok := a.config; ok == nil {
		return errors.New("config not initialized, consider running `rly config init`")
	}

	return a.performConfigLockingOperation(cmd.Context(), func() error {
		return addChainFromInput(cmd, a, args, file, rawURL, forceAdd, testnet)
	})
}

func addChainFromInput(
	cmd *cobra.Command,
	a *appState,
	args []string,
	file, rawURL string,
	forceAdd, testnet bool,
) error {
	// Default behavior fetches from the chain registry while still allowing a
	// config to be added from a URL or file.
	switch {
	case file != "":
		chainName, err := chainNameFromFile(args, file)
		if err != nil {
			return err
		}
		return addChainFromFile(a, chainName, file)
	case rawURL != "":
		if len(args) != 1 {
			return errors.New("one chain name is required")
		}
		return addChainFromURL(a, args[0], rawURL)
	default:
		return addChainsFromRegistry(cmd.Context(), a, forceAdd, testnet, args)
	}
}

func chainNameFromFile(args []string, file string) (string, error) {
	switch len(args) {
	case 0:
		return strings.Split(filepath.Base(file), ".")[0], nil
	case 1:
		return args[0], nil
	default:
		return "", errors.New("one chain name is required")
	}
}

func chainsAddDirCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-dir dir",
		Aliases: []string{"ad"},
		Args:    withUsage(cobra.ExactArgs(1)),
		Short:   `Add chain configuration data in bulk from a directory. Example dir: 'configs/demo/chains'`,
		Long: `Add chain configuration data in bulk from a directory housing individual chain config files. This is useful for spinning up testnets.
		
		See 'configs/demo/chains' for an example of individual chain config files.`,
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s chains add-dir configs/demo/chains
$ %s ch ad testnet/chains/`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return addChainsFromDirectory(cmd.Context(), cmd.ErrOrStderr(), a, args[0])
		},
	}

	return cmd
}

// addChainFromFile reads a JSON-formatted chain from the named file
// and adds it to a's chains.
func addChainFromFile(a *appState, chainName string, file string) error {
	// If the user passes in a file, attempt to read the chain config from that file
	var pcw ProviderConfigWrapper
	if _, err := os.Stat(file); err != nil {
		return err
	}

	byt, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(byt, &pcw); err != nil {
		return err
	}

	prov, err := pcw.Value.NewProvider(
		a.log.With(zap.String("provider_type", pcw.Type)),
		a.homePath, a.debug, chainName,
	)
	if err != nil {
		return fmt.Errorf("failed to build ChainProvider for %s: %w", file, err)
	}

	c := relayer.NewChain(a.log, prov, a.debug)
	if err = a.config.AddChain(c); err != nil {
		return err
	}

	return nil
}

// addChainFromURL fetches a JSON-encoded chain from the given URL
// and adds it to a's chains.
func addChainFromURL(a *appState, chainName string, rawurl string) error {
	u, err := url.Parse(rawurl)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid URL %s", rawurl)
	}

	// TODO: add a rly user agent to this outgoing request.
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var pcw ProviderConfigWrapper
	d := json.NewDecoder(resp.Body)
	d.DisallowUnknownFields()
	err = d.Decode(&pcw)
	if err != nil {
		return err
	}

	// build the ChainProvider before initializing the chain
	prov, err := pcw.Value.NewProvider(
		a.log.With(zap.String("provider_type", pcw.Type)),
		a.homePath, a.debug, chainName,
	)
	if err != nil {
		return fmt.Errorf("failed to build ChainProvider for %s: %w", rawurl, err)
	}

	c := relayer.NewChain(a.log, prov, a.debug)
	if err := a.config.AddChain(c); err != nil {
		return err
	}
	return nil
}

func addChainsFromRegistry(ctx context.Context, a *appState, forceAdd, testnet bool, chains []string) error {
	chainRegistry := cregistry.DefaultChainRegistry(a.log)

	var existed, failed, added []string

	for _, chain := range chains {
		switch addChainFromRegistry(ctx, a, chainRegistry, forceAdd, testnet, chain) {
		case registryChainExisted:
			existed = append(existed, chain)
		case registryChainFailed:
			failed = append(failed, chain)
		case registryChainAdded:
			added = append(added, chain)
		}
	}
	a.log.Info("Config update status",
		zap.Any("added", added),
		zap.Any("failed", failed),
		zap.Any("already existed", existed),
	)
	return nil
}

type registryChainResult uint8

const (
	registryChainExisted registryChainResult = iota
	registryChainFailed
	registryChainAdded
)

func addChainFromRegistry(
	ctx context.Context,
	a *appState,
	chainRegistry cregistry.ChainRegistry,
	forceAdd, testnet bool,
	chain string,
) registryChainResult {
	if _, ok := a.config.Chains[chain]; ok {
		a.log.Warn(
			"Chain already exists",
			zap.String("chain", chain),
			zap.String("source_link", chainRegistry.SourceLink()),
		)
		return registryChainExisted
	}

	chainInfo, err := chainRegistry.GetChain(ctx, testnet, chain)
	if err != nil {
		a.log.Warn(
			"Error retrieving chain",
			zap.String("chain", chain),
			zap.Error(err),
		)
		return registryChainFailed
	}

	chainConfig, err := chainInfo.GetChainConfig(ctx, forceAdd, testnet, chain)
	if err != nil {
		a.log.Warn(
			"Error generating chain config",
			zap.String("chain", chain),
			zap.Error(err),
		)
		return registryChainFailed
	}
	chainConfig.ChainName = chainInfo.ChainName
	chainConfig.Broadcast = provider.BroadcastModeBatch

	// build the ChainProvider
	prov, err := chainConfig.NewProvider(
		a.log.With(zap.String("provider_type", "cosmos")),
		a.homePath, a.debug, chainInfo.ChainName,
	)
	if err != nil {
		a.log.Warn(
			"Failed to build ChainProvider",
			zap.String("chain_id", chainConfig.ChainID),
			zap.Error(err),
		)
		return registryChainFailed
	}

	// add to config
	c := relayer.NewChain(a.log, prov, a.debug)
	if err = a.config.AddChain(c); err != nil {
		a.log.Warn(
			"Failed to add chain to config",
			zap.String("chain", chain),
			zap.Error(err),
		)
		return registryChainFailed
	}

	return registryChainAdded
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
