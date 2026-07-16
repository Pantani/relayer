package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const flushTimeout = 10 * time.Minute

// transactionCmd returns a parent transaction command handler, where all child
// commands can submit transactions on IBC-connected networks.
func transactionCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transact",
		Aliases: []string{"tx"},
		Short:   "IBC transaction commands",
		Long: strings.TrimSpace(`Commands to create IBC transactions on pre-configured chains.
Most of these commands take a [path] argument. Make sure:
  1. Chains are properly configured to relay over by using the 'rly chains list' command
  2. Path is properly configured to relay over by using the 'rly paths list' command`,
		),
	}

	cmd.AddCommand(
		linkCmd(a),
		linkThenStartCmd(a),
		flushCmd(a),
		relayMsgsCmd(a),
		relayAcksCmd(a),
		xfersend(a),
		lineBreakCommand(),
		createClientsCmd(a),
		createClientCmd(a),
		updateClientsCmd(a),
		upgradeClientsCmd(a),
		createConnectionCmd(a),
		createChannelCmd(a),
		closeChannelCmd(a),
		lineBreakCommand(),
		registerCounterpartyCmd(a),
	)

	return cmd
}

func createClientsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clients path_name",
		Short: "create a clients between two configured chains with a configured path",
		Long: "Creates a working ibc client for chain configured on each end of the" +
			" path by querying headers from each chain and then sending the corresponding create-client messages",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s transact clients demo-path`, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateClients(a, cmd, args)
		},
	}

	cmd = clientParameterFlags(a.viper, cmd)
	cmd = overrideFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

func createClientCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client src_chain_name dst_chain_name path_name",
		Short: "create a client between two configured chains with a configured path",
		Long: "Creates a working ibc client for chain configured on each end of the" +
			" path by querying headers from each chain and then sending the corresponding create-client messages",
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s transact client demo-path`, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateClient(a, cmd, args)
		},
	}

	cmd = clientParameterFlags(a.viper, cmd)
	cmd = clientUnbondingPeriodFlag(a.viper, cmd)
	cmd = overrideFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

func updateClientsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-clients path_name",
		Short: "update IBC clients between two configured chains with a configured path",
		Long: `Updates IBC client for chain configured on each end of the supplied path.
Clients are updated by querying headers from each chain and then sending the
corresponding update-client messages.`,
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s transact update-clients demo-path`, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := a.config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			// ensure that keys exist
			if exists := c[src].ChainProvider.KeyExists(c[src].ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on src chain %s", c[src].ChainProvider.Key(), c[src].ChainID())
			}

			if exists := c[dst].ChainProvider.KeyExists(c[dst].ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on dst chain %s", c[dst].ChainProvider.Key(), c[dst].ChainID())
			}

			return relayer.UpdateClients(cmd.Context(), c[src], c[dst], a.config.memo(cmd))
		},
	}

	return memoFlag(a.viper, cmd)
}

func upgradeClientsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade-clients path_name chain_id",
		Short: "upgrades IBC clients between two configured chains with a configured path and chain-id",
		Args:  withUsage(cobra.ExactArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := a.config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			height, err := cmd.Flags().GetInt64(flagHeight)
			if err != nil {
				return err
			}

			// ensure that keys exist
			if exists := c[src].ChainProvider.KeyExists(c[src].ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on src chain %s", c[src].ChainProvider.Key(), c[src].ChainID())
			}

			if exists := c[dst].ChainProvider.KeyExists(c[dst].ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on dst chain %s", c[dst].ChainProvider.Key(), c[dst].ChainID())
			}

			targetChainID := args[1]

			memo := a.config.memo(cmd)

			// send the upgrade message on the targetChainID
			if src == targetChainID {
				return relayer.UpgradeClient(cmd.Context(), c[dst], c[src], height, memo)
			}

			return relayer.UpgradeClient(cmd.Context(), c[src], c[dst], height, memo)
		},
	}

	cmd = heightFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

func createConnectionCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connection path_name",
		Aliases: []string{"conn"},
		Short:   "create a connection between two configured chains with a configured path; if existing client does not exist, it will create one",
		Long: strings.TrimSpace(`Create or repair a connection between two IBC-connected networks
