package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/gogoproto/proto"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/spf13/cobra"
)

const (
	formatJson   = "json"
	formatLegacy = "legacy"
)

// queryCmd represents the chain command
func queryCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "IBC query commands",
		Long:    "Commands to query IBC primitives and other useful data on configured chains.",
	}

	cmd.AddCommand(
		queryUnrelayedPackets(a),
		queryUnrelayedAcknowledgements(a),
		lineBreakCommand(),
		queryBalanceCmd(a),
		queryBalancesCmd(a),
		queryHeaderCmd(a),
		queryNodeStateCmd(a),
		queryTxs(a),
		queryTx(a),
		lineBreakCommand(),
		queryClientCmd(a),
		queryClientsCmd(a),
		queryClientsExpiration(a),
		queryConnection(a),
		queryConnections(a),
		queryConnectionsUsingClient(a),
		queryChannel(a),
		queryChannels(a),
		queryConnectionChannels(a),
		queryPacketCommitment(a),
		lineBreakCommand(),
		queryIBCDenoms(a),
		queryBaseDenomFromIBCDenom(a),
		feegrantQueryCmd(a),
		queryIBCDenomHash(a),
	)

	return cmd
}

// feegrantQueryCmd returns the fee grant query commands for this module
func feegrantQueryCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feegrant",
		Short: "Querying commands for the feegrant module [currently BasicAllowance only]",
	}

	cmd.AddCommand(
		feegrantBasicGrantsCmd(a),
	)
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryIBCDenoms(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-denoms chain_name",
		Short: "query denomination traces for a given network by chain ID",
		Args:  withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query ibc-denoms ibc-0
$ %s q ibc-denoms ibc-0`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			h, err := chain.ChainProvider.QueryLatestHeight(cmd.Context())
			if err != nil {
				return err
			}

			res, err := chain.ChainProvider.QueryDenomTraces(cmd.Context(), 0, 100, h)
			if err != nil {
				return err
			}

			for _, d := range res {
				fmt.Fprintln(cmd.OutOrStdout(), d)
			}
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryBaseDenomFromIBCDenom(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-trace chain_id denom_hash",
		Short: "query that retrieves the base denom from the IBC denomination trace",
		Args:  withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query denom-trace osmosis 9BBA9A1C257E971E38C1422780CE6F0B0686F0A3085E2D61118D904BFE0F5F5E
$ %s q denom-trace osmosis 9BBA9A1C257E971E38C1422780CE6F0B0686F0A3085E2D61118D904BFE0F5F5E`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			res, err := c.ChainProvider.QueryDenomTrace(cmd.Context(), args[1])
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryIBCDenomHash(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-hash chain_id trace",
		Short: "query the denom hash info from a given denom trace",
		Args:  withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query denom-hash osmosis transfer/channel-0/uatom
$ %s q denom-hash osmosis transfer/channel-0/uatom`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			res, err := c.ChainProvider.QueryDenomHash(cmd.Context(), args[1])
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryTx(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx chain_name tx_hash",
		Short: "query for a transaction on a given network by transaction hash and chain ID",
		Args:  withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query tx ibc-0 [tx-hash]
$ %s q tx ibc-0 A5DF8D272F1C451CFF92BA6C41942C4D29B5CF180279439ED6AB038282F956BE`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			txs, err := chain.ChainProvider.QueryTx(cmd.Context(), args[1])
			if err != nil {
				return err
			}

			out, err := json.Marshal(txs)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryTxs(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "txs chain_name events",
		Short: "query for transactions on a given network by chain ID and a set of transaction events",
		Long: strings.TrimSpace(`Search for a paginated list of transactions that match the given set of
events. Each event takes the form of '{eventType}.{eventAttribute}={value}' with multiple events
separated by '&'.

Please refer to each module's documentation for the full set of events to query for. Each module
documents its respective events under 'cosmos-sdk/x/{module}/spec/xx_events.md'.`,
		),
		Args: withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query txs ibc-0 "message.action=transfer" --page 1 --limit 10
$ %s q txs ibc-0 "message.action=transfer"`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			page, err := cmd.Flags().GetUint64(flagPage)
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetUint64(flagLimit)
			if err != nil {
				return err
			}

			txs, err := chain.ChainProvider.QueryTxs(cmd.Context(), int(page), int(limit), []string{args[1]})
			if err != nil {
				return err
			}

			out, err := json.Marshal(txs)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = paginationFlags(a.viper, cmd, "txs")
	return cmd
}

func queryBalanceCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "balance chain_name [key_name]",
		Aliases: []string{"bal"},
		Short:   "query the relayer's account balance on a given network by chain-ID",
		Args:    withUsage(cobra.RangeArgs(1, 2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query balance ibc-0
$ %s query balance ibc-0 testkey`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryBalance(a, cmd, args)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = ibcDenomFlags(a.viper, cmd)
	return cmd
}

func runQueryBalance(a *appState, cmd *cobra.Command, args []string) error {
	chain, ok := a.config.Chains[args[0]]
	if !ok {
		return errChainNotFound(args[0])
	}

	showDenoms, err := cmd.Flags().GetBool(flagIBCDenoms)
	if err != nil {
		return err
	}

	keyName := chain.ChainProvider.Key()
	if len(args) == 2 {
		keyName = args[1]
	}

	if !chain.ChainProvider.KeyExists(keyName) {
		return errKeyDoesntExist(keyName)
	}

	addr, err := chain.ChainProvider.ShowAddress(keyName)
	if err != nil {
		return err
	}

	coins, err := relayer.QueryBalance(cmd.Context(), chain, addr, showDenoms)
	if err != nil {
		return err
	}

	jsonOutput, err := json.Marshal(map[string]string{
		"address": addr,
		"balance": coins.String(),
	})
	if err != nil {
		return err
	}

	printQueryBalance(cmd, addr, coins.String(), jsonOutput)
	return nil
}

func printQueryBalance(cmd *cobra.Command, addr, balance string, jsonOutput []byte) {
	output, _ := cmd.Flags().GetString(flagOutput)
	switch output {
	case formatJson:
		fmt.Fprint(cmd.OutOrStdout(), string(jsonOutput))
	case formatLegacy:
		fallthrough
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "address {%s} balance {%s} \n", addr, balance)
	}
}

func queryBalancesCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balances [chain-name...]",
		Short: "query the relayer's account balances on given networks by chain-ID",
		Args:  withUsage(cobra.MinimumNArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query balances ibc-0 ibc-1
$ %s query balances ibc-0 ibc-1 --key-name=test`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryBalances(a, cmd, args)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = ibcDenomFlags(a.viper, cmd)
	cmd = keyNameFlag(a.viper, cmd)
	return cmd
}

func runQueryBalances(a *appState, cmd *cobra.Command, args []string) error {
	keyName, _ := cmd.Flags().GetString(flagKeyName)
	data := map[string]string{}
	for _, arg := range args {
		addr, balance, err := queryBalanceEntry(a, cmd, arg, args[0], keyName)
		if err != nil {
			return err
		}
		data[addr] = balance
	}

	jsonOutput, err := json.Marshal(data)
	if err != nil {
		return err
	}
	printQueryBalances(cmd, data, jsonOutput)
	return nil
}

func queryBalanceEntry(
	a *appState,
	cmd *cobra.Command,
	chainName,
	missingChainName,
	keyName string,
) (string, string, error) {
	chain, ok := a.config.Chains[chainName]
	if !ok {
		return "", "", errChainNotFound(missingChainName)
	}

	chainKey := keyName
	if chainKey == "" {
		chainKey = chain.ChainProvider.Key()
	}

	showDenoms, err := cmd.Flags().GetBool(flagIBCDenoms)
	if err != nil {
		return "", "", err
	}
	if !chain.ChainProvider.KeyExists(chainKey) {
		return "", "", errKeyDoesntExist(chainKey)
	}

	addr, err := chain.ChainProvider.ShowAddress(chainKey)
	if err != nil {
		return "", "", err
	}
	coins, err := relayer.QueryBalance(cmd.Context(), chain, addr, showDenoms)
	if err != nil {
		return "", "", err
	}
	return addr, coins.String(), nil
}

func printQueryBalances(cmd *cobra.Command, data map[string]string, jsonOutput []byte) {
	output, _ := cmd.Flags().GetString(flagOutput)
	switch output {
	case formatJson:
		fmt.Fprint(cmd.OutOrStdout(), string(jsonOutput))
	case formatLegacy:
		fallthrough
	default:
		for addr, balance := range data {
			fmt.Fprintf(cmd.OutOrStdout(), "address {%s} balance {%s} \n", addr, balance)
		}
	}
}

func queryHeaderCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "header chain_name [height]",
		Short: "query the header of a network by chain ID at a given height or the latest height",
		Args:  withUsage(cobra.RangeArgs(1, 2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query header ibc-0
$ %s query header ibc-0 1400`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryHeader(a, cmd, args)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func runQueryHeader(a *appState, cmd *cobra.Command, args []string) error {
	chain, ok := a.config.Chains[args[0]]
	if !ok {
		return errChainNotFound(args[0])
	}
	height, err := queryHeaderHeight(cmd, chain, args)
	if err != nil {
		return err
	}
	header, err := chain.ChainProvider.QueryIBCHeader(cmd.Context(), height)
	if err != nil {
		return err
	}
	s, err := json.Marshal(header)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal header: %v\n", err)
		return err
	}
	printQueryHeader(cmd, s)
	return nil
}

func queryHeaderHeight(cmd *cobra.Command, chain *relayer.Chain, args []string) (int64, error) {
	switch len(args) {
	case 1:
		return chain.ChainProvider.QueryLatestHeight(cmd.Context())
	case 2:
		return strconv.ParseInt(args[1], 10, 64)
	default:
		return 0, nil
	}
}

func printQueryHeader(cmd *cobra.Command, header []byte) {
	output, _ := cmd.Flags().GetString(flagOutput)
	switch output {
	case formatJson:
		fmt.Fprintln(cmd.OutOrStdout(), string(header))
	case formatLegacy:
		fallthrough
	default:
		fmt.Fprintln(cmd.OutOrStdout(), header)
	}
}

// GetCmdQueryConsensusState defines the command to query the consensus state of
// the chain as defined in https://github.com/cosmos/ics/tree/master/spec/ics-002-client-semantics#query
func queryNodeStateCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node-state chain_name",
		Short: "query the consensus state of a network by chain ID",
		Args:  withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query node-state ibc-0
$ %s q node-state ibc-1`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			height, err := chain.ChainProvider.QueryLatestHeight(cmd.Context())
			if err != nil {
				return err
			}

			csRes, _, err := chain.ChainProvider.QueryConsensusState(cmd.Context(), height)
			if err != nil {
				return err
			}

			s, err := chain.ChainProvider.Sprint(csRes)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal consensus state: %v\n", err)
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryClientCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client chain_name client_id",
		Short: "query the state of a light client on a network by chain ID",
		Args:  withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query client osmosis 07-tendermint-259
$ %s query client ibc-0 ibczeroclient --height 1205`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryClient(a, cmd, args)
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	cmd = heightFlag(a.viper, cmd)
	return cmd
}

