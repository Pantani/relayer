package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type queryCharacterizationProvider struct {
	provider.ChainProvider
	mu                   sync.Mutex
	chainID              string
	key                  string
	address              string
	coins                sdk.Coins
	height               int64
	header               provider.IBCHeader
	heightErr            error
	headerErr            error
	keyExists            bool
	showAddrErr          error
	balanceErr           error
	requestedKey         []string
	heights              []int64
	sprint               string
	sprintErr            error
	clientRes            *clienttypes.QueryClientStateResponse
	connections          *conntypes.QueryConnectionsResponse
	connectionsByClient  map[string]*conntypes.QueryConnectionsResponse
	connectionByID       map[string]*conntypes.QueryConnectionResponse
	channelRes           *chantypes.QueryChannelResponse
	channels             []*chantypes.IdentifiedChannel
	channelsByConnection map[string][]*chantypes.IdentifiedChannel
	clients              clienttypes.IdentifiedClientStates
	clientByID           map[string]*clienttypes.QueryClientStateResponse
	queryCalls           []string
	queryClientsErr      error
	queryChannelsErr     error
	clientErr            error
	connectionDelay      time.Duration
	activeConnections    atomic.Int32
	maxConnections       atomic.Int32
	packetCommitments    *chantypes.QueryPacketCommitmentsResponse
	packetAcks           []*chantypes.PacketState
	unreceivedPackets    []uint64
	unreceivedAcks       []uint64
	blockTimes           map[int64]time.Time
}

