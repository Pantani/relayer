package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestTxCommandsCharacterizeMetadataDefaultsAndArgs(t *testing.T) {
	state := txCharacterizationState(nil, nil)

	create := createChannelCmd(state)
	assertTxCommandArgs(t, create, "channel path_name", []string{"chan"}, 1, 1)
	require.Equal(t, "10s", txStringFlag(t, create, flagTimeout))
	require.Equal(t, uint64(3), txUint64Flag(t, create, flagMaxRetries))
	require.False(t, txBoolFlag(t, create, flagOverride))
	require.Equal(t, "transfer", txStringFlag(t, create, flagSrcPort))
	require.Equal(t, "transfer", txStringFlag(t, create, flagDstPort))
	require.Equal(t, "unordered", txStringFlag(t, create, flagOrder))
	require.Equal(t, "ics20-1", txStringFlag(t, create, flagVersion))
	require.Empty(t, txStringFlag(t, create, flagMemo))

	closeCmd := closeChannelCmd(state)
	assertTxCommandArgs(t, closeCmd, "channel-close path_name src_channel_id src_port_id", nil, 3, 3)
	require.Equal(t, "10s", txStringFlag(t, closeCmd, flagTimeout))
	require.Equal(t, uint64(3), txUint64Flag(t, closeCmd, flagMaxRetries))
	require.Empty(t, txStringFlag(t, closeCmd, flagMemo))

	flush := flushCmd(state)
	assertTxCommandArgs(t, flush, "flush [path_name]? [src_channel_id]?", []string{"relay-pkts"}, 0, 2)
	require.Equal(t, uint64(relayer.DefaultMaxMsgLength), txUint64Flag(t, flush, flagMaxMsgLength))
	require.Empty(t, txStringFlag(t, flush, flagMemo))
	require.Empty(t, txStringFlag(t, flush, flagStuckPacketChainID))
	require.Zero(t, txUint64Flag(t, flush, flagStuckPacketHeightStart))
	require.Zero(t, txUint64Flag(t, flush, flagStuckPacketHeightEnd))

	transfer := xfersend(state)
	assertTxCommandArgs(t, transfer, "transfer src_chain_name dst_chain_name amount dst_addr src_channel_id", nil, 5, 5)
	require.Empty(t, txStringFlag(t, transfer, flagPath))
	require.Empty(t, txStringFlag(t, transfer, flagMemo))
	require.Zero(t, txUint64Flag(t, transfer, flagTimeoutHeightOffset))
	require.Zero(t, txDurationFlag(t, transfer, flagTimeoutTimeOffset))
}

func TestSetPathsFromArgsCharacterizesSinglePathSelectionAndIdentity(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"only": path})

	selected, err := setPathsFromArgs(state, src, dst, "")
	require.NoError(t, err)
	require.Same(t, path, selected)
	require.Same(t, path.Src, src.PathEnd)
	require.Same(t, path.Dst, dst.PathEnd)

	src.PathEnd, dst.PathEnd = nil, nil
	selected, err = setPathsFromArgs(state, src, dst, "only")
	require.NoError(t, err)
	require.Same(t, path, selected)
	require.Same(t, path.Src, src.PathEnd)
	require.Same(t, path.Dst, dst.PathEnd)
}

func TestSetPathsFromArgsCharacterizesMultiplePathChoiceAndAmbiguity(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	first := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	second := txCharacterizationPath("chain-b-1", "chain-a-1", protocol.ProtocolClassic)
	state := txCharacterizationState(
		relayer.Chains{"alpha": src, "beta": dst},
		relayer.Paths{"first": first, "second": second},
	)

	selected, err := setPathsFromArgs(state, src, dst, "")
	require.EqualError(t, err, "more than one path between chain-a-1 and chain-b-1 exists, pass in path name")
	require.Nil(t, selected)
	require.Nil(t, src.PathEnd)
	require.Nil(t, dst.PathEnd)

	selected, err = setPathsFromArgs(state, src, dst, "second")
	require.NoError(t, err)
	require.Same(t, second, selected)
	require.Same(t, second.Dst, src.PathEnd)
	require.Same(t, second.Src, dst.PathEnd)
}

