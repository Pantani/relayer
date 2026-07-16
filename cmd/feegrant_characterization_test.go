package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cometbytes "github.com/cometbft/cometbft/libs/bytes"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/relayer/v2/cclient"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCliFeegrantCharacterizesCobraDefaultsAndLookupOrder(t *testing.T) {
	t.Run("cobra structure and defaults", characterizeFeegrantCobraDefaults)
	t.Run("chain lookup precedes provider assertion", characterizeFeegrantLookupOrder)
	t.Run("mutually exclusive flags fail before command execution", characterizeFeegrantMutualExclusion)
}

func TestCliFeegrantCharacterizesConfigureGranterDeleteAndPartialEffects(t *testing.T) {
	t.Run("external granter requires explicit grantees", characterizeExternalGranterRequiresGrantees)
	t.Run("delete clears disk config and logs before returning", characterizeFeegrantDelete)
	t.Run("overwrite mutates captured provider before memo getter error", characterizeFeegrantOverwritePartialEffect)
}

func TestCliFeegrantCharacterizesConfigureBroadcastMessagesGasMemoAndHeight(t *testing.T) {
	state, prov, consensus, logs := characterizedFeegrantState(t, "chain")
	cmd := feegrantConfigureBasicCmd(state)
	require.NoError(t, cmd.Flags().Set("grantees", "grantee1,grantee2"))
	require.NoError(t, cmd.Flags().Set(flagMemo, "characterized memo"))
	require.NoError(t, cmd.Flags().Set(sdkflags.FlagGas, "456789"))

	stdout, stderr, err := executeCharacterizedFeegrantCommand(t, cmd, "chain")

	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Equal(t, 1, consensus.broadcasts)
	require.NotEmpty(t, consensus.broadcastTx)
	require.Equal(t, int64(91), prov.PCfg.FeeGrants.BlockHeightVerified)
	require.Equal(t, []string{"grantee1", "grantee2"}, prov.PCfg.FeeGrants.ManagedGrantees)
	disk := readCharacterizedDiskConfig(t, state.homePath)
	diskProvider := disk.ProviderConfigs["chain"].Value.(*cosmos.CosmosProviderConfig)
	require.Equal(t, int64(91), diskProvider.FeeGrants.BlockHeightVerified)
	require.Equal(t, []string{"grantee1", "grantee2"}, diskProvider.FeeGrants.ManagedGrantees)

	body, auth := decodeCharacterizedFeegrantTx(t, consensus.broadcastTx)
	require.Equal(t, "characterized memo", body.Memo)
	require.Len(t, body.Messages, 2)
	require.Equal(t, uint64(456789), auth.Fee.GasLimit)
	granterAddr := characterizedFeegrantKeyAddress(t, prov, "default")
	grantee1Addr := characterizedFeegrantKeyAddress(t, prov, "grantee1")
	grantee2Addr := characterizedFeegrantKeyAddress(t, prov, "grantee2")
	requireCharacterizedGrantMessage(t, body.Messages[0], granterAddr, grantee1Addr)
	requireCharacterizedGrantMessage(t, body.Messages[1], granterAddr, grantee2Addr)

	require.Equal(t, []string{
		"Creating feegrant",
		"Creating feegrant",
		"Feegrant succeeded",
		"feegrant configured",
	}, characterizedFeegrantCommandMessages(logs))
	created := logs.FilterMessage("Creating feegrant").All()
	require.Equal(t, granterAddr, created[0].ContextMap()["granter"])
	require.Equal(t, grantee1Addr, created[0].ContextMap()["grantee"])
	require.Equal(t, grantee2Addr, created[1].ContextMap()["grantee"])
	require.Equal(t, int64(91), logs.FilterMessage("feegrant configured").All()[0].ContextMap()["height"])
	require.Contains(t, consensus.queryPaths, "/cosmos.feegrant.v1beta1.Query/AllowancesByGranter")
	require.Contains(t, consensus.queryPaths, "/cosmos.auth.v1beta1.Query/Account")
	require.Contains(t, consensus.queryPaths, "/cosmos.tx.v1beta1.Service/GetTx")
}