func (p *queryCharacterizationProvider) ChainId() string { return p.chainID }
func (p *queryCharacterizationProvider) Key() string     { return p.key }
func (p *queryCharacterizationProvider) KeyExists(key string) bool {
	p.requestedKey = append(p.requestedKey, key)
	return p.keyExists
}
func (p *queryCharacterizationProvider) ShowAddress(key string) (string, error) {
	p.requestedKey = append(p.requestedKey, "address:"+key)
	return p.address, p.showAddrErr
}
func (p *queryCharacterizationProvider) QueryBalanceWithAddress(context.Context, string) (sdk.Coins, error) {
	return p.coins, p.balanceErr
}
func (p *queryCharacterizationProvider) QueryLatestHeight(context.Context) (int64, error) {
	return p.height, p.heightErr
}
func (p *queryCharacterizationProvider) QueryIBCHeader(_ context.Context, height int64) (provider.IBCHeader, error) {
	p.heights = append(p.heights, height)
	return p.header, p.headerErr
}
func (p *queryCharacterizationProvider) Sprint(message proto.Message) (string, error) {
	if channel, ok := message.(*chantypes.IdentifiedChannel); ok && p.sprint == "channel-json" {
		return fmt.Sprintf(`{"channel_id":%q,"counterparty":{}}`, channel.ChannelId), p.sprintErr
	}
	return p.sprint, p.sprintErr
}
func (p *queryCharacterizationProvider) QueryClientStateResponse(_ context.Context, height int64, clientID string) (*clienttypes.QueryClientStateResponse, error) {
	p.recordQuery("client", height, clientID)
	if response, ok := p.clientByID[clientID]; ok {
		return response, p.clientErr
	}
	return p.clientRes, p.clientErr
}
func (p *queryCharacterizationProvider) QueryConnectionsUsingClient(_ context.Context, height int64, clientID string) (*conntypes.QueryConnectionsResponse, error) {
	p.recordQuery("client-connections", height, clientID)
	if response, ok := p.connectionsByClient[clientID]; ok {
		return response, nil
	}
	return p.connections, nil
}
func (p *queryCharacterizationProvider) QueryChannel(_ context.Context, height int64, channelID, portID string) (*chantypes.QueryChannelResponse, error) {
	p.recordQuery("channel", height, channelID+":"+portID)
	return p.channelRes, nil
}
func (p *queryCharacterizationProvider) QueryConnectionChannels(_ context.Context, height int64, connectionID string) ([]*chantypes.IdentifiedChannel, error) {
	p.recordQuery("connection-channels", height, connectionID)
	if channels, ok := p.channelsByConnection[connectionID]; ok {
		return channels, nil
	}
	return p.channels, nil
}
func (p *queryCharacterizationProvider) QueryClients(context.Context) (clienttypes.IdentifiedClientStates, error) {
	return p.clients, p.queryClientsErr
}
func (p *queryCharacterizationProvider) QueryChannels(context.Context) ([]*chantypes.IdentifiedChannel, error) {
	return p.channels, p.queryChannelsErr
}
func (p *queryCharacterizationProvider) QueryConnection(_ context.Context, height int64, connectionID string) (*conntypes.QueryConnectionResponse, error) {
	p.recordQuery("connection", height, connectionID)
	active := p.activeConnections.Add(1)
	defer p.activeConnections.Add(-1)
	for {
		maximum := p.maxConnections.Load()
		if active <= maximum || p.maxConnections.CompareAndSwap(maximum, active) {
			break
		}
	}
	if p.connectionDelay > 0 {
		time.Sleep(p.connectionDelay)
	}
	return p.connectionByID[connectionID], nil
}
func (p *queryCharacterizationProvider) QueryPacketCommitments(_ context.Context, height uint64, channelID, portID string) (*chantypes.QueryPacketCommitmentsResponse, error) {
	p.recordQuery("packet-commitments", int64(height), channelID+":"+portID)
	return p.packetCommitments, nil
}
func (p *queryCharacterizationProvider) QueryPacketAcknowledgements(_ context.Context, height uint64, channelID, portID string) ([]*chantypes.PacketState, error) {
	p.recordQuery("packet-acks", int64(height), channelID+":"+portID)
	return p.packetAcks, nil
}
func (p *queryCharacterizationProvider) QueryUnreceivedPackets(_ context.Context, height uint64, channelID, portID string, sequences []uint64) ([]uint64, error) {
	p.recordQuery("unreceived-packets", int64(height), fmt.Sprintf("%s:%s:%v", channelID, portID, sequences))
	return p.unreceivedPackets, nil
}
func (p *queryCharacterizationProvider) QueryUnreceivedAcknowledgements(_ context.Context, height uint64, channelID, portID string, sequences []uint64) ([]uint64, error) {
	p.recordQuery("unreceived-acks", int64(height), fmt.Sprintf("%s:%s:%v", channelID, portID, sequences))
	return p.unreceivedAcks, nil
}
func (p *queryCharacterizationProvider) BlockTime(_ context.Context, height int64) (time.Time, error) {
	p.recordQuery("block-time", height, "")
	return p.blockTimes[height], nil
}
func (p *queryCharacterizationProvider) recordQuery(kind string, height int64, id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queryCalls = append(p.queryCalls, fmt.Sprintf("%s:%d:%s", kind, height, id))
}

type queryCharacterizationHeader struct {
	Marker string `json:"marker"`
}

func (h queryCharacterizationHeader) Height() uint64 { return 77 }
func (queryCharacterizationHeader) ConsensusState() ibcexported.ConsensusState {
	return nil
}
func (queryCharacterizationHeader) NextValidatorsHash() []byte { return nil }

func TestQueryBalanceAndBalancesMetadata(t *testing.T) {
	t.Parallel()
	state := queryCharacterizationState(nil)
	tests := []struct {
		cmd     *cobra.Command
		use     string
		aliases []string
		flags   []string
	}{
		{queryBalanceCmd(state), "balance chain_name [key_name]", []string{"bal"}, []string{flagOutput, flagIBCDenoms}},
		{queryBalancesCmd(state), "balances [chain-name...]", nil, []string{flagOutput, flagIBCDenoms, flagKeyName}},
		{queryHeaderCmd(state), "header chain_name [height]", nil, []string{flagOutput}},
	}
	for _, test := range tests {
		require.Equal(t, test.use, test.cmd.Use)
		require.Equal(t, test.aliases, test.cmd.Aliases)
		for _, name := range test.flags {
			require.NotNil(t, test.cmd.Flags().Lookup(name), name)
		}
	}
}