func TestSetPathsFromArgsCharacterizesLookupErrorsBeforeMutation(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{})

	selected, err := setPathsFromArgs(state, src, dst, "")
	require.EqualError(t, err, "failed to find path in config between chains chain-a-1 and chain-b-1")
	require.Nil(t, selected)
	require.Nil(t, src.PathEnd)
	require.Nil(t, dst.PathEnd)

	state.config.Paths["only"] = txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	selected, err = setPathsFromArgs(state, src, dst, "missing")
	require.EqualError(t, err, "path with name missing does not exist")
	require.Nil(t, selected)
	require.Nil(t, src.PathEnd)
	require.Nil(t, dst.PathEnd)
}

func TestSetPathsFromArgsCharacterizesPartialMutationOnSecondInvalidEnd(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	path.Dst.ClientID = "invalid client id"
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"only": path})

	selected, err := setPathsFromArgs(state, src, dst, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "path on chain chain-b-1 failed to set")
	require.Nil(t, selected)
	require.Same(t, path.Src, src.PathEnd)
	require.Nil(t, dst.PathEnd)
}

func TestSetPathsFromArgsCharacterizesCurrentV2Acceptance(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolV2)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"v2": path})

	selected, err := setPathsFromArgs(state, src, dst, "v2")
	require.NoError(t, err)
	require.Same(t, path, selected)
	require.Same(t, path.Src, src.PathEnd)
	require.Same(t, path.Dst, dst.PathEnd)
}

func TestCreateChannelCharacterizesFlagGetterOrder(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	target := createChannelCmd(state)

	cmd := &cobra.Command{Use: "characterize"}
	err := target.RunE(cmd, []string{"demo"})
	require.EqualError(t, err, "flag accessed but not defined: override")

	cmd.Flags().Bool(flagOverride, false, "")
	err = target.RunE(cmd, []string{"demo"})
	require.EqualError(t, err, "flag accessed but not defined: src-port")

	cmd.Flags().String(flagSrcPort, "transfer", "")
	err = target.RunE(cmd, []string{"demo"})
	require.EqualError(t, err, "flag accessed but not defined: dst-port")

	cmd.Flags().String(flagDstPort, "transfer", "")
	err = target.RunE(cmd, []string{"demo"})
	require.EqualError(t, err, "flag accessed but not defined: order")
}

func TestCreateChannelCharacterizesKeysBeforeChannelValidation(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.key, dstProvider.key = "alice", "bob"
	srcProvider.keyExists, dstProvider.keyExists = true, true
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	cmd := createChannelCmd(state)
	require.NoError(t, cmd.Flags().Set(flagOverride, "true"))
	require.NoError(t, cmd.Flags().Set(flagOrder, "invalid"))
	require.NoError(t, cmd.Flags().Set(flagMemo, "observed memo"))

	err := cmd.RunE(cmd, []string{"demo"})
	require.EqualError(t, err, "invalid order input (invalid), order must be 'ordered' or 'unordered'")
	require.Equal(t, []string{"Key()", "KeyExists(alice)"}, srcProvider.callsSnapshot())
	require.Equal(t, []string{"Key()", "KeyExists(bob)"}, dstProvider.callsSnapshot())
	require.Same(t, path.Src, src.PathEnd)
	require.Same(t, path.Dst, dst.PathEnd)
}

func TestCreateChannelCharacterizesExistingSourceChannelShortCircuit(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.keyExists, dstProvider.keyExists = true, true
	srcProvider.connectionChannels = []*chantypes.IdentifiedChannel{{
		ChannelId: "channel-7",
		PortId:    "transfer",
	}}
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})

	cmd := createChannelCmd(state)
	cmd.SetContext(context.Background())
	err := cmd.RunE(cmd, []string{"demo"})
	require.EqualError(t, err, "channel {channel-7} with port {transfer} already exists on chain {chain-a-1}")
	require.Contains(t, srcProvider.callsSnapshot(), "QueryLatestHeight()")
	require.Contains(t, srcProvider.callsSnapshot(), "QueryConnectionChannels(41,connection-0)")
	require.NotContains(t, strings.Join(dstProvider.callsSnapshot(), "\n"), "QueryLatestHeight")
}