along a specific path.`,
		),
		Args: withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact connection demo-path
$ %s tx conn demo-path --timeout 5s`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateConnection(a, cmd, args)
		},
	}

	cmd = timeoutFlag(a.viper, cmd)
	cmd = retryFlag(a.viper, cmd)
	cmd = clientParameterFlags(a.viper, cmd)
	cmd = overrideFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	cmd = initBlockFlag(a.viper, cmd)
	return cmd
}

func createChannelCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel path_name",
		Aliases: []string{"chan"},
		Short: "create a channel between two configured chains with a configured path using specified or " +
			"default channel identifiers",
		Long: strings.TrimSpace(`Create or repair a channel between two IBC-connected networks
along a specific path.`,
		),
		Args: withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact channel demo-path --src-port transfer --dst-port transfer --order unordered --version ics20-1
$ %s tx chan demo-path --timeout 5s --max-retries 10`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateChannel(a, cmd, args)
		},
	}

	cmd = timeoutFlag(a.viper, cmd)
	cmd = retryFlag(a.viper, cmd)
	cmd = overrideFlag(a.viper, cmd)
	cmd = channelParameterFlags(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

type createChannelOptions struct {
	override bool
	srcPort  string
	dstPort  string
	order    string
	version  string
	timeout  time.Duration
	retries  uint64
}

func runCreateChannel(a *appState, cmd *cobra.Command, args []string) error {
	pathName := args[0]
	chains, src, dst, err := a.config.ChainsFromPath(pathName)
	if err != nil {
		return err
	}
	options, err := readCreateChannelOptions(cmd)
	if err != nil {
		return err
	}
	if err := ensurePathKeysExist(chains, src, dst); err != nil {
		return err
	}
	return chains[src].CreateOpenChannels(
		cmd.Context(),
		chains[dst],
		options.retries,
		options.timeout,
		options.srcPort,
		options.dstPort,
		options.order,
		options.version,
		options.override,
		a.config.memo(cmd),
		pathName,
	)
}

func readCreateChannelOptions(cmd *cobra.Command) (createChannelOptions, error) {
	var options createChannelOptions
	var err error
	if options.override, err = cmd.Flags().GetBool(flagOverride); err != nil {
		return options, err
	}
	if options.srcPort, err = cmd.Flags().GetString(flagSrcPort); err != nil {
		return options, err
	}
	if options.dstPort, err = cmd.Flags().GetString(flagDstPort); err != nil {
		return options, err
	}
	if options.order, err = cmd.Flags().GetString(flagOrder); err != nil {
		return options, err
	}
	if options.version, err = cmd.Flags().GetString(flagVersion); err != nil {
		return options, err
	}
	if options.timeout, err = getTimeout(cmd); err != nil {
		return options, err
	}
	if options.retries, err = cmd.Flags().GetUint64(flagMaxRetries); err != nil {
		return options, err
	}
	return options, nil
}

func ensurePathKeysExist(chains relayer.Chains, src, dst string) error {
	if exists := chains[src].ChainProvider.KeyExists(chains[src].ChainProvider.Key()); !exists {
		return fmt.Errorf(
			"key %s not found on src chain %s",
			chains[src].ChainProvider.Key(),
			chains[src].ChainID(),
		)
	}
	if exists := chains[dst].ChainProvider.KeyExists(chains[dst].ChainProvider.Key()); !exists {
		return fmt.Errorf(
			"key %s not found on dst chain %s",
			chains[dst].ChainProvider.Key(),
			chains[dst].ChainID(),
		)
	}
	return nil
}

func closeChannelCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel-close path_name src_channel_id src_port_id",
		Short: "close a channel between two configured chains with a configured path",
		Args:  withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact channel-close demo-path channel-0 transfer
$ %s tx channel-close demo-path channel-0 transfer --timeout 5s
$ %s tx channel-close demo-path channel-0 transfer
$ %s tx channel-close demo-path channel-0 transfer -o 3s`,
			appName, appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloseChannel(a, cmd, args)
		},
	}

	cmd = timeoutFlag(a.viper, cmd)
	cmd = retryFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