func TestQueryBalanceFreezesKeySelectionAndOutput(t *testing.T) {
	t.Parallel()
	provider := queryBalanceProvider()
	state := queryCharacterizationState(provider)

	legacy := runQueryCommand(t, queryBalanceCmd(state), []string{"chain-a"}, map[string]string{flagIBCDenoms: "true"})
	require.NoError(t, legacy.err)
	require.Equal(t, "address {cosmos1alice} balance {7uatom} \n", legacy.stdout)
	require.Equal(t, []string{"default", "address:default"}, provider.requestedKey)

	provider.requestedKey = nil
	jsonResult := runQueryCommand(t, queryBalanceCmd(state), []string{"chain-a", "override"}, map[string]string{
		flagIBCDenoms: "true",
		flagOutput:    formatJson,
	})
	require.NoError(t, jsonResult.err)
	require.JSONEq(t, `{"address":"cosmos1alice","balance":"7uatom"}`, jsonResult.stdout)
	require.Equal(t, []string{"override", "address:override"}, provider.requestedKey)
}

func TestQueryBalancesFreezesSelectionErrorsAndJSON(t *testing.T) {
	t.Parallel()
	provider := queryBalanceProvider()
	state := queryCharacterizationState(provider)

	result := runQueryCommand(t, queryBalancesCmd(state), []string{"chain-a"}, map[string]string{
		flagIBCDenoms: "true",
		flagKeyName:   "shared",
		flagOutput:    formatJson,
	})
	require.NoError(t, result.err)
	require.JSONEq(t, `{"cosmos1alice":"7uatom"}`, result.stdout)
	require.Equal(t, []string{"shared", "address:shared"}, provider.requestedKey)

	missing := runQueryCommand(t, queryBalancesCmd(state), []string{"chain-a", "missing"}, map[string]string{flagIBCDenoms: "true"})
	require.EqualError(t, missing.err, "chain with name \"chain-a\" not found in config. consider running `rly chains add chain-a`")
}

func TestQueryBalancePreservesProviderErrorOrder(t *testing.T) {
	t.Parallel()
	provider := queryBalanceProvider()
	provider.keyExists = false
	result := runQueryCommand(t, queryBalanceCmd(queryCharacterizationState(provider)), []string{"chain-a"}, nil)
	require.EqualError(t, result.err, "a key with name default doesn't exist")
	require.Equal(t, []string{"default"}, provider.requestedKey)

	provider = queryBalanceProvider()
	provider.showAddrErr = errors.New("show address failed")
	result = runQueryCommand(t, queryBalanceCmd(queryCharacterizationState(provider)), []string{"chain-a"}, nil)
	require.ErrorIs(t, result.err, provider.showAddrErr)
}

func TestQueryHeaderFreezesHeightAndFormats(t *testing.T) {
	t.Parallel()
	provider := queryBalanceProvider()
	provider.height = 321
	provider.header = queryCharacterizationHeader{Marker: "original"}
	state := queryCharacterizationState(provider)

	jsonResult := runQueryCommand(t, queryHeaderCmd(state), []string{"chain-a"}, map[string]string{flagOutput: formatJson})
	require.NoError(t, jsonResult.err)
	require.Equal(t, "{\"marker\":\"original\"}\n", jsonResult.stdout)
	require.Equal(t, []int64{321}, provider.heights)

	provider.heights = nil
	legacy := runQueryCommand(t, queryHeaderCmd(state), []string{"chain-a", "42"}, nil)
	require.NoError(t, legacy.err)
	require.Equal(t, "[123 34 109 97 114 107 101 114 34 58 34 111 114 105 103 105 110 97 108 34 125]\n", legacy.stdout)
	require.Equal(t, []int64{42}, provider.heights)
}

func TestQueryClientConnectionChannelMetadata(t *testing.T) {
	t.Parallel()
	state := queryCharacterizationState(nil)
	tests := []struct {
		cmd     *cobra.Command
		use     string
		aliases []string
		flags   []string
	}{
		{queryClientCmd(state), "client chain_name client_id", nil, []string{flagOutput, flagHeight}},
		{queryConnectionsUsingClient(state), "client-connections chain_name client_id", nil, []string{flagOutput, flagHeight}},
		{queryChannel(state), "channel chain_name channel_id port_id", nil, []string{flagOutput, flagHeight}},
		{queryConnectionChannels(state), "connection-channels chain_name connection_id", nil, []string{flagOutput, flagPage, flagPageKey, flagLimit, flagCountTotal, flagReverse}},
	}
	for _, test := range tests {
		require.Equal(t, test.use, test.cmd.Use)
		require.Equal(t, test.aliases, test.cmd.Aliases)
		for _, name := range test.flags {
			require.NotNil(t, test.cmd.Flags().Lookup(name), name)
		}
	}
}