func TestCreateChannelCharacterizesSourceKeyFailureBeforeDestination(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.key, dstProvider.key = "alice", "bob"
	dstProvider.keyExists = true
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})

	err := createChannelCmd(state).RunE(createChannelCmd(state), []string{"demo"})
	require.EqualError(t, err, "key alice not found on src chain chain-a-1")
	require.Equal(t, []string{"Key()", "KeyExists(alice)", "Key()"}, srcProvider.callsSnapshot())
	require.Empty(t, dstProvider.callsSnapshot())
}

func TestCloseChannelCharacterizesProviderOrderAndQueryError(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.keyExists, dstProvider.keyExists = true, true
	srcProvider.queryChannelErr = errors.New("channel query failed")
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	cmd := closeChannelCmd(state)
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{"demo", "channel-9", "transfer"})
	require.EqualError(t, err, "channel query failed")
	require.Equal(t, []string{
		"Key()",
		"KeyExists(default-key)",
		"QueryLatestHeight()",
		"QueryChannel(41,channel-9,transfer)",
	}, srcProvider.callsSnapshot())
	require.Equal(t, []string{"Key()", "KeyExists(default-key)"}, dstProvider.callsSnapshot())
}

func TestCloseChannelCharacterizesDestinationKeyFailureBeforeHeight(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.keyExists = true
	dstProvider.key = "bob"
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})

	err := closeChannelCmd(state).RunE(closeChannelCmd(state), []string{"demo", "channel-9", "transfer"})
	require.EqualError(t, err, "key bob not found on dst chain chain-b-1")
	require.NotContains(t, strings.Join(srcProvider.callsSnapshot(), "\n"), "QueryLatestHeight")
	require.Equal(t, []string{"Key()", "KeyExists(bob)", "Key()"}, dstProvider.callsSnapshot())
}

func TestFlushCharacterizesPreflightAndFilterSideEffect(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.keyExists, dstProvider.keyExists = true, true
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	cmd := flushCmd(state)
	require.NoError(t, cmd.Flags().Set(flagStuckPacketChainID, "chain-a-1"))

	err := cmd.RunE(cmd, []string{"demo", "channel-8"})
	require.EqualError(t, err, "stuck packet chain ID chain-a-1 is set but start height is not")
	require.Equal(t, processor.RuleAllowList, path.Filter.Rule)
	require.Equal(t, []string{"channel-8"}, path.Filter.ChannelList)
	require.Equal(t, []string{"Key()", "KeyExists(default-key)"}, srcProvider.callsSnapshot())
	require.Equal(t, []string{"Key()", "KeyExists(default-key)"}, dstProvider.callsSnapshot())
}

func TestFlushCharacterizesMissingPathPanicAndMissingChainError(t *testing.T) {
	state := txCharacterizationState(nil, nil)
	require.PanicsWithError(t, "path with name missing does not exist", func() {
		_ = flushCmd(state).RunE(flushCmd(state), []string{"missing"})
	})

	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state.config.Paths["demo"] = path
	err := flushCmd(state).RunE(flushCmd(state), []string{"demo"})
	require.EqualError(t, err, "chain with ID chain-a-1 is not configured")
}

func TestDeprecatedPacketAndAckCommandsCharacterizeWarningsBeforeFlushError(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(nil, relayer.Paths{"demo": path})
	state.log = zap.New(core)

	err := relayMsgsCmd(state).RunE(relayMsgsCmd(state), []string{"demo", "channel-0"})
	require.EqualError(t, err, "chain with ID chain-a-1 is not configured")
	err = relayAcksCmd(state).RunE(relayAcksCmd(state), []string{"demo", "channel-0"})
	require.EqualError(t, err, "chain with ID chain-a-1 is not configured")
	require.Equal(t, []string{
		"This command is deprecated. Please use 'tx flush' command instead",
		"This command is deprecated. Please use 'tx flush' command instead",
	}, txObservedMessages(logs))
}