func runQueryClient(a *appState, cmd *cobra.Command, args []string) error {
	chain, ok := a.config.Chains[args[0]]
	if !ok {
		return errChainNotFound(args[0])
	}
	height, err := queryHeightFromFlag(cmd, chain)
	if err != nil {
		return err
	}
	if err = chain.AddPath(args[1], dcon); err != nil {
		return err
	}
	res, err := chain.ChainProvider.QueryClientStateResponse(cmd.Context(), height, chain.ClientID())
	if err != nil {
		return err
	}
	return printQueryProto(cmd, chain, res, "Failed to marshal state")
}

func queryHeightFromFlag(cmd *cobra.Command, chain *relayer.Chain) (int64, error) {
	height, err := cmd.Flags().GetInt64(flagHeight)
	if err != nil {
		return 0, err
	}
	if height == 0 {
		return chain.ChainProvider.QueryLatestHeight(cmd.Context())
	}
	return height, nil
}

func printQueryProto(
	cmd *cobra.Command,
	chain *relayer.Chain,
	res proto.Message,
	errorPrefix string,
) error {
	s, err := chain.ChainProvider.Sprint(res)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", errorPrefix, err)
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), s)
	return nil
}

func queryClientsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clients chain_name",
		Aliases: []string{"clnts"},
		Short:   "query for all light client states on a network by chain ID",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query clients osmosis