func TestQueryClientFreezesPathHeightCallAndOutput(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	state := queryCharacterizationState(provider)
	result := runQueryCommand(t, queryClientCmd(state), []string{"chain-a", "07-tendermint-9"}, map[string]string{flagHeight: "17"})

	require.NoError(t, result.err)
	require.Equal(t, "rendered\n", result.stdout)
	require.Equal(t, []string{"client:17:07-tendermint-9"}, provider.queryCalls)
	chain := state.config.Chains["chain-a"]
	require.Equal(t, "07-tendermint-9", chain.PathEnd.ClientID)
	require.Equal(t, dcon, chain.PathEnd.ConnectionID)
}

func TestQueryConnectionsUsingClientFreezesLatestHeight(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	provider.height = 88
	state := queryCharacterizationState(provider)
	result := runQueryCommand(t, queryConnectionsUsingClient(state), []string{"chain-a", "07-tendermint-4"}, nil)

	require.NoError(t, result.err)
	require.Equal(t, "rendered\n", result.stdout)
	require.Equal(t, []string{"client-connections:88:07-tendermint-4"}, provider.queryCalls)
}

func TestQueryChannelFreezesExplicitArgumentsAndSprintError(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	provider.sprintErr = errors.New("sprint failed")
	result := runQueryCommand(t, queryChannel(queryCharacterizationState(provider)), []string{"chain-a", "channel-7", "transfer"}, map[string]string{flagHeight: "23"})

	require.ErrorIs(t, result.err, provider.sprintErr)
	require.Equal(t, []string{"channel:23:channel-7:transfer"}, provider.queryCalls)
	require.Equal(t, "Failed to marshal channel state: sprint failed\n", result.stderr)
}

func TestQueryConnectionChannelsContinuesAfterSprintFailure(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	provider.channels = []*chantypes.IdentifiedChannel{{ChannelId: "channel-1"}, {ChannelId: "channel-2"}}
	provider.sprintErr = errors.New("cannot print")
	state := queryCharacterizationState(provider)
	result := runQueryCommand(t, queryConnectionChannels(state), []string{"chain-a", "connection-3"}, nil)

	require.NoError(t, result.err)
	require.Empty(t, result.stdout)
	require.Equal(t, "Failed to marshal channel: cannot print\nFailed to marshal channel: cannot print\n", result.stderr)
	require.Equal(t, []string{"connection-channels:0:connection-3"}, provider.queryCalls)
	require.Equal(t, "connection-3", state.config.Chains["chain-a"].PathEnd.ConnectionID)
}

func TestQueryChannelsPaginatedEnrichesInInputOrder(t *testing.T) {
	t.Parallel()
	provider := queryPaginationProvider(t, []string{"connection-1", "connection-1"})
	chain := relayer.NewChain(zap.NewNop(), provider, false)
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := queryChannelsPaginated(cmd, chain, nil)
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	require.JSONEq(t, `{"chain_id":"chain-a-1","channel_id":"channel-0","client_id":"client-1","counterparty":{"chain_id":"chain-b-1","client_id":"counter-client-1","connection_id":"counter-connection-1"}}`, lines[0])
	require.JSONEq(t, `{"chain_id":"chain-a-1","channel_id":"channel-1","client_id":"client-1","counterparty":{"chain_id":"chain-b-1","client_id":"counter-client-1","connection_id":"counter-connection-1"}}`, lines[1])
	require.Equal(t, []string{"client:0:client-1", "connection:0:connection-1"}, sortedQueryCalls(provider))
}