func TestTransferCharacterizesLookupAndAmountErrorOrder(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})

	tick := string(rune(96))
	err := xfersend(state).RunE(xfersend(state), []string{"missing", "beta", "1uatom", "addr", "channel-0"})
	require.EqualError(t, err, "chain with name \"missing\" not found in config. consider running "+tick+"rly chains add missing"+tick)
	err = xfersend(state).RunE(xfersend(state), []string{"alpha", "missing", "1uatom", "addr", "channel-0"})
	require.EqualError(t, err, "chain with name \"missing\" not found in config. consider running "+tick+"rly chains add missing"+tick)

	cmd := xfersend(state)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set(flagPath, "demo"))
	err = cmd.RunE(cmd, []string{"alpha", "beta", "not-a-coin", "addr", "channel-0"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid decimal coin expression")
	require.Same(t, path.Src, src.PathEnd)
	require.Same(t, path.Dst, dst.PathEnd)
}

func TestTransferCharacterizesDenomRawReceiverTimeoutsMemoAndBroadcast(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.connectionChannels = []*chantypes.IdentifiedChannel{{
		ChannelId: "channel-4",
		PortId:    "transfer",
	}}
	srcProvider.denomTraces = []transfertypes.Denom{
		transfertypes.ParseDenomTrace("transfer/channel-9/uatom"),
	}
	srcProvider.clientState = &tmclient.ClientState{
		ChainId:      "chain-b-1",
		LatestHeight: clienttypes.NewHeight(3, 70),
	}
	srcProvider.sendSuccess = true
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	state.config.Global.Memo = "config memo"
	cmd := xfersend(state)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set(flagPath, "demo"))
	require.NoError(t, cmd.Flags().Set(flagTimeoutHeightOffset, "9"))
	require.NoError(t, cmd.Flags().Set(flagMemo, "flag memo"))

	err := cmd.RunE(cmd, []string{
		"alpha",
		"beta",
		"5transfer/channel-9/uatom",
		"raw:destination-without-bech32",
		"channel-4",
	})
	require.NoError(t, err)
	require.Equal(t, "destination-without-bech32", srcProvider.transferAddress)
	require.Equal(t, sdk.NewInt64Coin(srcProvider.denomTraces[0].IBCDenom(), 5), srcProvider.transferAmount)
	require.Equal(t, "channel-4", srcProvider.transferPacket.SourceChannel)
	require.Equal(t, "transfer", srcProvider.transferPacket.SourcePort)
	require.Equal(t, clienttypes.NewHeight(3, 79), srcProvider.transferPacket.TimeoutHeight)
	require.Zero(t, srcProvider.transferPacket.TimeoutTimestamp)
	require.Equal(t, "flag memo | rly()", srcProvider.sendMemo)
	require.Equal(t, 1, srcProvider.sendCount)
	require.Contains(t, srcProvider.callsSnapshot(), "QueryConnectionChannels(41,connection-0)")
	require.Contains(t, srcProvider.callsSnapshot(), "QueryDenomTraces(0,100,41)")
	require.Contains(t, srcProvider.callsSnapshot(), "QueryClientState(41,07-tendermint-0)")
	require.Contains(t, dstProvider.callsSnapshot(), "QueryLatestHeight()")
}

func TestTransferCharacterizesNegativeTimeOffsetAfterChannelAndDenomQueries(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	srcProvider.connectionChannels = []*chantypes.IdentifiedChannel{{
		ChannelId: "channel-4",
		PortId:    "transfer",
	}}
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	cmd := xfersend(state)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set(flagPath, "demo"))
	require.NoError(t, cmd.Flags().Set(flagTimeoutTimeOffset, "-1s"))

	err := cmd.RunE(cmd, []string{"alpha", "beta", "5uatom", "cosmos1dest", "channel-4"})
	require.EqualError(t, err, "transfer timeout time offset cannot be negative: -1s")
	require.Equal(t, 1, countTxCalls(srcProvider.callsSnapshot(), "QueryLatestHeight()"))
	require.Contains(t, srcProvider.callsSnapshot(), "QueryConnectionChannels(41,connection-0)")
	require.Contains(t, srcProvider.callsSnapshot(), "QueryDenomTraces(0,100,41)")
	require.NotContains(t, strings.Join(srcProvider.callsSnapshot(), "\n"), "MsgTransfer")
}