func runCloseChannel(a *appState, cmd *cobra.Command, args []string) error {
	pathName := args[0]
	chains, src, dst, err := a.config.ChainsFromPath(pathName)
	if err != nil {
		return err
	}
	timeout, err := getTimeout(cmd)
	if err != nil {
		return err
	}
	retries, err := cmd.Flags().GetUint64(flagMaxRetries)
	if err != nil {
		return err
	}
	channelID := args[1]
	portID := args[2]
	if err := ensurePathKeysExist(chains, src, dst); err != nil {
		return err
	}
	srcHeight, err := chains[src].ChainProvider.QueryLatestHeight(cmd.Context())
	if err != nil {
		return err
	}
	if _, err = chains[src].ChainProvider.QueryChannel(cmd.Context(), srcHeight, channelID, portID); err != nil {
		return err
	}
	return chains[src].CloseChannel(
		cmd.Context(),
		chains[dst],
		retries,
		timeout,
		channelID,
		portID,
		a.config.memo(cmd),
		pathName,
	)
}

func linkCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "link path_name",
		Aliases: []string{"connect"},
		Short:   "create clients, connection, and channel between two configured chains with a configured path",
		Long: strings.TrimSpace(`Create an IBC client between two IBC-enabled networks, in addition
to creating a connection and a channel between the two networks on a configured path.`,
		),
		Args: withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact link demo-path --src-port transfer --dst-port transfer
$ %s tx link demo-path
$ %s tx connect demo-path --src-port transfer --dst-port transfer --order unordered --version ics20-1`,
			appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLink(a, cmd, args)
		},
	}
	cmd = timeoutFlag(a.viper, cmd)
	cmd = retryFlag(a.viper, cmd)
	cmd = clientParameterFlags(a.viper, cmd)
	cmd = channelParameterFlags(a.viper, cmd)
	cmd = overrideFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	cmd = initBlockFlag(a.viper, cmd)
	return cmd
}

func linkThenStartCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "link-then-start path_name",
		Aliases: []string{"connect-then-start"},
		Short:   "a shorthand command to execute 'link' followed by 'start'",
		Long: strings.TrimSpace(`Create IBC clients, connection, and channel between two configured IBC
networks with a configured path and then start the relayer on that path.`,
		),
		Args: withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact link-then-start demo-path
$ %s tx link-then-start demo-path --timeout 5s`, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			lCmd := linkCmd(a)

			for err := lCmd.RunE(cmd, args); err != nil; err = lCmd.RunE(cmd, args) {
				a.log.Info("Error running link; retrying", zap.Error(err))
				select {
				case <-time.After(time.Second):
					// Keep going.
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				}
			}

			sCmd := startCmd(a)
			return sCmd.RunE(cmd, args)
		},
	}

	cmd = timeoutFlag(a.viper, cmd)
	cmd = retryFlag(a.viper, cmd)
	cmd = strategyFlag(a.viper, cmd)
	cmd = clientParameterFlags(a.viper, cmd)
	cmd = channelParameterFlags(a.viper, cmd)
	cmd = overrideFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	cmd = debugServerFlags(a.viper, cmd)
	cmd = initBlockFlag(a.viper, cmd)
	cmd = processorFlag(a.viper, cmd)
	cmd = updateTimeFlags(a.viper, cmd)
	cmd = flushIntervalFlag(a.viper, cmd)
	return cmd
}

func flushCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "flush [path_name]? [src_channel_id]?",
		Aliases: []string{"relay-pkts"},
		Short:   "flush any pending MsgRecvPacket and MsgAcknowledgement messages on a given path, in both directions",
		Args:    withUsage(cobra.RangeArgs(0, 2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s tx flush
$ %s tx flush demo-path
$ %s tx flush demo-path channel-0`,
			appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFlush(a, cmd, args)
		},
	}

	cmd = strategyFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	cmd = stuckPacketFlags(a.viper, cmd)

	return cmd
}

func runFlush(a *appState, cmd *cobra.Command, args []string) error {
	paths, chainIDs := flushPathsAndChainIDs(a, args)
	chains, err := a.config.Chains.Gets(chainIDs...)
	if err != nil {
		return err
	}
	if err := ensureKeysExist(chains); err != nil {
		return err
	}
	maxMsgLength, err := cmd.Flags().GetUint64(flagMaxMsgLength)
	if err != nil {
		return err
	}
	if len(args) == 2 {
		paths[0].Path.Filter = relayer.ChannelFilter{
			Rule:        processor.RuleAllowList,
			ChannelList: []string{args[1]},
		}
	}
	stuckPacket, err := parseStuckPacketFromFlags(cmd)
	if err != nil {
		return err
	}
	return startFlushRelayer(a, cmd, chains, paths, maxMsgLength, stuckPacket)
}

func flushPathsAndChainIDs(a *appState, args []string) ([]relayer.NamedPath, []string) {
	seenChainIDs := make(map[string]struct{})
	var paths []relayer.NamedPath
	var chainIDs []string
	if len(args) > 0 {
		pathName := args[0]
		path := a.config.Paths.MustGet(pathName)
		paths = append(paths, relayer.NamedPath{Name: pathName, Path: path})
		chainIDs = appendPathChainIDs(chainIDs, seenChainIDs, path)
	} else {
		for name, path := range a.config.Paths {
			paths = append(paths, relayer.NamedPath{Name: name, Path: path})
			chainIDs = appendPathChainIDs(chainIDs, seenChainIDs, path)
		}
	}
	return paths, chainIDs
}

func appendPathChainIDs(chainIDs []string, seen map[string]struct{}, path *relayer.Path) []string {
	if _, ok := seen[path.Src.ChainID]; !ok {
		seen[path.Src.ChainID] = struct{}{}
		chainIDs = append(chainIDs, path.Src.ChainID)
	}
	if _, ok := seen[path.Dst.ChainID]; !ok {
		seen[path.Dst.ChainID] = struct{}{}
		chainIDs = append(chainIDs, path.Dst.ChainID)
	}
	return chainIDs
}

func startFlushRelayer(
	a *appState,
	cmd *cobra.Command,
	chains relayer.Chains,
	paths []relayer.NamedPath,
	maxMsgLength uint64,
	stuckPacket *processor.StuckPacket,
) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), flushTimeout)
	defer cancel()
	rlyErrCh := relayer.StartRelayer(
		ctx,
		a.log,
		chains,
		paths,
		maxMsgLength,
		a.config.Global.MaxReceiverSize,
		a.config.Global.ICS20MemoLimit,
		a.config.memo(cmd),
		0,
		0,
		&processor.FlushLifecycle{},
		relayer.ProcessorEvents,
		0,
		nil,
		stuckPacket,
	)

	// Block until the error channel sends a message. The canceled context will
	// stop the relayer, so returning on ctx.Done could precede its cleanup.
	if err := <-rlyErrCh; err != nil && !errors.Is(err, context.Canceled) {
		a.log.Warn("Relayer start error", zap.Error(err))
		return err
	}
	return nil
}

func relayMsgsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "relay-packets path_name src_channel_id",
		Aliases: []string{"relay-pkts"},
		Short:   "relay any remaining non-relayed packets on a given path, in both directions",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact relay-packets demo-path channel-0
$ %s tx relay-pkts demo-path channel-0`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.log.Warn("This command is deprecated. Please use 'tx flush' command instead")
			return flushCmd(a).RunE(cmd, args)
		},
	}

	cmd = strategyFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

func relayAcksCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "relay-acknowledgements path_name src_channel_id",
		Aliases: []string{"relay-acks"},
		Short:   "relay any remaining non-relayed acknowledgements on a given path, in both directions",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s transact relay-acknowledgements demo-path channel-0
$ %s tx relay-acks demo-path channel-0 -l 3 -s 6`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.log.Warn("This command is deprecated. Please use 'tx flush' command instead")
			return flushCmd(a).RunE(cmd, args)
		},
	}

	cmd = strategyFlag(a.viper, cmd)
	cmd = memoFlag(a.viper, cmd)
	return cmd
}

func xfersend(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer src_chain_name dst_chain_name amount dst_addr src_channel_id",
		Short: "initiate a transfer from one network to another",
		Long: `Initiate a token transfer via IBC between two networks. The created packet