$ %s query clients ibc-2 --offset 2 --limit 30`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			// TODO fix pagination
			//pagereq, err := client.ReadPageRequest(cmd.Flags())
			//if err != nil {
			//	return err
			//}

			res, err := chain.ChainProvider.QueryClients(cmd.Context())
			if err != nil {
				return err
			}

			for _, client := range res {
				s, err := chain.ChainProvider.Sprint(&client)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal state: %v\n", err)
					continue
				}

				fmt.Fprintln(cmd.OutOrStdout(), s)
			}

			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	cmd = paginationFlags(a.viper, cmd, "client states")
	return cmd
}

func queryConnections(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connections chain_id",
		Aliases: []string{"conns"},
		Short:   "query for all connections on a network by chain ID",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query connections ibc-0
$ %s query connections ibc-2 --offset 2 --limit 30
$ %s q conns ibc-1`,
			appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			// TODO fix pagination
			//pagereq, err := client.ReadPageRequest(cmd.Flags())
			//if err != nil {
			//	return err
			//}

			res, err := chain.ChainProvider.QueryConnections(cmd.Context())
			if err != nil {
				return err
			}

			for _, connection := range res {
				s, err := chain.ChainProvider.Sprint(connection)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal connection: %v\n", err)
					continue
				}

				fmt.Fprintln(cmd.OutOrStdout(), s)
			}

			return nil
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = paginationFlags(a.viper, cmd, "connections on a network")
	return cmd
}