func TestTransferCharacterizesPositiveTimeOffsetWithLocalhostClients(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	dstProvider := txProvider(dst)
	srcProvider.connectionChannels = []*chantypes.IdentifiedChannel{{
		ChannelId: "channel-4",
		PortId:    "transfer",
	}}
	srcProvider.sendSuccess = true
	dstProvider.blockTime = time.Date(2100, time.January, 2, 3, 4, 5, 0, time.UTC)
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	path.Src.ClientID = ibcexported.LocalhostClientID
	path.Dst.ClientID = ibcexported.LocalhostClientID
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	cmd := xfersend(state)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set(flagPath, "demo"))
	require.NoError(t, cmd.Flags().Set(flagTimeoutTimeOffset, "2s"))

	err := cmd.RunE(cmd, []string{"alpha", "beta", "5uatom", "cosmos1dest", "channel-4"})
	require.NoError(t, err)
	require.Equal(t, clienttypes.NewHeight(1, 0), srcProvider.transferPacket.TimeoutHeight)
	wantTimestamp := uint64(dstProvider.blockTime.Add(2 * time.Second).UnixNano())
	require.Equal(t, wantTimestamp, srcProvider.transferPacket.TimeoutTimestamp)
	require.Equal(t, "cosmos1dest", srcProvider.transferAddress)
	require.NotContains(t, strings.Join(srcProvider.callsSnapshot(), "\n"), "QueryClientState")
	require.Contains(t, dstProvider.callsSnapshot(), "BlockTime(52)")
}

func TestTransferCharacterizesMissingChannelBeforeDenomAndTimeoutFlags(t *testing.T) {
	src, dst := txCharacterizationChains("chain-a-1", "chain-b-1")
	srcProvider := txProvider(src)
	path := txCharacterizationPath("chain-a-1", "chain-b-1", protocol.ProtocolClassic)
	state := txCharacterizationState(relayer.Chains{"alpha": src, "beta": dst}, relayer.Paths{"demo": path})
	cmd := xfersend(state)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set(flagPath, "demo"))

	err := cmd.RunE(cmd, []string{"alpha", "beta", "5uatom", "cosmos1dest", "channel-missing"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find channel{channel-missing}")
	require.Contains(t, err.Error(), "@connection{connection-0}")
	require.NotContains(t, strings.Join(srcProvider.callsSnapshot(), "\n"), "QueryDenomTraces")
}

type txCharacterizationProvider struct {
	provider.ChainProvider
	mu                 sync.Mutex
	chainID            string
	key                string
	keyExists          bool
	latestHeight       int64
	calls              []string
	connectionChannels []*chantypes.IdentifiedChannel
	queryChannelErr    error
	denomTraces        []transfertypes.Denom
	clientState        ibcexported.ClientState
	blockTime          time.Time
	transferAddress    string
	transferAmount     sdk.Coin
	transferPacket     provider.PacketInfo
	sendMemo           string
	sendSuccess        bool
	sendCount          int
}

func (p *txCharacterizationProvider) ChainId() string {
	return p.chainID
}

func (p *txCharacterizationProvider) record(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, fmt.Sprintf(format, args...))
}

func (p *txCharacterizationProvider) callsSnapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.calls...)
}

func (p *txCharacterizationProvider) Key() string {
	p.record("Key()")
	return p.key
}

func (p *txCharacterizationProvider) KeyExists(key string) bool {
	p.record("KeyExists(%s)", key)
	return p.keyExists
}

func (p *txCharacterizationProvider) QueryLatestHeight(context.Context) (int64, error) {
	p.record("QueryLatestHeight()")
	return p.latestHeight, nil
}

func (p *txCharacterizationProvider) QueryConnectionChannels(
	_ context.Context,
	height int64,
	connectionID string,
) ([]*chantypes.IdentifiedChannel, error) {
	p.record("QueryConnectionChannels(%d,%s)", height, connectionID)
	return p.connectionChannels, nil
}

func (p *txCharacterizationProvider) QueryChannel(
	_ context.Context,
	height int64,
	channelID, portID string,
) (*chantypes.QueryChannelResponse, error) {
	p.record("QueryChannel(%d,%s,%s)", height, channelID, portID)
	return &chantypes.QueryChannelResponse{}, p.queryChannelErr
}

func (p *txCharacterizationProvider) QueryDenomTraces(
	_ context.Context,
	offset, limit uint64,
	height int64,
) ([]transfertypes.Denom, error) {
	p.record("QueryDenomTraces(%d,%d,%d)", offset, limit, height)
	return p.denomTraces, nil
}

func (p *txCharacterizationProvider) QueryClientState(
	_ context.Context,
	height int64,
	clientID string,
) (ibcexported.ClientState, error) {
	p.record("QueryClientState(%d,%s)", height, clientID)
	return p.clientState, nil
}

func (p *txCharacterizationProvider) BlockTime(_ context.Context, height int64) (time.Time, error) {
	p.record("BlockTime(%d)", height)
	return p.blockTime, nil
}