must be relayed to the destination chain.`,
		Args: withUsage(cobra.ExactArgs(5)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s tx transfer ibc-0 ibc-1 100000stake cosmos1skjwj5whet0lpe65qaq4rpq03hjxlwd9nf39lk channel-0 --path demo-path
$ %s tx transfer ibc-0 ibc-1 100000stake cosmos1skjwj5whet0lpe65qaq4rpq03hjxlwd9nf39lk channel-0 --path demo -y 2 -c 10
$ %s tx transfer ibc-0 ibc-1 100000stake raw:non-bech32-address channel-0 --path demo
$ %s tx raw send ibc-0 ibc-1 100000stake cosmos1skjwj5whet0lpe65qaq4rpq03hjxlwd9nf39lk channel-0 --path demo -c 5
`, appName, appName, appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTransfer(a, cmd, args)
		},
	}

	cmd = memoFlag(a.viper, cmd)
	return timeoutFlags(a.viper, pathFlag(a.viper, cmd))
}

type transferPreparation struct {
	src       *relayer.Chain
	dst       *relayer.Chain
	path      *relayer.Path
	amount    sdk.Coin
	srcHeight int64
}

func runTransfer(a *appState, cmd *cobra.Command, args []string) error {
	transfer, err := prepareTransfer(a, cmd, args)
	if err != nil {
		return err
	}
	srcChannel, amount, err := prepareTransferChannel(cmd, transfer, args[4])
	if err != nil {
		return err
	}
	timeoutHeightOffset, timeoutTimeOffset, err := readTransferTimeouts(cmd)
	if err != nil {
		return err
	}

	// If the argument begins with "raw:" then use the suffix directly.
	rawDstAddr := strings.TrimPrefix(args[3], "raw:")
	dstAddr := args[3]
	if rawDstAddr != args[3] {
		// Don't parse the rest of the dstAddr; it is raw.
		dstAddr = rawDstAddr
	}
	return transfer.src.SendTransferMsg(
		cmd.Context(),
		a.log,
		transfer.dst,
		amount,
		dstAddr,
		a.config.memo(cmd),
		timeoutHeightOffset,
		timeoutTimeOffset,
		srcChannel,
	)
}

func prepareTransfer(a *appState, cmd *cobra.Command, args []string) (transferPreparation, error) {
	var transfer transferPreparation
	var ok bool
	transfer.src, ok = a.config.Chains[args[0]]
	if !ok {
		return transfer, errChainNotFound(args[0])
	}
	transfer.dst, ok = a.config.Chains[args[1]]
	if !ok {
		return transfer, errChainNotFound(args[1])
	}
	pathName, err := cmd.Flags().GetString(flagPath)
	if err != nil {
		return transfer, err
	}
	transfer.path, err = setPathsFromArgs(a, transfer.src, transfer.dst, pathName)
	if err != nil {
		return transfer, err
	}
	transfer.amount, err = sdk.ParseCoinNormalized(args[2])
	if err != nil {
		return transfer, err
	}
	transfer.srcHeight, err = transfer.src.ChainProvider.QueryLatestHeight(cmd.Context())
	return transfer, err
}

func prepareTransferChannel(
	cmd *cobra.Command,
	transfer transferPreparation,
	srcChannelID string,
) (*chantypes.IdentifiedChannel, sdk.Coin, error) {
	connectionID, err := transferPathConnectionID(transfer.src, transfer.path)
	if err != nil {
		return nil, sdk.Coin{}, err
	}
	channels, err := transfer.src.ChainProvider.QueryConnectionChannels(
		cmd.Context(),
		transfer.srcHeight,
		connectionID,
	)
	if err != nil {
		return nil, sdk.Coin{}, err
	}
	srcChannel := transferChannel(channels, srcChannelID)
	if srcChannel == nil {
		return nil, sdk.Coin{}, fmt.Errorf(
			"could not find channel{%s} for chain{%s}@connection{%s}",
			srcChannelID,
			transfer.src,
			connectionID,
		)
	}
	denomTraces, err := transfer.src.ChainProvider.QueryDenomTraces(
		cmd.Context(),
		0,
		100,
		transfer.srcHeight,
	)
	if err != nil {
		return nil, sdk.Coin{}, err
	}
	return srcChannel, transferIBCDenom(transfer.amount, denomTraces), nil
}

