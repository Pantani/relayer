package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type clientCreationOptions struct {
	allowUpdateAfterExpiry         bool
	allowUpdateAfterMisbehaviour   bool
	customTrustingPeriod           time.Duration
	customTrustingPeriodPercentage int64
	override                       bool
	maxClockDrift                  time.Duration
}

type singleClientCreationOptions struct {
	clientCreationOptions
	overrideUnbondingPeriod time.Duration
}

// These private command inputs model only the existing IBC Classic workflow.
// IBC v2 must get a separate command contract instead of extending these types.
type commandChainPair struct {
	chains map[string]*relayer.Chain
	src    string
	dst    string
}

type channelCreationOptions struct {
	srcPort string
	dstPort string
	order   string
	version string
}

type handshakeExecutionOptions struct {
	timeout       time.Duration
	retries       uint64
	override      bool
	maxClockDrift time.Duration
}

type connectionCommandInput struct {
	pathName            string
	pair                commandChainPair
	client              clientCreationOptions
	execution           handshakeExecutionOptions
	memo                string
	initialBlockHistory uint64
}

type linkCommandInput struct {
	connectionCommandInput
	channel channelCreationOptions
}

func runCreateClients(a *appState, cmd *cobra.Command, args []string) error {
	options, err := readCreateClientsOptions(cmd)
	if err != nil {
		return err
	}

	pathName := args[0]
	pair, err := chainsForPath(a.config, pathName)
	if err != nil {
		return err
	}
	if err := ensureChainKeys(pair); err != nil {
		return err
	}

	clientSrc, clientDst, err := pair.chains[pair.src].CreateClients(
		cmd.Context(),
		pair.chains[pair.dst],
		options.allowUpdateAfterExpiry,
		options.allowUpdateAfterMisbehaviour,
		options.override,
		options.customTrustingPeriod,
		options.maxClockDrift,
		options.customTrustingPeriodPercentage,
		a.config.memo(cmd),
	)
	if err != nil {
		return err
	}

	return updateCreatedClients(a, cmd.Context(), pathName, clientSrc, clientDst)
}

func readCreateClientsOptions(cmd *cobra.Command) (clientCreationOptions, error) {
	var options clientCreationOptions
	var err error
	options.allowUpdateAfterExpiry, err = cmd.Flags().GetBool(flagUpdateAfterExpiry)
	if err != nil {
		return options, err
	}
	options.allowUpdateAfterMisbehaviour, err = cmd.Flags().GetBool(flagUpdateAfterMisbehaviour)
	if err != nil {
		return options, err
	}
	options.customTrustingPeriod, err = cmd.Flags().GetDuration(flagClientTrustingPeriod)
	if err != nil {
		return options, err
	}
	options.maxClockDrift, err = cmd.Flags().GetDuration(flagMaxClockDrift)
	if err != nil {
		return options, err
	}
	options.customTrustingPeriodPercentage, err = cmd.Flags().GetInt64(flagClientTrustingPeriodPercentage)
	if err != nil {
		return options, err
	}
	options.override, err = cmd.Flags().GetBool(flagOverride)
	return options, err
}

func runCreateClient(a *appState, cmd *cobra.Command, args []string) error {
	options, err := readSingleClientOptions(cmd)
	if err != nil {
		return err
	}

	src, dst, path, err := resolveSingleClientPath(a.config, args)
	if err != nil {
		return err
	}
	if err := ensureSelectedChainKeys(src, dst); err != nil {
		return err
	}

	srcHeader, dstHeader, err := queryClientHeaders(cmd.Context(), a.log, src, dst)
	if err != nil {
		return err
	}
	clientID, err := relayer.CreateClient(
		cmd.Context(), src, dst, srcHeader, dstHeader,
		options.allowUpdateAfterExpiry,
		options.allowUpdateAfterMisbehaviour,
		options.override,
		options.customTrustingPeriod,
		options.overrideUnbondingPeriod,
		options.maxClockDrift,
		options.customTrustingPeriodPercentage,
		a.config.memo(cmd),
	)
	if err != nil {
		return err
	}

	clientSrc, clientDst := clientIDForPath(path, src.ChainID(), clientID)
	return updateCreatedClients(a, cmd.Context(), args[2], clientSrc, clientDst)
}