func TestQueryChannelsPaginatedBatchesAtTen(t *testing.T) {
	t.Parallel()
	connections := make([]string, 12)
	for i := range connections {
		connections[i] = fmt.Sprintf("connection-%d", i)
	}
	provider := queryPaginationProvider(t, connections)
	provider.connectionDelay = 10 * time.Millisecond
	chain := relayer.NewChain(zap.NewNop(), provider, false)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, queryChannelsPaginated(cmd, chain, nil))
	require.Equal(t, int32(concurrentQueries), provider.maxConnections.Load())
}

func TestQueryChannelsPaginatedPreservesEmptyHopPanic(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	provider.channels = []*chantypes.IdentifiedChannel{{ChannelId: "channel-no-hop"}}
	chain := relayer.NewChain(zap.NewNop(), provider, false)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.Panics(t, func() { _ = queryChannelsPaginated(cmd, chain, nil) })
}

func TestQueryChannelsToChainFiltersClientsAndEnriches(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	provider.sprint = "channel-json"
	provider.clients = clienttypes.IdentifiedClientStates{
		{ClientId: "foreign", ClientState: queryPackedClient(t, "chain-c-1")},
		{ClientId: "matching", ClientState: queryPackedClient(t, "chain-b-1")},
		{ClientId: "malformed", ClientState: &codectypes.Any{}},
	}
	provider.connectionsByClient = map[string]*conntypes.QueryConnectionsResponse{
		"matching": {Connections: []*conntypes.IdentifiedConnection{{
			Id:           "connection-9",
			Counterparty: conntypes.Counterparty{ClientId: "counter-client-9", ConnectionId: "counter-connection-9"},
		}}},
	}
	provider.channelsByConnection = map[string][]*chantypes.IdentifiedChannel{
		"connection-9": {{ChannelId: "channel-9", ConnectionHops: []string{"connection-9"}}},
	}
	src := relayer.NewChain(zap.NewNop(), provider, false)
	dst := relayer.NewChain(zap.NewNop(), &queryCharacterizationProvider{chainID: "chain-b-1"}, false)
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, queryChannelsToChain(cmd, src, dst))
	require.JSONEq(t, `{"chain_id":"chain-a-1","channel_id":"channel-9","client_id":"matching","counterparty":{"chain_id":"chain-b-1","client_id":"counter-client-9","connection_id":"counter-connection-9"}}`, strings.TrimSpace(stdout.String()))
	require.Equal(t, []string{"client-connections:0:matching", "connection-channels:0:connection-9"}, sortedQueryCalls(provider))
}

func TestQueryChannelsTopLevelErrorsAndDestinationLookup(t *testing.T) {
	t.Parallel()
	provider := queryProtocolProvider()
	provider.queryClientsErr = errors.New("clients failed")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.ErrorIs(t, queryChannelsToChain(cmd, relayer.NewChain(zap.NewNop(), provider, false), relayer.NewChain(zap.NewNop(), provider, false)), provider.queryClientsErr)

	result := runQueryCommand(t, queryChannels(queryCharacterizationState(provider)), []string{"chain-a", "missing"}, nil)
	require.EqualError(t, result.err, "chain with name \"missing\" not found in config. consider running `rly chains add missing`")
}

func TestQueryUnrelayedMetadata(t *testing.T) {
	t.Parallel()
	state := queryCharacterizationState(nil)
	tests := []struct {
		cmd     *cobra.Command
		use     string
		aliases []string
	}{
		{queryUnrelayedPackets(state), "unrelayed-packets path src_channel_id", []string{"unrelayed-pkts"}},
		{queryUnrelayedAcknowledgements(state), "unrelayed-acknowledgements path src_channel_id", []string{"unrelayed-acks"}},
		{queryClientsExpiration(state), "clients-expiration path", []string{"ce"}},
	}
	for _, test := range tests {
		require.Equal(t, test.use, test.cmd.Use)
		require.Equal(t, test.aliases, test.cmd.Aliases)
		require.NotNil(t, test.cmd.Flags().Lookup(flagOutput))
	}
}