func transferPathConnectionID(src *relayer.Chain, path *relayer.Path) (string, error) {
	switch src.ChainID() {
	case path.Src.ChainID:
		return path.Src.ConnectionID, nil
	case path.Dst.ChainID:
		return path.Dst.ConnectionID, nil
	default:
		return "", fmt.Errorf("no path configured using chain-id: %s", src.ChainID())
	}
}

func transferChannel(
	channels []*chantypes.IdentifiedChannel,
	srcChannelID string,
) *chantypes.IdentifiedChannel {
	for _, channel := range channels {
		if channel.ChannelId == srcChannelID {
			return channel
		}
	}
	return nil
}

func transferIBCDenom(amount sdk.Coin, denomTraces []transfertypes.Denom) sdk.Coin {
	for _, denomTrace := range denomTraces {
		if amount.Denom == denomTrace.Path() {
			amount = sdk.NewCoin(denomTrace.IBCDenom(), amount.Amount)
		}
	}
	return amount
}

func readTransferTimeouts(cmd *cobra.Command) (uint64, time.Duration, error) {
	heightOffset, err := cmd.Flags().GetUint64(flagTimeoutHeightOffset)
	if err != nil {
		return 0, 0, err
	}
	timeOffset, err := cmd.Flags().GetDuration(flagTimeoutTimeOffset)
	if err != nil {
		return 0, 0, err
	}
	return heightOffset, timeOffset, nil
}

func setPathsFromArgs(a *appState, src, dst *relayer.Chain, name string) (*relayer.Path, error) {
	// find any configured paths between the chains
	paths, err := a.config.Paths.PathsFromChains(src.ChainID(), dst.ChainID())
	if err != nil {
		return nil, err
	}
	path, err := selectPathFromArgs(paths, src, dst, name)
	if err != nil {
		return nil, err
	}
	if err := src.SetPath(path.End(src.ChainID())); err != nil {
		return nil, err
	}
	if err := dst.SetPath(path.End(dst.ChainID())); err != nil {
		return nil, err
	}
	return path, nil
}

func selectPathFromArgs(paths relayer.Paths, src, dst *relayer.Chain, name string) (*relayer.Path, error) {
	if name != "" {
		return paths.Get(name)
	}
	if len(paths) > 1 {
		return nil, fmt.Errorf(
			"more than one path between %s and %s exists, pass in path name",
			src.ChainID(),
			dst.ChainID(),
		)
	}
	for _, path := range paths {
		return path, nil
	}
	return nil, nil
}

// ensureKeysExist returns an error if a configured key for a given chain does not exist.
func ensureKeysExist(chains map[string]*relayer.Chain) error {
	for _, v := range chains {
		if exists := v.ChainProvider.KeyExists(v.ChainProvider.Key()); !exists {
			return fmt.Errorf("key %s not found on chain %s", v.ChainProvider.Key(), v.ChainID())
		}
	}

	return nil
}

// registerCounterpartyCmd registers the counterparty_payee
func registerCounterpartyCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "register-counterparty chain_name channel_id port_id relay_addr counterparty_payee",
		Aliases: []string{"reg-cpt"},
		Short:   "register the counterparty relayer address for ics-29 fee middleware",
		Args:    withUsage(cobra.MatchAll(cobra.ExactArgs(5))),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s register-counterparty channel-1 transfer cosmos1skjwj5whet0lpe65qaq4rpq03hjxlwd9nf39lk 
juno1g0ny488ws4064mjjxk4keenwfjrthn503ngjxd
$ %s reg-cpt channel-1 cosmos1skjwj5whet0lpe65qaq4rpq03hjxlwd9nf39lk juno1g0ny488ws4064mjjxk4keenwfjrthn503ngjxd`,
			appName, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			channelID := args[1]
			portID := args[2]

			relayerAddr := args[3]
			counterpartyPayee := args[4]

			msg, err := chain.ChainProvider.MsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayee)
			if err != nil {
				return err
			}

			memo := a.config.memo(cmd)

			res, success, err := chain.ChainProvider.SendMessage(cmd.Context(), msg, memo)
			fmt.Println(res, success, err)

			return nil
		},
	}

	return memoFlag(a.viper, cmd)
}