func TestCliFeegrantCharacterizesBroadcastErrorWrappingAndPartialConfig(t *testing.T) {
	state, prov, consensus, logs := characterizedFeegrantState(t, "chain")
	consensus.broadcastErr = errors.New("broadcast failed")
	cmd := feegrantConfigureBasicCmd(state)
	require.NoError(t, cmd.Flags().Set("grantees", "grantee1,grantee2"))
	require.NoError(t, cmd.Flags().Set(sdkflags.FlagGas, "123"))

	stdout, stderr, err := executeCharacterizedFeegrantCommand(t, cmd, "chain")

	require.EqualError(t, err, "error writing grants on chain: 'broadcast failed'")
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Equal(t, 1, consensus.broadcasts)
	require.Equal(t, int64(0), prov.PCfg.FeeGrants.BlockHeightVerified)
	disk := readCharacterizedDiskConfig(t, state.homePath)
	diskFeegrants := disk.ProviderConfigs["chain"].Value.(*cosmos.CosmosProviderConfig).FeeGrants
	require.Equal(t, []string{"grantee1", "grantee2"}, diskFeegrants.ManagedGrantees)
	require.Equal(t, int64(0), diskFeegrants.BlockHeightVerified)
	require.Equal(t, []string{"Creating feegrant", "Creating feegrant"}, characterizedFeegrantCommandMessages(logs))
}

func TestCliFeegrantCharacterizesBasicQueryArgumentAndLogOrder(t *testing.T) {
	state, prov, consensus, logs := characterizedFeegrantState(t, "chain")
	chainAddr := characterizedFeegrantKeyAddress(t, prov, "chain")
	allowance, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{})
	require.NoError(t, err)
	consensus.grants = []*feegrant.Grant{
		{Granter: chainAddr, Grantee: "grantee-a", Allowance: allowance},
		{Granter: chainAddr, Grantee: "grantee-b", Allowance: allowance},
	}
	cmd := feegrantBasicGrantsCmd(state)

	stdout, stderr, err := executeCharacterizedFeegrantCommand(t, cmd, "chain", "ignored-granter")

	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Equal(t, chainAddr, consensus.lastGranterQuery)
	require.Equal(t, []string{"feegrant", "feegrant"}, characterizedLogMessages(logs))
	entries := logs.FilterMessage("feegrant").All()
	require.Equal(t, "grantee-a", entries[0].ContextMap()["grantee"])
	require.Equal(t, "grantee-b", entries[1].ContextMap()["grantee"])
	require.Equal(t, chainAddr, entries[0].ContextMap()["granter"])
	require.NotEmpty(t, entries[0].ContextMap()["allowance"])
	require.Equal(t, uint64(1), mustUint64Flag(t, cmd, flagPage))
	require.Equal(t, uint64(100), mustUint64Flag(t, cmd, flagLimit))
	require.False(t, mustBoolFlag(t, cmd, flagCountTotal))
	require.False(t, mustBoolFlag(t, cmd, flagReverse))
}

func characterizeFeegrantCobraDefaults(t *testing.T) {
	state, _, _, _ := characterizedFeegrantState(t, "chain")
	configure := feegrantConfigureBasicCmd(state)
	require.Equal(t, "basicallowance [chain-name] [granter] --num-grantees [int] --overwrite-granter --overwrite-grantees", configure.Use)
	require.Equal(t, 10, mustIntFlag(t, configure, "num-grantees"))
	require.False(t, mustBoolFlag(t, configure, "delete"))
	require.False(t, mustBoolFlag(t, configure, "overwrite-granter"))
	require.False(t, mustBoolFlag(t, configure, "overwrite-grantees"))
	require.Empty(t, mustStringSliceFlag(t, configure, "grantees"))
	require.Empty(t, mustStringFlag(t, configure, sdkflags.FlagGas))
	require.Empty(t, mustStringFlag(t, configure, flagMemo))
	require.NoError(t, configure.Args(configure, []string{"chain"}))
	require.NoError(t, configure.Args(configure, []string{"chain", "granter", "extra"}))
	require.Error(t, configure.Args(configure, nil))

	query := feegrantBasicGrantsCmd(state)
	require.Equal(t, "basic chain-name [granter]", query.Use)
	require.NoError(t, query.Args(query, []string{"chain"}))
	require.NoError(t, query.Args(query, []string{"chain", "granter"}))
	require.Error(t, query.Args(query, nil))
	require.Error(t, query.Args(query, []string{"chain", "granter", "extra"}))
}