func readSingleClientOptions(cmd *cobra.Command) (singleClientCreationOptions, error) {
	var options singleClientCreationOptions
	var err error
	options.allowUpdateAfterExpiry, err = cmd.Flags().GetBool(flagUpdateAfterExpiry)
	if err != nil {
		return options, err
	}
	options.allowUpdateAfterMisbehaviour, err = cmd.Flags().GetBool(flagUpdateAfterMisbehaviour)
	if err != nil {
		return options, err
	}
	options.customTrustingPeriod, err = cmd.Flags().GetDuration(flagClientTrustingPeriod)
	if err != nil {
		return options, err
	}
	options.customTrustingPeriodPercentage, err = cmd.Flags().GetInt64(flagClientTrustingPeriodPercentage)
	if err != nil {
		return options, err
	}
	options.overrideUnbondingPeriod, err = cmd.Flags().GetDuration(flagClientUnbondingPeriod)
	if err != nil {
		return options, err
	}
	options.override, err = cmd.Flags().GetBool(flagOverride)
	if err != nil {
		return options, err
	}
	options.maxClockDrift, err = cmd.Flags().GetDuration(flagMaxClockDrift)
	return options, err
}

func resolveSingleClientPath(config *Config, args []string) (*relayer.Chain, *relayer.Chain, *relayer.Path, error) {
	src, ok := config.Chains[args[0]]
	if !ok {
		return nil, nil, nil, errChainNotFound(args[0])
	}
	dst, ok := config.Chains[args[1]]
	if !ok {
		return nil, nil, nil, errChainNotFound(args[1])
	}
	path, err := config.Paths.Get(args[2])
	if err != nil {
		return nil, nil, nil, err
	}

	src.PathEnd = path.End(src.ChainID())
	dst.PathEnd = path.End(dst.ChainID())
	return src, dst, path, nil
}

func queryClientHeaders(ctx context.Context, log *zap.Logger, src, dst *relayer.Chain) (provider.IBCHeader, provider.IBCHeader, error) {
	srcHeight, dstHeight, err := queryLatestHeights(ctx, src, dst)
	if err != nil {
		return nil, nil, err
	}

	var srcHeader, dstHeader provider.IBCHeader
	err = retry.Do(func() error {
		srcHeader, dstHeader, err = relayer.QueryIBCHeaders(ctx, src, dst, srcHeight, dstHeight)
		return err
	}, retry.Context(ctx), relayer.RtyAtt, relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
		log.Info(
			"Failed to get light signed header",
			zap.String("src_chain_id", src.ChainID()),
			zap.Int64("src_height", srcHeight),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.Int64("dst_height", dstHeight),
			zap.Uint("attempt", n+1),
			zap.Uint("max_attempts", relayer.RtyAttNum),
			zap.Error(err),
		)
		srcHeight, dstHeight, _ = relayer.QueryLatestHeights(ctx, src, dst)
	}))
	return srcHeader, dstHeader, err
}

func queryLatestHeights(ctx context.Context, src, dst *relayer.Chain) (int64, int64, error) {
	var srcHeight, dstHeight int64
	var queryErr error
	err := retry.Do(func() error {
		srcHeight, dstHeight, queryErr = relayer.QueryLatestHeights(ctx, src, dst)
		if srcHeight == 0 || dstHeight == 0 || queryErr != nil {
			return fmt.Errorf("failed to query latest heights: %w", queryErr)
		}
		return queryErr
	}, retry.Context(ctx), relayer.RtyAtt, relayer.RtyDel, relayer.RtyErr)
	return srcHeight, dstHeight, err
}

func clientIDForPath(path *relayer.Path, srcChainID, clientID string) (string, string) {
	if path.Src.ChainID == srcChainID {
		return clientID, ""
	}
	return "", clientID
}