func queryConnectionsUsingClient(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client-connections chain_name client_id",
		Short: "query for all connections for a given client on a network by chain ID",
		Args:  withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query client-connections ibc-0 ibczeroclient
$ %s query client-connections ibc-0 ibczeroclient --height 1205`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryConnectionsUsingClient(a, cmd, args)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = heightFlag(a.viper, cmd)
	return cmd
}

func runQueryConnectionsUsingClient(a *appState, cmd *cobra.Command, args []string) error {
	// TODO - Add pagination.
	chain, ok := a.config.Chains[args[0]]
	if !ok {
		return errChainNotFound(args[0])
	}
	if err := chain.AddPath(args[1], dcon); err != nil {
		return err
	}
	height, err := queryHeightFromFlag(cmd, chain)
	if err != nil {
		return err
	}
	res, err := chain.ChainProvider.QueryConnectionsUsingClient(cmd.Context(), height, chain.ClientID())
	if err != nil {
		return err
	}
	return printQueryProto(cmd, chain, res, "Failed to marshal client connection state")
}

func queryConnection(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connection chain_name connection_id",
		Aliases: []string{"conn"},
		Short:   "query the connection state for a given connection id on a network by chain ID",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query connection ibc-0 ibconnection0
$ %s q conn ibc-1 ibconeconn`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			if err := chain.AddPath(dcli, args[1]); err != nil {
				return err
			}

			height, err := chain.ChainProvider.QueryLatestHeight(cmd.Context())
			if err != nil {
				return err
			}

			res, err := chain.ChainProvider.QueryConnection(cmd.Context(), height, chain.ConnectionID())
			if err != nil {
				return err
			}

			s, err := chain.ChainProvider.Sprint(res)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal connection state: %v\n", err)
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryConnectionChannels(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection-channels chain_name connection_id",
		Short: "query all channels associated with a given connection on a network by chain ID",
		Args:  withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query connection-channels ibc-0 ibcconnection1
$ %s query connection-channels ibc-2 ibcconnection2 --offset 2 --limit 30`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryConnectionChannels(a, cmd, args)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = paginationFlags(a.viper, cmd, "channels associated with a connection")
	return cmd
}

func runQueryConnectionChannels(a *appState, cmd *cobra.Command, args []string) error {
	chain, ok := a.config.Chains[args[0]]
	if !ok {
		return errChainNotFound(args[0])
	}
	if err := chain.AddPath(dcli, args[1]); err != nil {
		return err
	}

	// TODO fix pagination.
	chans, err := chain.ChainProvider.QueryConnectionChannels(cmd.Context(), 0, args[1])
	if err != nil {
		return err
	}
	for _, channel := range chans {
		if err := printQueryProto(cmd, chain, channel, "Failed to marshal channel"); err != nil {
			continue
		}
	}
	return nil
}

func queryChannel(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel chain_name channel_id port_id",
		Short: "query a channel by channel and port ID on a network by chain ID",
		Args:  withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query channel ibc-0 ibczerochannel transfer
$ %s query channel ibc-2 ibctwochannel transfer --height 1205`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryChannel(a, cmd, args)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = heightFlag(a.viper, cmd)
	return cmd
}

func runQueryChannel(a *appState, cmd *cobra.Command, args []string) error {
	chain, ok := a.config.Chains[args[0]]
	if !ok {
		return errChainNotFound(args[0])
	}
	channelID := args[1]
	portID := args[2]
	if err := chain.AddPath(dcli, dcon); err != nil {
		return err
	}
	height, err := queryHeightFromFlag(cmd, chain)
	if err != nil {
		return err
	}
	res, err := chain.ChainProvider.QueryChannel(cmd.Context(), height, channelID, portID)
	if err != nil {
		return err
	}
	return printQueryProto(cmd, chain, res, "Failed to marshal channel state")
}

// chanExtendedInfo is an intermediate type for holding additional useful
// channel information regarding IBC hierarchy of clients/conns/chans.
type chanExtendedInfo struct {
	clientID             string
	counterpartyChainID  string
	counterpartyConnID   string
	counterpartyClientID string
}

func printChannelWithExtendedInfo(
	cmd *cobra.Command,
	chain *relayer.Chain,
	channel *chantypes.IdentifiedChannel,
	extendedInfo *chanExtendedInfo) {
	s, err := chain.ChainProvider.Sprint(channel)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal channel: %v\n", err)
		return
	}

	if extendedInfo == nil || len(channel.ConnectionHops) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), s)
		return
	}

	asJson := make(map[string]any)
	if err := json.Unmarshal([]byte(s), &asJson); err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), s)
		return
	}

	asJson["chain_id"] = chain.ChainProvider.ChainId()
	asJson["client_id"] = extendedInfo.clientID
	counterparty, ok := asJson["counterparty"].(map[string]any)
	if ok {
		counterparty["chain_id"] = extendedInfo.counterpartyChainID
		counterparty["client_id"] = extendedInfo.counterpartyClientID
		counterparty["connection_id"] = extendedInfo.counterpartyConnID
	}

	newJson, err := json.Marshal(asJson)
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), s)
		return
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(newJson))
}