func characterizeFeegrantLookupOrder(t *testing.T) {
	state := &appState{config: DefaultConfig(""), viper: viper.New(), log: zap.NewNop()}
	configure := feegrantConfigureBasicCmd(state)
	require.EqualError(t, configure.RunE(configure, []string{"missing"}),
		"chain with name \"missing\" not found in config. consider running `rly chains add missing`")

	nonCosmos := &pathValidationProvider{chainID: "penumbra-1"}
	state.config.Chains["other"] = relayer.NewChain(zap.NewNop(), nonCosmos, false)
	require.EqualError(t, configure.RunE(configure, []string{"other"}), "only CosmosProvider can be feegranted")

	query := feegrantBasicGrantsCmd(state)
	require.EqualError(t, query.RunE(query, []string{"missing"}),
		"chain with name \"missing\" not found in config. consider running `rly chains add missing`")
	require.EqualError(t, query.RunE(query, []string{"other"}), "only CosmosProvider can be feegranted")
}

func characterizeFeegrantMutualExclusion(t *testing.T) {
	state, _, consensus, _ := characterizedFeegrantState(t, "chain")
	cmd := feegrantConfigureBasicCmd(state)
	_, _, err := executeCharacterizedFeegrantCommand(t, cmd, "chain", "--delete", "--num-grantees", "2")
	require.ErrorContains(t, err, "if any flags in the group [num-grantees grantees delete] are set none of the others can be")
	require.Zero(t, consensus.broadcasts)
}

func characterizeExternalGranterRequiresGrantees(t *testing.T) {
	state, prov, consensus, _ := characterizedFeegrantState(t, "chain")
	external := characterizedExternalFeegrantAddress(t, prov)
	cmd := feegrantConfigureBasicCmd(state)
	stdout, stderr, err := executeCharacterizedFeegrantCommand(t, cmd, "chain", external)
	require.EqualError(t, err, "external granter "+external+" was specified, pre-authorized grantees must also be specified")
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Nil(t, prov.PCfg.FeeGrants)
	require.Zero(t, consensus.broadcasts)
}

func characterizeFeegrantDelete(t *testing.T) {
	state, prov, consensus, logs := characterizedFeegrantState(t, "chain")
	prov.PCfg.FeeGrants = characterizedExistingFeegrant("default")
	writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
	cmd := feegrantConfigureBasicCmd(state)
	stdout, stderr, err := executeCharacterizedFeegrantCommand(t, cmd, "chain", "--delete")
	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Zero(t, consensus.broadcasts)
	disk := readCharacterizedDiskConfig(t, state.homePath)
	require.Nil(t, disk.ProviderConfigs["chain"].Value.(*cosmos.CosmosProviderConfig).FeeGrants)
	require.Equal(t, []string{"Deleting feegrant configuration"}, characterizedFeegrantCommandMessages(logs))
}