func runCreateConnection(a *appState, cmd *cobra.Command, args []string) error {
	input, err := prepareConnectionCommand(a, cmd, args[0])
	if err != nil {
		return err
	}

	clientSrc, clientDst, err := createConnectionClients(cmd.Context(), input)
	if err != nil {
		return err
	}
	if err := updateCreatedClients(a, cmd.Context(), input.pathName, clientSrc, clientDst); err != nil {
		return err
	}

	connectionSrc, connectionDst, err := createOpenConnections(cmd.Context(), input)
	if err != nil {
		return err
	}
	return updateCreatedConnections(a, cmd.Context(), input.pathName, connectionSrc, connectionDst)
}

func prepareConnectionCommand(a *appState, cmd *cobra.Command, pathName string) (connectionCommandInput, error) {
	var input connectionCommandInput
	input.pathName = pathName

	client, err := readHandshakeClientOptions(cmd)
	if err != nil {
		return input, err
	}
	input.client = client
	input.pair, err = chainsForPath(a.config, pathName)
	if err != nil {
		return input, err
	}
	input.execution, err = readHandshakeExecutionOptions(cmd)
	if err != nil {
		return input, err
	}
	if err := ensureChainKeys(input.pair); err != nil {
		return input, err
	}
	input.memo = a.config.memo(cmd)
	input.initialBlockHistory, err = cmd.Flags().GetUint64(flagInitialBlockHistory)
	return input, err
}

func readHandshakeClientOptions(cmd *cobra.Command) (clientCreationOptions, error) {
	var options clientCreationOptions
	var err error
	options.allowUpdateAfterExpiry, err = cmd.Flags().GetBool(flagUpdateAfterExpiry)
	if err != nil {
		return options, err
	}
	options.allowUpdateAfterMisbehaviour, err = cmd.Flags().GetBool(flagUpdateAfterMisbehaviour)
	if err != nil {
		return options, err
	}
	options.customTrustingPeriod, err = cmd.Flags().GetDuration(flagClientTrustingPeriod)
	if err != nil {
		return options, err
	}
	options.customTrustingPeriodPercentage, err = cmd.Flags().GetInt64(flagClientTrustingPeriodPercentage)
	return options, err
}

func readHandshakeExecutionOptions(cmd *cobra.Command) (handshakeExecutionOptions, error) {
	var options handshakeExecutionOptions
	var err error
	options.timeout, err = getTimeout(cmd)
	if err != nil {
		return options, err
	}
	options.retries, err = cmd.Flags().GetUint64(flagMaxRetries)
	if err != nil {
		return options, err
	}
	options.override, err = cmd.Flags().GetBool(flagOverride)
	if err != nil {
		return options, err
	}
	options.maxClockDrift, err = cmd.Flags().GetDuration(flagMaxClockDrift)
	return options, err
}

func createConnectionClients(ctx context.Context, input connectionCommandInput) (string, string, error) {
	return input.pair.chains[input.pair.src].CreateClients(
		ctx,
		input.pair.chains[input.pair.dst],
		input.client.allowUpdateAfterExpiry,
		input.client.allowUpdateAfterMisbehaviour,
		input.execution.override,
		input.client.customTrustingPeriod,
		input.execution.maxClockDrift,
		input.client.customTrustingPeriodPercentage,
		input.memo,
	)
}

func createOpenConnections(ctx context.Context, input connectionCommandInput) (string, string, error) {
	return input.pair.chains[input.pair.src].CreateOpenConnections(
		ctx,
		input.pair.chains[input.pair.dst],
		input.execution.retries,
		input.execution.timeout,
		input.memo,
		input.initialBlockHistory,
		input.pathName,
	)
}

func runLink(a *appState, cmd *cobra.Command, args []string) error {
	input, err := prepareLinkCommand(a, cmd, args[0])
	if err != nil {
		return err
	}

	clientSrc, clientDst, err := createConnectionClients(cmd.Context(), input.connectionCommandInput)
	if err != nil {
		return fmt.Errorf("error creating clients: %w", err)
	}
	if err := updateCreatedClients(a, cmd.Context(), input.pathName, clientSrc, clientDst); err != nil {
		return err
	}

	connectionSrc, connectionDst, err := createOpenConnections(cmd.Context(), input.connectionCommandInput)
	if err != nil {
		return fmt.Errorf("error creating connections: %w", err)
	}
	if err := updateCreatedConnections(a, cmd.Context(), input.pathName, connectionSrc, connectionDst); err != nil {
		return err
	}
	return createOpenChannel(cmd.Context(), input)
}

