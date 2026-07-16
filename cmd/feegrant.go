package cmd

import (
	"context"
	"errors"
	"fmt"

	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// feegrantConfigureCmd returns the fee grant configuration commands for this module
func feegrantConfigureBaseCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feegrant",
		Short: "Configure the client to use round-robin feegranted accounts when sending TXs",
		Long:  "Use round-robin feegranted accounts when sending TXs. Useful for relayers and applications where sequencing is important",
	}

	cmd.AddCommand(
		feegrantConfigureBasicCmd(a),
	)

	return cmd
}

func feegrantConfigureBasicCmd(a *appState) *cobra.Command {
	var numGrantees int
	var update bool
	var delete bool
	var updateGrantees bool
	var grantees []string

	cmd := &cobra.Command{
		Use:   "basicallowance [chain-name] [granter] --num-grantees [int] --overwrite-granter --overwrite-grantees",
		Short: "feegrants for the given chain and granter (if granter is unspecified, use the default key)",
		Long:  "feegrants for the given chain. 10 grantees by default, all with an unrestricted BasicAllowance.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeegrantConfigureBasic(cmd, a, args, feegrantConfigureOptions{
				numGrantees:       numGrantees,
				overwriteGranter:  update,
				delete:            delete,
				overwriteGrantees: updateGrantees,
				grantees:          grantees,
			})
		},
	}

	cmd.Flags().BoolVar(&delete, "delete", false, "delete the feegrant configuration")
	cmd.Flags().BoolVar(&update, "overwrite-granter", false, "allow overwriting the existing granter")
	cmd.Flags().BoolVar(&updateGrantees, "overwrite-grantees", false, "allow overwriting existing grantees")
	cmd.Flags().IntVar(&numGrantees, "num-grantees", 10, "number of grantees that will be feegranted with basic allowances")
	cmd.Flags().StringSliceVar(&grantees, "grantees", nil, "comma separated list of grantee key names (keys are created if they do not exist)")
	cmd.MarkFlagsMutuallyExclusive("num-grantees", "grantees", "delete")
	cmd.Flags().String(sdkflags.FlagGas, "", fmt.Sprintf("gas limit to set per-transaction; set to %q to calculate sufficient gas automatically (default %d)", sdkflags.GasFlagAuto, sdkflags.DefaultGasLimit))

	memoFlag(a.viper, cmd)
	return cmd
}

type feegrantConfigureOptions struct {
	numGrantees       int
	overwriteGranter  bool
	delete            bool
	overwriteGrantees bool
	grantees          []string
}

type feegrantGranter struct {
	keyOrAddress string
	key          string
	external     bool
}

func runFeegrantConfigureBasic(
	cmd *cobra.Command,
	a *appState,
	args []string,
	options feegrantConfigureOptions,
) error {
	chain := args[0]
	prov, err := feegrantCosmosProvider(a, chain)
	if err != nil {
		return err
	}

	granter, err := resolveFeegrantGranter(prov, args)
	if err != nil {
		return err
	}

	if options.delete {
		return deleteFeegrantConfiguration(cmd, a, chain)
	}
	if err := overwriteFeegrantGranter(cmd, a, prov, granter, options.overwriteGranter); err != nil {
		return err
	}
	if err := configureFeegrantGrantees(cmd, a, chain, prov, granter, options); err != nil {
		return err
	}

	memo, err := cmd.Flags().GetString(flagMemo)
	if err != nil {
		return err
	}
	gas := feegrantGas(cmd)

	ctx := cmd.Context()
	if _, err = prov.EnsureBasicGrants(ctx, memo, gas); err != nil {
		return fmt.Errorf("error writing grants on chain: '%s'", err.Error())
	}

	return verifyFeegrantConfiguration(cmd, ctx, a, chain, prov, granter.external)
}

func feegrantCosmosProvider(a *appState, chain string) (*cosmos.CosmosProvider, error) {
	cosmosChain, ok := a.config.Chains[chain]
	if !ok {
		return nil, errChainNotFound(chain)
	}
	prov, ok := cosmosChain.ChainProvider.(*cosmos.CosmosProvider)
	if !ok {
		return nil, errors.New("only CosmosProvider can be feegranted")
	}
	return prov, nil
}

func resolveFeegrantGranter(prov *cosmos.CosmosProvider, args []string) (feegrantGranter, error) {
	granter := feegrantGranter{}
	switch {
	case len(args) > 1:
		granter.keyOrAddress = args[1]
	case prov.PCfg.FeeGrants != nil:
		granter.keyOrAddress = prov.PCfg.FeeGrants.GranterKeyOrAddr
	default:
		granter.keyOrAddress = prov.PCfg.Key
	}

	var err error
	granter.key, err = prov.KeyFromKeyOrAddress(granter.keyOrAddress)
	if err != nil {
		granter.external = true
	}
	if granter.external {
		if _, err := prov.DecodeBech32AccAddr(granter.keyOrAddress); err != nil {
			return feegrantGranter{}, fmt.Errorf("an unknown granter was specified: '%s' is not a valid bech32 address", granter.keyOrAddress)
		}
	}
	return granter, nil
}

func deleteFeegrantConfiguration(cmd *cobra.Command, a *appState, chain string) error {
	a.log.Info("Deleting feegrant configuration", zap.String("chain", chain))
	cfgErr := a.performConfigLockingOperation(cmd.Context(), func() error {
		chain := a.config.Chains[chain]
		oldProv := chain.ChainProvider.(*cosmos.CosmosProvider)
		oldProv.PCfg.FeeGrants = nil
		return nil
	})
	cobra.CheckErr(cfgErr)
	return nil
}