func characterizeFeegrantOverwritePartialEffect(t *testing.T) {
	state, prov, consensus, _ := characterizedFeegrantState(t, "chain")
	prov.PCfg.FeeGrants = characterizedExistingFeegrant("default")
	writeCharacterizedRuntimeConfig(t, state.homePath, state.config)
	target := feegrantConfigureBasicCmd(state)
	require.NoError(t, target.Flags().Set("overwrite-granter", "true"))
	custom := &cobra.Command{Use: "basicallowance"}
	custom.SetContext(context.Background())
	err := target.RunE(custom, []string{"chain", "other"})
	require.EqualError(t, err, "flag accessed but not defined: memo")
	require.Equal(t, "other", prov.PCfg.FeeGrants.GranterKeyOrAddr)
	disk := readCharacterizedDiskConfig(t, state.homePath)
	require.Equal(t, "default", disk.ProviderConfigs["chain"].Value.(*cosmos.CosmosProviderConfig).FeeGrants.GranterKeyOrAddr)
	require.Zero(t, consensus.broadcasts)
}

func characterizedFeegrantState(
	t *testing.T,
	chainName string,
) (*appState, *cosmos.CosmosProvider, *characterizedFeegrantConsensus, *observer.ObservedLogs) {
	t.Helper()
	homeState := newCharacterizedConfigState(t)
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	homeState.log = logger
	homeState.viper = viper.New()
	homeState.config = DefaultConfig("")
	config := cosmos.CosmosProviderConfig{
		Key: "default", ChainID: chainName + "-1", AccountPrefix: "cosmos", KeyringBackend: "test",
		Timeout: "1s", GasPrices: "0uatom", GasAdjustment: 1.2, OutputFormat: "json",
		SignModeStr: "direct", Broadcast: provider.BroadcastModeBatch,
	}
	chainProvider, err := config.NewProvider(logger, homeState.homePath, false, chainName)
	require.NoError(t, err)
	prov := chainProvider.(*cosmos.CosmosProvider)
	prov.Keybase = keyring.NewInMemory(prov.Cdc.Marshaler, prov.KeyringOptions...)
	for _, name := range []string{"default", "other", chainName, "grantee1", "grantee2"} {
		_, err := prov.AddKey(name, sdk.CoinType, "secp256k1")
		require.NoError(t, err)
	}
	consensus := &characterizedFeegrantConsensus{height: 91}
	prov.ConsensusClient = consensus
	homeState.config.Chains[chainName] = relayer.NewChain(logger, prov, false)
	writeCharacterizedRuntimeConfig(t, homeState.homePath, homeState.config)
	return homeState, prov, consensus, logs
}

func executeCharacterizedFeegrantCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

func characterizedExistingFeegrant(granter string) *cosmos.FeeGrantConfiguration {
	return &cosmos.FeeGrantConfiguration{
		GranteesWanted: 1, GranterKeyOrAddr: granter, ManagedGrantees: []string{"grantee1"},
	}
}

func characterizedExternalFeegrantAddress(t *testing.T, prov *cosmos.CosmosProvider) string {
	t.Helper()
	address := sdk.AccAddress(bytes.Repeat([]byte{0x42}, 20))
	encoded, err := prov.EncodeBech32AccAddr(address)
	require.NoError(t, err)
	return encoded
}

func characterizedFeegrantKeyAddress(t *testing.T, prov *cosmos.CosmosProvider, name string) string {
	t.Helper()
	address, err := prov.GetKeyAddressForKey(name)
	require.NoError(t, err)
	encoded, err := prov.EncodeBech32AccAddr(address)
	require.NoError(t, err)
	return encoded
}

func decodeCharacterizedFeegrantTx(t *testing.T, txBytes []byte) (*txtypes.TxBody, *txtypes.AuthInfo) {
	t.Helper()
	var raw txtypes.TxRaw
	require.NoError(t, proto.Unmarshal(txBytes, &raw))
	var body txtypes.TxBody
	require.NoError(t, proto.Unmarshal(raw.BodyBytes, &body))
	var auth txtypes.AuthInfo
	require.NoError(t, proto.Unmarshal(raw.AuthInfoBytes, &auth))
	return &body, &auth
}