func (p *txCharacterizationProvider) MsgTransfer(
	address string,
	amount sdk.Coin,
	packet provider.PacketInfo,
) (provider.RelayerMessage, error) {
	p.record("MsgTransfer(%s,%s)", address, amount.String())
	p.transferAddress = address
	p.transferAmount = amount
	p.transferPacket = packet
	return txCharacterizationMessage{}, nil
}

func (p *txCharacterizationProvider) SendMessages(
	_ context.Context,
	messages []provider.RelayerMessage,
	memo string,
) (*provider.RelayerTxResponse, bool, error) {
	p.record("SendMessages(%d,%s)", len(messages), memo)
	p.sendMemo = memo
	p.sendCount++
	return &provider.RelayerTxResponse{}, p.sendSuccess, nil
}

type txCharacterizationMessage struct{}

func (txCharacterizationMessage) Type() string {
	return "characterized-transfer"
}

func (txCharacterizationMessage) MsgBytes() ([]byte, error) {
	return []byte("transfer"), nil
}

func txCharacterizationChains(srcID, dstID string) (*relayer.Chain, *relayer.Chain) {
	srcProvider := &txCharacterizationProvider{chainID: srcID, key: "default-key", latestHeight: 41}
	dstProvider := &txCharacterizationProvider{chainID: dstID, key: "default-key", latestHeight: 52}
	src := relayer.NewChain(zap.NewNop(), srcProvider, false)
	dst := relayer.NewChain(zap.NewNop(), dstProvider, false)
	return src, dst
}

func txProvider(chain *relayer.Chain) *txCharacterizationProvider {
	return chain.ChainProvider.(*txCharacterizationProvider)
}

func txCharacterizationPath(srcID, dstID string, pathProtocol protocol.Protocol) *relayer.Path {
	return &relayer.Path{
		Protocol: pathProtocol,
		Src: &relayer.PathEnd{
			ChainID:      srcID,
			ClientID:     "07-tendermint-0",
			ConnectionID: "connection-0",
		},
		Dst: &relayer.PathEnd{
			ChainID:      dstID,
			ClientID:     "07-tendermint-1",
			ConnectionID: "connection-1",
		},
	}
}

func txCharacterizationState(chains relayer.Chains, paths relayer.Paths) *appState {
	if chains == nil {
		chains = relayer.Chains{}
	}
	if paths == nil {
		paths = relayer.Paths{}
	}
	return &appState{
		viper:  viper.New(),
		log:    zap.NewNop(),
		config: &Config{Chains: chains, Paths: paths},
	}
}

func assertTxCommandArgs(t *testing.T, cmd *cobra.Command, use string, aliases []string, minArgs, maxArgs int) {
	t.Helper()
	require.Equal(t, use, cmd.Use)
	require.Equal(t, aliases, cmd.Aliases)
	if minArgs > 0 {
		require.Error(t, cmd.Args(cmd, make([]string, minArgs-1)))
	}
	require.NoError(t, cmd.Args(cmd, make([]string, minArgs)))
	require.Error(t, cmd.Args(cmd, make([]string, maxArgs+1)))
}

func txStringFlag(t *testing.T, cmd *cobra.Command, name string) string {
	t.Helper()
	value, err := cmd.Flags().GetString(name)
	require.NoError(t, err)
	return value
}

func txUint64Flag(t *testing.T, cmd *cobra.Command, name string) uint64 {
	t.Helper()
	value, err := cmd.Flags().GetUint64(name)
	require.NoError(t, err)
	return value
}

func txBoolFlag(t *testing.T, cmd *cobra.Command, name string) bool {
	t.Helper()
	value, err := cmd.Flags().GetBool(name)
	require.NoError(t, err)
	return value
}

func txDurationFlag(t *testing.T, cmd *cobra.Command, name string) time.Duration {
	t.Helper()
	value, err := cmd.Flags().GetDuration(name)
	require.NoError(t, err)
	return value
}

func txObservedMessages(logs *observer.ObservedLogs) []string {
	entries := logs.All()
	messages := make([]string, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, entry.Message)
	}
	return messages
}

func countTxCalls(calls []string, target string) int {
	count := 0
	for _, call := range calls {
		if call == target {
			count++
		}
	}
	return count
}