func overwriteFeegrantGranter(
	cmd *cobra.Command,
	a *appState,
	prov *cosmos.CosmosProvider,
	granter feegrantGranter,
	overwrite bool,
) error {
	configured := prov.PCfg.FeeGrants
	if configured == nil || granter.key == configured.GranterKeyOrAddr {
		return nil
	}
	if !overwrite {
		return fmt.Errorf("you specified granter '%s' which is different than configured feegranter '%s', but you did not specify the --overwrite-granter flag", granter.keyOrAddress, configured.GranterKeyOrAddr)
	}

	cfgErr := a.performConfigLockingOperation(cmd.Context(), func() error {
		prov.PCfg.FeeGrants.GranterKeyOrAddr = granter.key
		prov.PCfg.FeeGrants.IsExternalGranter = granter.external
		return nil
	})
	cobra.CheckErr(cfgErr)
	return nil
}

func configureFeegrantGrantees(
	cmd *cobra.Command,
	a *appState,
	chain string,
	prov *cosmos.CosmosProvider,
	granter feegrantGranter,
	options feegrantConfigureOptions,
) error {
	if prov.PCfg.FeeGrants != nil && !options.overwriteGrantees && len(options.grantees) == 0 {
		return nil
	}

	if err := setFeegrantGrantees(prov, granter, options); err != nil {
		return err
	}
	cfgErr := a.performConfigLockingOperation(cmd.Context(), func() error {
		chain := a.config.Chains[chain]
		oldProv := chain.ChainProvider.(*cosmos.CosmosProvider)
		prov.PCfg.FeeGrants.IsExternalGranter = granter.external
		oldProv.PCfg.FeeGrants = prov.PCfg.FeeGrants
		return nil
	})
	cobra.CheckErr(cfgErr)
	return nil
}

func setFeegrantGrantees(
	prov *cosmos.CosmosProvider,
	granter feegrantGranter,
	options feegrantConfigureOptions,
) error {
	if options.grantees == nil {
		if granter.external {
			return fmt.Errorf("external granter %s was specified, pre-authorized grantees must also be specified", granter.keyOrAddress)
		}
		return prov.ConfigureFeegrants(options.numGrantees, granter.key)
	}
	if !granter.external {
		return prov.ConfigureWithGrantees(options.grantees, granter.key)
	}
	return prov.ConfigureWithExternalGranter(options.grantees, granter.keyOrAddress)
}

func feegrantGas(cmd *cobra.Command) uint64 {
	gasStr, _ := cmd.Flags().GetString(sdkflags.FlagGas)
	if gasStr == "" {
		return 0
	}
	gasSetting, _ := sdkflags.ParseGasSetting(gasStr)
	return gasSetting.Gas
}

func verifyFeegrantConfiguration(
	cmd *cobra.Command,
	ctx context.Context,
	a *appState,
	chain string,
	prov *cosmos.CosmosProvider,
	externalGranter bool,
) error {
	// Get latest height from the chain, mark feegrant configuration as verified up to that height.
	// This means we've verified feegranting is enabled on-chain and TXs can be sent with a feegranter.
	if prov.PCfg.FeeGrants == nil {
		return nil
	}
	h, err := prov.QueryLatestHeight(ctx)
	cobra.CheckErr(err)

	cfgErr := a.performConfigLockingOperation(cmd.Context(), func() error {
		chain := a.config.Chains[chain]
		oldProv := chain.ChainProvider.(*cosmos.CosmosProvider)
		prov.PCfg.FeeGrants.IsExternalGranter = externalGranter
		oldProv.PCfg.FeeGrants = prov.PCfg.FeeGrants
		oldProv.PCfg.FeeGrants.BlockHeightVerified = h
		a.log.Info("feegrant configured", zap.Int64("height", h))
		return nil
	})
	cobra.CheckErr(cfgErr)
	return nil
}

func feegrantBasicGrantsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "basic chain-name [granter]",
		Short: "query the grants for an account (if none is specified, the default account is returned)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeegrantBasicGrants(a, args)
		},
	}
	return paginationFlags(a.viper, cmd, "feegrant")
}

func runFeegrantBasicGrants(a *appState, args []string) error {
	chain := args[0]
	prov, err := feegrantCosmosProvider(a, chain)
	if err != nil {
		return err
	}

	keyNameOrAddress := ""
	if len(args) == 0 {
		keyNameOrAddress = prov.PCfg.Key
	} else {
		keyNameOrAddress = args[0]
	}

	granterAcc, err := prov.AccountFromKeyOrAddress(keyNameOrAddress)
	if err != nil {
		a.log.Error("Unknown account", zap.String("key_or_address", keyNameOrAddress), zap.Error(err))
		return err
	}
	granterAddr := prov.MustEncodeAccAddr(granterAcc)

	res, err := prov.QueryFeegrantsByGranter(granterAddr, nil)
	if err != nil {
		return err
	}

	for _, grant := range res {
		allowance, e := prov.Sprint(grant.Allowance)
		cobra.CheckErr(e)
		a.log.Info("feegrant", zap.String("granter", grant.Granter), zap.String("grantee", grant.Grantee), zap.String("allowance", allowance))
	}
	return nil
}