func prepareLinkCommand(a *appState, cmd *cobra.Command, pathName string) (linkCommandInput, error) {
	var input linkCommandInput
	input.pathName = pathName

	client, err := readHandshakeClientOptions(cmd)
	if err != nil {
		return input, err
	}
	input.client = client
	input.pair, err = linkChainsForPath(a.config, pathName)
	if err != nil {
		return input, err
	}
	input.channel, err = readChannelCreationOptions(cmd)
	if err != nil {
		return input, err
	}
	input.execution, err = readHandshakeExecutionOptions(cmd)
	if err != nil {
		return input, err
	}
	if err := ensureChainKeys(input.pair); err != nil {
		return input, err
	}
	input.memo = a.config.memo(cmd)
	input.initialBlockHistory, err = cmd.Flags().GetUint64(flagInitialBlockHistory)
	return input, err
}

func readChannelCreationOptions(cmd *cobra.Command) (channelCreationOptions, error) {
	var options channelCreationOptions
	var err error
	options.srcPort, err = cmd.Flags().GetString(flagSrcPort)
	if err != nil {
		return options, err
	}
	options.dstPort, err = cmd.Flags().GetString(flagDstPort)
	if err != nil {
		return options, err
	}
	options.order, err = cmd.Flags().GetString(flagOrder)
	if err != nil {
		return options, err
	}
	options.version, err = cmd.Flags().GetString(flagVersion)
	return options, err
}

func createOpenChannel(ctx context.Context, input linkCommandInput) error {
	return input.pair.chains[input.pair.src].CreateOpenChannels(
		ctx,
		input.pair.chains[input.pair.dst],
		input.execution.retries,
		input.execution.timeout,
		input.channel.srcPort,
		input.channel.dstPort,
		input.channel.order,
		input.channel.version,
		input.execution.override,
		input.memo,
		input.pathName,
	)
}

func chainsForPath(config *Config, pathName string) (commandChainPair, error) {
	chains, src, dst, err := config.ChainsFromPath(pathName)
	return commandChainPair{chains: chains, src: src, dst: dst}, err
}

func linkChainsForPath(config *Config, pathName string) (commandChainPair, error) {
	path, err := config.Paths.Get(pathName)
	if err != nil {
		return commandChainPair{}, err
	}
	src, dst := path.Src.ChainID, path.Dst.ChainID
	chains, err := config.Chains.Gets(src, dst)
	if err != nil {
		return commandChainPair{}, err
	}
	chains[src].PathEnd = path.Src
	chains[dst].PathEnd = path.Dst
	return commandChainPair{chains: chains, src: src, dst: dst}, nil
}

func ensureChainKeys(pair commandChainPair) error {
	return ensureSelectedChainKeys(pair.chains[pair.src], pair.chains[pair.dst])
}

func ensureSelectedChainKeys(src, dst *relayer.Chain) error {
	if !src.ChainProvider.KeyExists(src.ChainProvider.Key()) {
		return fmt.Errorf("key %s not found on src chain %s", src.ChainProvider.Key(), src.ChainID())
	}
	if !dst.ChainProvider.KeyExists(dst.ChainProvider.Key()) {
		return fmt.Errorf("key %s not found on dst chain %s", dst.ChainProvider.Key(), dst.ChainID())
	}
	return nil
}

func updateCreatedClients(a *appState, ctx context.Context, pathName, clientSrc, clientDst string) error {
	if clientSrc == "" && clientDst == "" {
		return nil
	}
	return a.updatePathConfig(ctx, pathName, clientSrc, clientDst, "", "")
}

func updateCreatedConnections(a *appState, ctx context.Context, pathName, connectionSrc, connectionDst string) error {
	if connectionSrc == "" && connectionDst == "" {
		return nil
	}
	return a.updatePathConfig(ctx, pathName, "", "", connectionSrc, connectionDst)
}