func TestQueryUnrelayedPacketsFreezesPathsCallsAndSequenceOrder(t *testing.T) {
	t.Parallel()
	state, srcProvider, dstProvider := queryUnrelayedState()
	srcProvider.packetCommitments = packetCommitments(1, 2)
	dstProvider.packetCommitments = packetCommitments(3, 4)
	srcProvider.unreceivedPackets = []uint64{4, 3}
	dstProvider.unreceivedPackets = []uint64{2, 1}

	result := runQueryCommand(t, queryUnrelayedPackets(state), []string{"demo", "channel-7"}, nil)
	require.NoError(t, result.err)
	require.Equal(t, "{\"src\":[2,1],\"dst\":[4,3]}\n", result.stdout)
	require.Equal(t, []string{
		"connection-channels:11:connection-a",
		"packet-commitments:11:channel-7:transfer",
		"unreceived-packets:11:channel-7:transfer:[3 4]",
	}, sortedQueryCalls(srcProvider))
	require.Equal(t, []string{
		"packet-commitments:22:channel-8:transfer-counter",
		"unreceived-packets:22:channel-8:transfer-counter:[1 2]",
	}, sortedQueryCalls(dstProvider))
	require.Equal(t, "client-a", state.config.Chains["src"].PathEnd.ClientID)
	require.Equal(t, "connection-b", state.config.Chains["dst"].PathEnd.ConnectionID)
}

func TestQueryUnrelayedAcknowledgementsFreezesCallsAndEmptySlices(t *testing.T) {
	t.Parallel()
	state, srcProvider, dstProvider := queryUnrelayedState()
	srcProvider.packetAcks = []*chantypes.PacketState{{Sequence: 9}, {Sequence: 7}}
	dstProvider.packetAcks = []*chantypes.PacketState{}
	dstProvider.unreceivedAcks = []uint64{7, 9}

	result := runQueryCommand(t, queryUnrelayedAcknowledgements(state), []string{"demo", "channel-7"}, nil)
	require.NoError(t, result.err)
	require.Equal(t, "{\"src\":[7,9],\"dst\":[]}\n", result.stdout)
	require.Contains(t, sortedQueryCalls(srcProvider), "packet-acks:11:channel-7:transfer")
	require.Equal(t, []string{
		"packet-acks:22:channel-8:transfer-counter",
		"unreceived-acks:22:channel-8:transfer-counter:[9 7]",
	}, sortedQueryCalls(dstProvider))
}

func TestQueryUnrelayedPreservesLookupErrors(t *testing.T) {
	t.Parallel()
	state := queryCharacterizationState(queryProtocolProvider())
	result := runQueryCommand(t, queryUnrelayedPackets(state), []string{"missing", "channel-0"}, nil)
	require.EqualError(t, result.err, "path with name missing does not exist")
}

func TestQueryClientsExpirationFreezesMathStatusAndOutputOrder(t *testing.T) {
	t.Parallel()
	state, srcProvider, dstProvider := queryExpirationState(t)
	result := runQueryCommand(t, queryClientsExpiration(state), []string{"demo"}, map[string]string{flagOutput: formatJson})

	require.NoError(t, result.err)
	lines := strings.Split(strings.TrimSpace(result.stdout), "\n")
	require.Len(t, lines, 2)
	srcFields := expirationFields(t, lines[0])
	dstFields := expirationFields(t, lines[1])
	require.Equal(t, "client-a (chain-a-1)", srcFields["client"])
	require.Equal(t, "GOOD", srcFields["HEALTH"])
	require.True(t, strings.HasPrefix(srcFields["TIME"], "01 Jan 30 02:00 UTC ("))
	require.Equal(t, "50", srcFields["LAST UPDATE HEIGHT"])
	require.Equal(t, "2h0m0s", srcFields["TRUSTING PERIOD"])
	require.Equal(t, "3h0m0s", srcFields["UNBONDING PERIOD"])
	require.Equal(t, "client-b (chain-b-1)", dstFields["client"])
	require.Equal(t, "EXPIRED", dstFields["HEALTH"])
	require.True(t, strings.HasPrefix(dstFields["TIME"], "01 Jan 20 04:00 UTC (-"))
	require.Equal(t, "60", dstFields["LAST UPDATE HEIGHT"])
	require.Equal(t, "4h0m0s", dstFields["TRUSTING PERIOD"])
	require.Equal(t, "5h0m0s", dstFields["UNBONDING PERIOD"])
	require.Equal(t, []string{"block-time:60:", "client:101:client-a"}, sortedQueryCalls(srcProvider))
	require.Equal(t, []string{"block-time:50:", "client:202:client-b"}, sortedQueryCalls(dstProvider))
}