func requireCharacterizedGrantMessage(t *testing.T, any *codectypes.Any, granter, grantee string) {
	t.Helper()
	require.Equal(t, "/cosmos.feegrant.v1beta1.MsgGrantAllowance", any.TypeUrl)
	var msg feegrant.MsgGrantAllowance
	require.NoError(t, proto.Unmarshal(any.Value, &msg))
	require.Equal(t, granter, msg.Granter)
	require.Equal(t, grantee, msg.Grantee)
	require.Equal(t, "/cosmos.feegrant.v1beta1.BasicAllowance", msg.Allowance.TypeUrl)
}

func mustIntFlag(t *testing.T, cmd *cobra.Command, name string) int {
	t.Helper()
	value, err := cmd.Flags().GetInt(name)
	require.NoError(t, err)
	return value
}

func mustStringSliceFlag(t *testing.T, cmd *cobra.Command, name string) []string {
	t.Helper()
	value, err := cmd.Flags().GetStringSlice(name)
	require.NoError(t, err)
	return value
}

func characterizedFeegrantCommandMessages(logs *observer.ObservedLogs) []string {
	var messages []string
	for _, entry := range logs.All() {
		if entry.Message != "No backup RPCs defined" {
			messages = append(messages, entry.Message)
		}
	}
	return messages
}

type characterizedFeegrantConsensus struct {
	cclient.ConsensusClient
	height           uint64
	grants           []*feegrant.Grant
	queryPaths       []string
	lastGranterQuery string
	broadcasts       int
	broadcastTx      []byte
	broadcastErr     error
}

func (c *characterizedFeegrantConsensus) GetStatus(context.Context) (*cclient.Status, error) {
	return &cclient.Status{LatestBlockHeight: c.height}, nil
}

func (c *characterizedFeegrantConsensus) DoBroadcastTxAsync(_ context.Context, tx tmtypes.Tx) (*cclient.ResultBroadcastTx, error) {
	c.broadcasts++
	c.broadcastTx = append([]byte(nil), tx...)
	if c.broadcastErr != nil {
		return nil, c.broadcastErr
	}
	return &cclient.ResultBroadcastTx{Hash: cometbytes.HexBytes{0xab, 0xcd}}, nil
}

func (c *characterizedFeegrantConsensus) GetABCIQueryWithOptions(
	_ context.Context,
	path string,
	data cometbytes.HexBytes,
	_ rpcclient.ABCIQueryOptions,
) (*coretypes.ResultABCIQuery, error) {
	c.queryPaths = append(c.queryPaths, path)
	response, err := c.characterizedQueryResponse(path, data)
	if err != nil {
		return nil, err
	}
	encoded, err := proto.Marshal(response)
	if err != nil {
		return nil, err
	}
	return &coretypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: encoded, Height: int64(c.height)}}, nil
}

func (c *characterizedFeegrantConsensus) characterizedQueryResponse(path string, data []byte) (proto.Message, error) {
	switch path {
	case "/cosmos.feegrant.v1beta1.Query/AllowancesByGranter":
		var request feegrant.QueryAllowancesByGranterRequest
		if err := proto.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		c.lastGranterQuery = request.Granter
		return &feegrant.QueryAllowancesByGranterResponse{Allowances: c.grants, Pagination: &querytypes.PageResponse{}}, nil
	case "/cosmos.auth.v1beta1.Query/Account":
		var request authtypes.QueryAccountRequest
		if err := proto.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		account := &authtypes.BaseAccount{Address: request.Address, AccountNumber: 7, Sequence: 9}
		any, err := codectypes.NewAnyWithValue(account)
		if err != nil {
			return nil, err
		}
		return &authtypes.QueryAccountResponse{Account: any}, nil
	case "/cosmos.tx.v1beta1.Service/GetTx":
		return &txtypes.GetTxResponse{TxResponse: &sdk.TxResponse{TxHash: "ABCD", Code: 0}}, nil
	default:
		return nil, fmt.Errorf("unexpected ABCI query path %s", path)
	}
}