const concurrentQueries = 10

func queryChannelsToChain(cmd *cobra.Command, chain *relayer.Chain, dstChain *relayer.Chain) error {
	ctx := cmd.Context()
	clients, err := chain.ChainProvider.QueryClients(ctx)
	if err != nil {
		return err
	}
	for _, client := range clients {
		queryClientChannelsToChain(cmd, chain, dstChain, client)
	}
	return nil
}

func queryClientChannelsToChain(
	cmd *cobra.Command,
	chain,
	dstChain *relayer.Chain,
	clientState clienttypes.IdentifiedClientState,
) {
	clientInfo, err := relayer.ClientInfoFromClientState(clientState.ClientState)
	if err != nil {
		return
	}
	if clientInfo.ChainID != dstChain.ChainProvider.ChainId() {
		return
	}
	connections, err := chain.ChainProvider.QueryConnectionsUsingClient(cmd.Context(), 0, clientState.ClientId)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for i, conn := range connections.Connections {
		wg.Add(1)
		info := chanExtendedInfo{
			clientID:             clientState.ClientId,
			counterpartyChainID:  clientInfo.ChainID,
			counterpartyClientID: conn.Counterparty.ClientId,
			counterpartyConnID:   conn.Counterparty.ConnectionId,
		}
		go queryConnectionChannelsToChain(cmd, chain, conn.Id, info, &wg)
		if (i+1)%concurrentQueries == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
}

func queryConnectionChannelsToChain(
	cmd *cobra.Command,
	chain *relayer.Chain,
	connectionID string,
	info chanExtendedInfo,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	channels, err := chain.ChainProvider.QueryConnectionChannels(cmd.Context(), 0, connectionID)
	if err != nil {
		return
	}
	for _, channel := range channels {
		printChannelWithExtendedInfo(cmd, chain, channel, &info)
	}
}

type queryChannelsPage struct {
	channels      []*chantypes.IdentifiedChannel
	next          []byte
	isCosmosChain bool
}

func queryChannelsPaginated(cmd *cobra.Command, chain *relayer.Chain, pageReq *query.PageRequest) error {
	page, err := loadQueryChannelsPage(cmd, chain, pageReq)
	if err != nil {
		return err
	}
	uniqueConnections := uniqueChannelConnections(page.channels)
	connectionClients := queryConnectionClients(cmd, chain, uniqueConnections)
	printChannelsWithConnectionClients(cmd, chain, page.channels, connectionClients)
	if page.isCosmosChain {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nPagination next key: %s\n", string(page.next))
	}
	return nil
}

func loadQueryChannelsPage(
	cmd *cobra.Command,
	chain *relayer.Chain,
	pageReq *query.PageRequest,
) (queryChannelsPage, error) {
	ccp, isCosmosChain := chain.ChainProvider.(*cosmos.CosmosProvider)
	if isCosmosChain {
		channels, next, err := ccp.QueryChannelsPaginated(cmd.Context(), pageReq)
		return queryChannelsPage{channels, next, true}, err
	}
	channels, err := chain.ChainProvider.QueryChannels(cmd.Context())
	return queryChannelsPage{channels: channels}, err
}

func uniqueChannelConnections(channels []*chantypes.IdentifiedChannel) map[string]interface{} {
	uniqueConnections := make(map[string]interface{})
	for _, channel := range channels {
		if len(channel.ConnectionHops) == 0 {
			continue
		}
		uniqueConnections[channel.ConnectionHops[0]] = struct{}{}
	}
	return uniqueConnections
}

func queryConnectionClients(
	cmd *cobra.Command,
	chain *relayer.Chain,
	connectionIDs map[string]interface{},
) map[string]chanExtendedInfo {
	connectionClients := make(map[string]chanExtendedInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup
	index := 0
	for connectionID := range connectionIDs {
		wg.Add(1)
		go queryConnectionClient(cmd, chain, connectionID, connectionClients, &mu, &wg)
		index++
		if index%concurrentQueries == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
	return connectionClients
}

func queryConnectionClient(
	cmd *cobra.Command,
	chain *relayer.Chain,
	connectionID string,
	connectionClients map[string]chanExtendedInfo,
	mu *sync.Mutex,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	conn, err := chain.ChainProvider.QueryConnection(cmd.Context(), 0, connectionID)
	if err != nil {
		return
	}
	clientState, err := chain.ChainProvider.QueryClientStateResponse(
		cmd.Context(), 0, conn.Connection.ClientId,
	)
	if err != nil {
		return
	}
	clientInfo, err := relayer.ClientInfoFromClientState(clientState.ClientState)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	connectionClients[connectionID] = chanExtendedInfo{
		clientID:             conn.Connection.ClientId,
		counterpartyClientID: conn.Connection.Counterparty.ClientId,
		counterpartyConnID:   conn.Connection.Counterparty.ConnectionId,
		counterpartyChainID:  clientInfo.ChainID,
	}
}

func printChannelsWithConnectionClients(
	cmd *cobra.Command,
	chain *relayer.Chain,
	channels []*chantypes.IdentifiedChannel,
	connectionClients map[string]chanExtendedInfo,
) {
	for _, channel := range channels {
		// Keep indexing the first hop before the lookup. Existing callers rely on
		// the panic for malformed channels with no connection hops.
		chanInfo, ok := connectionClients[channel.ConnectionHops[0]]
		if !ok {
			printChannelWithExtendedInfo(cmd, chain, channel, nil)
			continue
		}
		printChannelWithExtendedInfo(cmd, chain, channel, &chanInfo)
	}
}

func queryChannels(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels [src_chain_name] [dst_chain_name]?",
		Short: "query for all channels on a network by chain ID",
		Args:  withUsage(cobra.RangeArgs(1, 2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query channels ibc-0
$ %s query channels ibc-2 --offset 2 --limit 30
$ %s query channels ibc-0 ibc-2`,
			appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			if len(args) > 1 {
				dstChain, ok := a.config.Chains[args[1]]
				if !ok {
					return errChainNotFound(args[1])
				}
				return queryChannelsToChain(cmd, chain, dstChain)
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			return queryChannelsPaginated(cmd, chain, pageReq)
		},
	}

	cmd = addOutputFlag(a.viper, cmd)
	cmd = paginationFlags(a.viper, cmd, "channels on a network")
	return cmd
}

func queryPacketCommitment(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packet-commit chain_name channel_id port_id seq",
		Short: "query for the packet commitment given a sequence and channel ID on a network by chain ID",
		Args:  withUsage(cobra.ExactArgs(4)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query packet-commit ibc-0 ibczerochannel transfer 32
$ %s q packet-commit ibc-1 ibconechannel transfer 31`,
			appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, ok := a.config.Chains[args[0]]
			if !ok {
				return errChainNotFound(args[0])
			}

			channelID := args[1]
			portID := args[2]

			seq, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return err
			}

			res, err := chain.ChainProvider.QueryPacketCommitment(cmd.Context(), 0, channelID, portID, seq)
			if err != nil {
				return err
			}

			s, err := chain.ChainProvider.Sprint(res)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to marshal packet-commit state: %v\n", err)
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func queryUnrelayedPackets(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unrelayed-packets path src_channel_id",
		Aliases: []string{"unrelayed-pkts"},
		Short:   "query for the packet sequence numbers that remain to be relayed on a given path",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s q unrelayed-packets demo-path channel-0
$ %s query unrelayed-packets demo-path channel-0
$ %s query unrelayed-pkts demo-path channel-0`,
			appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryUnrelayedPackets(a, cmd, args)
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func runQueryUnrelayedPackets(a *appState, cmd *cobra.Command, args []string) error {
	src, dst, err := setupQueryPath(a, args[0])
	if err != nil {
		return err
	}
	channel, err := relayer.QueryChannel(cmd.Context(), src, args[1])
	if err != nil {
		return err
	}
	sequences := relayer.UnrelayedSequences(cmd.Context(), src, dst, channel)
	out, err := json.Marshal(sequences)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}

func queryUnrelayedAcknowledgements(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unrelayed-acknowledgements path src_channel_id",
		Aliases: []string{"unrelayed-acks"},
		Short:   "query for unrelayed acknowledgement sequence numbers that remain to be relayed on a given path",
		Args:    withUsage(cobra.ExactArgs(2)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s q unrelayed-acknowledgements demo-path channel-0
$ %s query unrelayed-acknowledgements demo-path channel-0
$ %s query unrelayed-acks demo-path channel-0`,
			appName, appName, appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryUnrelayedAcknowledgements(a, cmd, args)
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func runQueryUnrelayedAcknowledgements(a *appState, cmd *cobra.Command, args []string) error {
	src, dst, err := setupQueryPath(a, args[0])
	if err != nil {
		return err
	}
	channel, err := relayer.QueryChannel(cmd.Context(), src, args[1])
	if err != nil {
		return err
	}
	sequences := relayer.UnrelayedAcknowledgements(cmd.Context(), src, dst, channel)
	out, err := json.Marshal(sequences)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}

func setupQueryPath(a *appState, pathName string) (*relayer.Chain, *relayer.Chain, error) {
	path, err := a.config.Paths.Get(pathName)
	if err != nil {
		return nil, nil, err
	}
	src, dst := path.Src.ChainID, path.Dst.ChainID
	chains, err := a.config.Chains.Gets(src, dst)
	if err != nil {
		return nil, nil, err
	}
	if err = chains[src].SetPath(path.Src); err != nil {
		return nil, nil, err
	}
	if err = chains[dst].SetPath(path.Dst); err != nil {
		return nil, nil, err
	}
	return chains[src], chains[dst], nil
}

func queryClientsExpiration(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clients-expiration path",
		Aliases: []string{"ce"},
		Short:   "query for light clients expiration date",
		Args:    withUsage(cobra.ExactArgs(1)),
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s query clients-expiration demo-path`,
			appName,
		)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryClientsExpiration(a, cmd, args)
		},
	}
	cmd = addOutputFlag(a.viper, cmd)
	return cmd
}

func runQueryClientsExpiration(a *appState, cmd *cobra.Command, args []string) error {
	src, dst, err := setupQueryPath(a, args[0])
	if err != nil {
		return err
	}
	srcExpiration, srcClientInfo, errSrc := relayer.QueryClientExpiration(cmd.Context(), src, dst)
	if errSrc != nil && !strings.Contains(errSrc.Error(), "light client not found") {
		return errSrc
	}
	dstExpiration, dstClientInfo, errDst := relayer.QueryClientExpiration(cmd.Context(), dst, src)
	if errDst != nil && !strings.Contains(errDst.Error(), "light client not found") {
		return errDst
	}

	output, _ := cmd.Flags().GetString(flagOutput)
	srcClientExpiration := relayer.SPrintClientExpiration(src, srcExpiration, srcClientInfo)
	dstClientExpiration := relayer.SPrintClientExpiration(dst, dstExpiration, dstClientInfo)
	if output == formatJson {
		srcClientExpiration = relayer.SPrintClientExpirationJson(src, srcExpiration, srcClientInfo)
		dstClientExpiration = relayer.SPrintClientExpirationJson(dst, dstExpiration, dstClientInfo)
	}
	if errSrc == nil {
		fmt.Fprintln(cmd.OutOrStdout(), srcClientExpiration)
	}
	if errDst == nil {
		fmt.Fprintln(cmd.OutOrStdout(), dstClientExpiration)
	}
	return nil
}