func TestQueryClientsExpirationPreservesLightClientNotFoundPanic(t *testing.T) {
	t.Parallel()
	state, srcProvider, _ := queryExpirationState(t)
	srcProvider.clientErr = errors.New("light client not found on source")
	require.Panics(t, func() {
		_ = runQueryCommand(t, queryClientsExpiration(state), []string{"demo"}, map[string]string{flagOutput: formatJson})
	})

	state, srcProvider, _ = queryExpirationState(t)
	srcProvider.clientErr = errors.New("rpc unavailable")
	result := runQueryCommand(t, queryClientsExpiration(state), []string{"demo"}, nil)
	require.ErrorIs(t, result.err, srcProvider.clientErr)
}

type queryCommandResult struct {
	stdout string
	stderr string
	err    error
}

func runQueryCommand(t *testing.T, cmd *cobra.Command, args []string, flags map[string]string) queryCommandResult {
	t.Helper()
	for name, value := range flags {
		require.NoError(t, cmd.Flags().Set(name, value), name)
	}
	var stdout, stderr bytes.Buffer
	cmd.SetContext(context.Background())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := cmd.RunE(cmd, args)
	return queryCommandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func queryBalanceProvider() *queryCharacterizationProvider {
	return &queryCharacterizationProvider{
		chainID:   "chain-a-1",
		key:       "default",
		address:   "cosmos1alice",
		coins:     sdk.NewCoins(sdk.NewInt64Coin("uatom", 7)),
		keyExists: true,
	}
}

func queryProtocolProvider() *queryCharacterizationProvider {
	provider := queryBalanceProvider()
	provider.sprint = "rendered"
	provider.clientRes = &clienttypes.QueryClientStateResponse{}
	provider.connections = &conntypes.QueryConnectionsResponse{}
	provider.channelRes = &chantypes.QueryChannelResponse{}
	return provider
}

func queryPaginationProvider(t *testing.T, connections []string) *queryCharacterizationProvider {
	t.Helper()
	provider := queryProtocolProvider()
	provider.sprint = "channel-json"
	provider.channels = make([]*chantypes.IdentifiedChannel, 0, len(connections))
	provider.connectionByID = make(map[string]*conntypes.QueryConnectionResponse)
	provider.clientByID = make(map[string]*clienttypes.QueryClientStateResponse)
	for i, connectionID := range connections {
		clientID := "client-1"
		provider.channels = append(provider.channels, &chantypes.IdentifiedChannel{
			ChannelId:      fmt.Sprintf("channel-%d", i),
			ConnectionHops: []string{connectionID},
		})
		provider.connectionByID[connectionID] = &conntypes.QueryConnectionResponse{Connection: &conntypes.ConnectionEnd{
			ClientId: clientID,
			Counterparty: conntypes.Counterparty{
				ClientId:     "counter-" + clientID,
				ConnectionId: "counter-" + connectionID,
			},
		}}
		provider.clientByID[clientID] = &clienttypes.QueryClientStateResponse{ClientState: queryPackedClient(t, "chain-b-1")}
	}
	return provider
}

func queryPackedClient(t *testing.T, chainID string) *codectypes.Any {
	t.Helper()
	packed, err := codectypes.NewAnyWithValue(&tmclient.ClientState{
		ChainId:      chainID,
		LatestHeight: clienttypes.NewHeight(1, 50),
	})
	require.NoError(t, err)
	return packed
}

func sortedQueryCalls(provider *queryCharacterizationProvider) []string {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	calls := append([]string(nil), provider.queryCalls...)
	sort.Strings(calls)
	return calls
}

func queryCharacterizationState(p provider.ChainProvider) *appState {
	chains := make(relayer.Chains)
	if p != nil {
		chains["chain-a"] = relayer.NewChain(zap.NewNop(), p, false)
	}
	return &appState{viper: viper.New(), config: &Config{Chains: chains, Paths: make(relayer.Paths)}}
}

func queryUnrelayedState() (*appState, *queryCharacterizationProvider, *queryCharacterizationProvider) {
	srcProvider := queryProtocolProvider()
	srcProvider.chainID = "chain-a-1"
	srcProvider.height = 11
	srcProvider.channelsByConnection = map[string][]*chantypes.IdentifiedChannel{
		"connection-a": {{
			ChannelId: "channel-7",
			PortId:    "transfer",
			Ordering:  chantypes.UNORDERED,
			Counterparty: chantypes.Counterparty{
				ChannelId: "channel-8",
				PortId:    "transfer-counter",
			},
		}},
	}
	dstProvider := queryProtocolProvider()
	dstProvider.chainID = "chain-b-1"
	dstProvider.height = 22
	src := relayer.NewChain(zap.NewNop(), srcProvider, false)
	dst := relayer.NewChain(zap.NewNop(), dstProvider, false)
	path := &relayer.Path{
		Src: &relayer.PathEnd{ChainID: "chain-a-1", ClientID: "client-a", ConnectionID: "connection-a"},
		Dst: &relayer.PathEnd{ChainID: "chain-b-1", ClientID: "client-b", ConnectionID: "connection-b"},
	}
	state := &appState{
		viper: viper.New(),
		config: &Config{
			Chains: relayer.Chains{"src": src, "dst": dst},
			Paths:  relayer.Paths{"demo": path},
		},
	}
	return state, srcProvider, dstProvider
}

func packetCommitments(sequences ...uint64) *chantypes.QueryPacketCommitmentsResponse {
	states := make([]*chantypes.PacketState, 0, len(sequences))
	for _, sequence := range sequences {
		states = append(states, &chantypes.PacketState{Sequence: sequence})
	}
	return &chantypes.QueryPacketCommitmentsResponse{Commitments: states}
}

func queryExpirationState(t *testing.T) (*appState, *queryCharacterizationProvider, *queryCharacterizationProvider) {
	t.Helper()
	srcProvider := queryProtocolProvider()
	srcProvider.chainID = "chain-a-1"
	srcProvider.height = 101
	srcProvider.clientRes = queryExpirationClient(t, "chain-b-1", 50, 2*time.Hour, 3*time.Hour)
	srcProvider.blockTimes = map[int64]time.Time{60: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	dstProvider := queryProtocolProvider()
	dstProvider.chainID = "chain-b-1"
	dstProvider.height = 202
	dstProvider.clientRes = queryExpirationClient(t, "chain-a-1", 60, 4*time.Hour, 5*time.Hour)
	dstProvider.blockTimes = map[int64]time.Time{50: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}
	state := &appState{
		viper: viper.New(),
		config: &Config{
			Chains: relayer.Chains{
				"src": relayer.NewChain(zap.NewNop(), srcProvider, false),
				"dst": relayer.NewChain(zap.NewNop(), dstProvider, false),
			},
			Paths: relayer.Paths{"demo": {
				Src: &relayer.PathEnd{ChainID: "chain-a-1", ClientID: "client-a", ConnectionID: "connection-a"},
				Dst: &relayer.PathEnd{ChainID: "chain-b-1", ClientID: "client-b", ConnectionID: "connection-b"},
			}},
		},
	}
	return state, srcProvider, dstProvider
}

func queryExpirationClient(t *testing.T, chainID string, height uint64, trusting, unbonding time.Duration) *clienttypes.QueryClientStateResponse {
	t.Helper()
	packed, err := codectypes.NewAnyWithValue(&tmclient.ClientState{
		ChainId:         chainID,
		LatestHeight:    clienttypes.NewHeight(1, height),
		TrustingPeriod:  trusting,
		UnbondingPeriod: unbonding,
	})
	require.NoError(t, err)
	return &clienttypes.QueryClientStateResponse{ClientState: packed}
}

func expirationFields(t *testing.T, line string) map[string]string {
	t.Helper()
	fields := make(map[string]string)
	require.NoError(t, json.Unmarshal([]byte(line), &fields))
	return fields
}
