package interchaintest

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/avast/retry-go/v4"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/cosmos/go-bip39"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/interchaintest/v11"
	cosmosv8 "github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
	"github.com/cosmos/interchaintest/v11/testreporter"
	"github.com/cosmos/interchaintest/v11/testutil"
	"github.com/cosmos/relayer/v2/cclient"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
)

const (
	feegrantPath            = "gaia-osmosis"
	feegrantTxQueryAttempts = uint(30)
	feegrantTxQueryDelay    = 500 * time.Millisecond
)

const (
	recvPacketMsgType      = "/ibc.core.channel.v1.MsgRecvPacket"
	acknowledgementMsgType = "/ibc.core.channel.v1.MsgAcknowledgement"
)

// protoTxProvider is a type which can provide a proto transaction. It is a
// workaround to get access to the wrapper TxBuilder's method GetProtoTx().
type protoTxProvider interface {
	GetProtoTx() *txtypes.Tx
}

type chainFeegrantInfo struct {
	granter  string
	grantees []string
}

type feegrantWalletSet struct {
	granter          ibc.Wallet
	grantees         []ibc.Wallet
	counterpartyUser ibc.Wallet
	gaiaUser         ibc.Wallet
}

type feegrantTestScenario struct {
	ctx              context.Context
	relayer          ibc.Relayer
	reporter         *testreporter.RelayerExecReporter
	gaia             ibc.Chain
	counterparty     ibc.Chain
	gaiaChannel      ibc.ChannelOutput
	counterpartyChan ibc.ChannelCounterparty
	wallets          feegrantWalletSet
	fundAmount       sdkmath.Int
	granteeFund      sdkmath.Int
	feegrantedChains map[string]*chainFeegrantInfo
}

type feegrantTransferResult struct {
	amount           sdkmath.Int
	gaiaDestination  string
	counterpartyDest string
}

type restoreKeyRequest struct {
	chain        ibc.Chain
	wallet       ibc.Wallet
	errorChainID string
}

func genMnemonic(t *testing.T) string {
	// read entropy seed straight from tmcrypto.Rand and convert to mnemonic
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		t.Fail()
	}

	mn, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		t.Fail()
	}

	return mn
}

// TestRelayerFeeGrant Feegrant on a single chain
// Run this test with e.g. go test -timeout 300s -run ^TestRelayerFeeGrant$ github.com/cosmos/relayer/v2/ibctest.
//
// Helpful to debug:
// docker ps -a --format {{.Names}} then e.g. docker logs gaia-1-val-0-TestRelayerFeeGrant 2>&1 -f
func TestRelayerFeeGrant(t *testing.T) {
	runFeegrantTestCases(t, zaptest.NewLogger(t), feegrantChainSpecs("v14.1.0"), false)
}

// TestRelayerFeeGrantExternal Feegrant on a single chain where the granter is an externally controlled address (no private key).
// Run this test with e.g. go test -timeout 300s -run ^TestRelayerFeeGrantExternal$ github.com/cosmos/relayer/v2/ibctest.
func TestRelayerFeeGrantExternal(t *testing.T) {
	runFeegrantTestCases(t, zaptest.NewLogger(t), feegrantChainSpecs("v7.0.3"), true)
}

func feegrantChainSpecs(gaiaVersion string) [][]*interchaintest.ChainSpec {
	nv := 1
	nf := 0

	return [][]*interchaintest.ChainSpec{
		{
			{Name: "gaia", ChainName: "gaia", Version: gaiaVersion, NumValidators: &nv, NumFullNodes: &nf, ChainConfig: gaiaChainConfig(gaiaVersion, ibc.ChainConfig{})},
			{Name: "osmosis", ChainName: "osmosis", Version: "v14.0.1", NumValidators: &nv, NumFullNodes: &nf},
		},
		{
			{Name: "gaia", ChainName: "gaia", Version: gaiaVersion, NumValidators: &nv, NumFullNodes: &nf, ChainConfig: gaiaChainConfig(gaiaVersion, ibc.ChainConfig{})},
			{Name: "kujira", ChainName: "kujira", Version: "v0.8.7", NumValidators: &nv, NumFullNodes: &nf},
		},
	}
}

func runFeegrantTestCases(
	t *testing.T,
	logger *zap.Logger,
	tests [][]*interchaintest.ChainSpec,
	externalGranter bool,
) {
	for _, specs := range tests {
		testName := fmt.Sprintf("%s,%s", specs[0].Name, specs[1].Name)
		t.Run(testName, func(t *testing.T) {
			runFeegrantScenario(t, logger, specs, externalGranter)
		})
	}
}

func runFeegrantScenario(
	t *testing.T,
	logger *zap.Logger,
	specs []*interchaintest.ChainSpec,
	externalGranter bool,
) {
	scenario := buildFeegrantScenario(t, context.Background(), specs)
	t.Parallel()

	if externalGranter {
		feegrant.RegisterInterfaces(scenario.gaia.Config().EncodingConfig.InterfaceRegistry)
	}

	loadFeegrantChannel(t, scenario)
	prepareFeegrantWallets(t, scenario, externalGranter)
	if externalGranter {
		grantExternalFeeAllowances(t, scenario)
	}

	fmt.Printf("Wallet mnemonic: %s\n", scenario.wallets.granter.Mnemonic())
	rand.Seed(time.Now().UnixNano())
	restoreFeegrantKeys(t, scenario, !externalGranter)
	configureFeegrant(t, logger, scenario, externalGranter)

	time.Sleep(14 * time.Second) // commit a couple blocks
	scenario.relayer.StartRelayer(scenario.ctx, scenario.reporter, feegrantPath)
	transferResult := executeFeegrantTransfers(t, scenario)
	signers := collectFeegrantSigners(t, scenario)
	validateFeegrantSigners(t, scenario, signers)
	validateFeegrantBalances(t, scenario, transferResult)
	scenario.relayer.StopRelayer(scenario.ctx, scenario.reporter)
}

func buildFeegrantScenario(
	t *testing.T,
	ctx context.Context,
	specs []*interchaintest.ChainSpec,
) *feegrantTestScenario {
	chainFactory := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), specs)
	chains, err := chainFactory.Chains(t.Name())
	require.NoError(t, err)
	gaia, counterparty := chains[0], chains[1]

	r := NewRelayerFactory(RelayerConfig{
		Processor:           relayer.ProcessorEvents,
		InitialBlockHistory: 100,
	}).Build(t, nil, "")
	processor.PathProcMessageCollector = make(chan *processor.PathProcessorMessageResp, 10000)

	ic := interchaintest.NewInterchain().
		AddChain(gaia).
		AddChain(counterparty).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  gaia,
			Chain2:  counterparty,
			Relayer: r,
			Path:    feegrantPath,
		})

	reporter := testreporter.NewNopReporter().RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)
	require.NoError(t, ic.Build(ctx, reporter, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))

	return &feegrantTestScenario{
		ctx:          ctx,
		relayer:      r,
		reporter:     reporter,
		gaia:         gaia,
		counterparty: counterparty,
		fundAmount:   sdkmath.NewInt(10_000_000),
	}
}

func loadFeegrantChannel(t *testing.T, scenario *feegrantTestScenario) {
	channels, err := scenario.relayer.GetChannels(
		scenario.ctx,
		scenario.reporter,
		scenario.gaia.Config().ChainID,
	)
	require.NoError(t, err)
	scenario.gaiaChannel = channels[0]
	scenario.counterpartyChan = channels[0].Counterparty
}

func prepareFeegrantWallets(t *testing.T, scenario *feegrantTestScenario, externalGranter bool) {
	if externalGranter {
		scenario.granteeFund = sdkmath.ZeroInt()
		scenario.wallets = createExternalFeegrantWallets(t, scenario)
		return
	}

	scenario.granteeFund = sdkmath.NewInt(10)
	scenario.wallets = createManagedFeegrantWallets(t, scenario)
}

func createManagedFeegrantWallets(t *testing.T, scenario *feegrantTestScenario) feegrantWalletSet {
	return feegrantWalletSet{
		granter: newFundedFeegrantWallet(t, scenario.ctx, "default", scenario.fundAmount, scenario.gaia),
		grantees: []ibc.Wallet{
			newFundedFeegrantWallet(t, scenario.ctx, "grantee1", scenario.granteeFund, scenario.gaia),
			newFundedFeegrantWallet(t, scenario.ctx, "grantee2", scenario.granteeFund, scenario.gaia),
			newFundedFeegrantWallet(t, scenario.ctx, "grantee3", scenario.granteeFund, scenario.gaia),
		},
		counterpartyUser: newFundedFeegrantWallet(t, scenario.ctx, "recipient", scenario.fundAmount, scenario.counterparty),
		gaiaUser:         newFundedFeegrantWallet(t, scenario.ctx, "recipient", scenario.fundAmount, scenario.gaia),
	}
}

func createExternalFeegrantWallets(t *testing.T, scenario *feegrantTestScenario) feegrantWalletSet {
	return feegrantWalletSet{
		grantees: []ibc.Wallet{
			newUnfundedFeegrantWallet(t, scenario.ctx, "grantee1", scenario.gaia),
			newUnfundedFeegrantWallet(t, scenario.ctx, "grantee2", scenario.gaia),
			newUnfundedFeegrantWallet(t, scenario.ctx, "grantee3", scenario.gaia),
		},
		counterpartyUser: newFundedFeegrantWallet(t, scenario.ctx, "recipient", scenario.fundAmount, scenario.counterparty),
		gaiaUser:         newFundedFeegrantWallet(t, scenario.ctx, "recipient", scenario.fundAmount, scenario.gaia),
		granter:          newFundedFeegrantWallet(t, scenario.ctx, "default", scenario.fundAmount, scenario.gaia),
	}
}

func newFundedFeegrantWallet(
	t *testing.T,
	ctx context.Context,
	keyPrefix string,
	amount sdkmath.Int,
	chain ibc.Chain,
) ibc.Wallet {
	wallet, err := interchaintest.GetAndFundTestUserWithMnemonic(ctx, keyPrefix, genMnemonic(t), amount, chain)
	require.NoError(t, err)
	return wallet
}

func newUnfundedFeegrantWallet(
	t *testing.T,
	ctx context.Context,
	keyPrefix string,
	chain ibc.Chain,
) ibc.Wallet {
	wallet, err := buildUserUnfunded(ctx, keyPrefix, genMnemonic(t), chain)
	require.NoError(t, err)
	return wallet
}

func grantExternalFeeAllowances(t *testing.T, scenario *feegrantTestScenario) {
	done := cosmos.SetSDKConfigContext(scenario.gaia.Config().Bech32Prefix)
	for _, grantee := range scenario.wallets.grantees {
		err := Feegrant(
			t,
			scenario.gaia.(*cosmosv8.CosmosChain),
			scenario.wallets.granter,
			scenario.wallets.granter.Address(),
			grantee.Address(),
			scenario.wallets.granter.FormattedAddress(),
			grantee.FormattedAddress(),
		)
		require.NoError(t, err)
	}
	done()
}

func restoreFeegrantKeys(t *testing.T, scenario *feegrantTestScenario, includeGranter bool) {
	requests := []restoreKeyRequest{
		{chain: scenario.gaia, wallet: scenario.wallets.grantees[0], errorChainID: scenario.gaia.Config().ChainID},
		{chain: scenario.gaia, wallet: scenario.wallets.grantees[1], errorChainID: scenario.gaia.Config().ChainID},
		{chain: scenario.gaia, wallet: scenario.wallets.grantees[2], errorChainID: scenario.gaia.Config().ChainID},
		{chain: scenario.counterparty, wallet: scenario.wallets.counterpartyUser, errorChainID: scenario.counterparty.Config().ChainID},
		{chain: scenario.counterparty, wallet: scenario.wallets.gaiaUser, errorChainID: scenario.gaia.Config().ChainID},
	}
	if includeGranter {
		granter := restoreKeyRequest{
			chain:        scenario.gaia,
			wallet:       scenario.wallets.granter,
			errorChainID: scenario.gaia.Config().ChainID,
		}
		requests = append([]restoreKeyRequest{granter}, requests...)
	}

	for _, request := range requests {
		restoreFeegrantKey(t, scenario, request)
	}
}

func restoreFeegrantKey(t *testing.T, scenario *feegrantTestScenario, request restoreKeyRequest) {
	err := scenario.relayer.RestoreKey(
		scenario.ctx,
		scenario.reporter,
		request.chain.Config(),
		request.wallet.KeyName(),
		request.wallet.Mnemonic(),
	)
	if err != nil {
		t.Fatalf("failed to restore granter key to relayer for chain %s: %s", request.errorChainID, err.Error())
	}
}

func configureFeegrant(
	t *testing.T,
	logger *zap.Logger,
	scenario *feegrantTestScenario,
	externalGranter bool,
) {
	granteeKeyNames := make([]string, 0, len(scenario.wallets.grantees))
	granteeAddresses := make([]string, 0, len(scenario.wallets.grantees))
	for _, grantee := range scenario.wallets.grantees {
		granteeKeyNames = append(granteeKeyNames, grantee.KeyName())
		granteeAddresses = append(granteeAddresses, grantee.FormattedAddress())
	}

	configuredGranter := scenario.wallets.granter.KeyName()
	if externalGranter {
		configuredGranter = scenario.wallets.granter.FormattedAddress()
	}

	localRelayer := scenario.relayer.(*Relayer)
	res := localRelayer.Sys().Run(
		logger,
		"chains", "configure", "feegrant", "basicallowance",
		scenario.gaia.Config().ChainID,
		configuredGranter,
		"--grantees", strings.Join(granteeKeyNames, ","),
		"--overwrite-granter",
	)
	if res.Err != nil {
		fmt.Printf("configure feegrant results: %s\n", res.Stdout.String())
		t.Fatalf("failed to rly config feegrants: %v", res.Err)
	}

	scenario.feegrantedChains = map[string]*chainFeegrantInfo{
		scenario.gaia.Config().ChainID: {
			granter:  scenario.wallets.granter.FormattedAddress(),
			grantees: granteeAddresses,
		},
	}
}

func executeFeegrantTransfers(t *testing.T, scenario *feegrantTestScenario) feegrantTransferResult {
	result := buildFeegrantTransferResult(scenario)
	gaiaHeight, err := scenario.gaia.Height(scenario.ctx)
	require.NoError(t, err)
	counterpartyHeight, err := scenario.counterparty.Height(scenario.ctx)
	require.NoError(t, err)

	var group errgroup.Group
	group.Go(func() error {
		return sendFeegrantTransfer(
			scenario,
			scenario.gaia,
			scenario.gaiaChannel.ChannelID,
			scenario.wallets.gaiaUser,
			result.gaiaDestination,
			result.amount,
			gaiaHeight,
		)
	})
	for range 3 {
		group.Go(func() error {
			return sendFeegrantTransfer(
				scenario,
				scenario.counterparty,
				scenario.counterpartyChan.ChannelID,
				scenario.wallets.counterpartyUser,
				result.counterpartyDest,
				result.amount,
				counterpartyHeight,
			)
		})
	}

	require.NoError(t, err)
	require.NoError(t, group.Wait())
	return result
}

func buildFeegrantTransferResult(scenario *feegrantTestScenario) feegrantTransferResult {
	gaiaDestination := types.MustBech32ifyAddressBytes(
		scenario.counterparty.Config().Bech32Prefix,
		scenario.wallets.gaiaUser.Address(),
	)
	counterpartyDestination := types.MustBech32ifyAddressBytes(
		scenario.gaia.Config().Bech32Prefix,
		scenario.wallets.counterpartyUser.Address(),
	)
	return feegrantTransferResult{
		amount:           sdkmath.NewInt(1_000),
		gaiaDestination:  gaiaDestination,
		counterpartyDest: counterpartyDestination,
	}
}

func sendFeegrantTransfer(
	scenario *feegrantTestScenario,
	chain ibc.Chain,
	channelID string,
	wallet ibc.Wallet,
	destination string,
	amount sdkmath.Int,
	startHeight int64,
) error {
	tx, err := chain.SendIBCTransfer(
		scenario.ctx,
		channelID,
		wallet.KeyName(),
		ibc.WalletAmount{Address: destination, Denom: chain.Config().Denom, Amount: amount},
		ibc.TransferOptions{},
	)
	if err != nil {
		return err
	}
	if err := tx.Validate(); err != nil {
		return err
	}

	_, err = testutil.PollForAck(scenario.ctx, chain, startHeight, startHeight+20, tx.Packet)
	return err
}

func collectFeegrantSigners(t *testing.T, scenario *feegrantTestScenario) map[string][]string {
	signers := map[string][]string{}
	for len(processor.PathProcMessageCollector) > 0 {
		select {
		case current, ok := <-processor.PathProcMessageCollector:
			if ok {
				collectFeegrantSigner(t, scenario, current, signers)
			}
		default:
			fmt.Println("Unknown channel message")
		}
	}
	return signers
}

func collectFeegrantSigner(
	t *testing.T,
	scenario *feegrantTestScenario,
	current *processor.PathProcessorMessageResp,
	signers map[string][]string,
) {
	if current.Error != nil || !current.SuccessfulTx {
		return
	}
	provider, ok := current.DestinationChain.(*cosmos.CosmosProvider)
	if !ok {
		return
	}

	chainID := provider.PCfg.ChainID
	feegrantInfo, feegrantedChain := scenario.feegrantedChains[chainID]
	if feegrantedChain && !strings.Contains(provider.PCfg.KeyDirectory, t.Name()) {
		fmt.Println("Skipping PathProcessorMessageResp from unrelated Parallel test case")
		return
	}

	done := provider.SetSDKContext()
	fullTx := decodeFeegrantTx(t, scenario.ctx, provider, current.Response.TxHash)
	feegrantedMessage, messageTypes := summarizeFeegrantMessages(fullTx)
	if feegrantedChain && feegrantedMessage {
		recordFeegrantSigner(t, provider, current, fullTx, feegrantInfo, chainID, messageTypes, signers)
	}
	done()
}

func decodeFeegrantTx(
	t *testing.T,
	ctx context.Context,
	provider *cosmos.CosmosProvider,
	txHash string,
) *txtypes.Tx {
	hash, err := hex.DecodeString(txHash)
	require.Nil(t, err)
	txResponse, err := TxWithRetry(ctx, provider.ConsensusClient, hash)
	require.NoError(t, err)

	decoder := provider.Cdc.TxConfig.TxDecoder()
	tx, err := decoder(txResponse.Tx)
	require.Nil(t, err)
	builder, err := provider.Cdc.TxConfig.WrapTxBuilder(tx)
	require.Nil(t, err)
	return builder.(protoTxProvider).GetProtoTx()
}

func summarizeFeegrantMessages(tx *txtypes.Tx) (bool, string) {
	feegrantedMessage := false
	messageTypes := ""
	for _, message := range tx.GetMsgs() {
		messageType := types.MsgTypeURL(message)
		if messageType == recvPacketMsgType || messageType == acknowledgementMsgType {
			feegrantedMessage = true
		}
		messageTypes += messageType + ", "
	}
	return feegrantedMessage, messageTypes
}

func recordFeegrantSigner(
	t *testing.T,
	provider *cosmos.CosmosProvider,
	current *processor.PathProcessorMessageResp,
	tx *txtypes.Tx,
	feegrantInfo *chainFeegrantInfo,
	chainID string,
	messageTypes string,
	signersByChain map[string][]string,
) {
	fmt.Printf("Msg types: %+v\n", messageTypes)
	signers, _, err := provider.Cdc.Marshaler.GetMsgV1Signers(tx)
	require.NoError(t, err)
	require.Equal(t, len(signers), 1)

	granter := tx.FeeGranter(provider.Cdc.Marshaler)
	require.Equal(t, feegrantInfo.granter, string(granter))
	require.NotEmpty(t, granter)
	lastMessageType := logFeegrantPacketData(tx)

	actualGrantee := string(signers[0])
	signersByChain[chainID] = append(signersByChain[chainID], actualGrantee)
	fmt.Printf(
		"Chain: %s, msg type: %s, height: %d, signer: %s, granter: %s\n",
		chainID,
		lastMessageType,
		current.Response.Height,
		actualGrantee,
		string(granter),
	)
}

func logFeegrantPacketData(tx *txtypes.Tx) string {
	lastMessageType := ""
	for _, message := range tx.GetMsgs() {
		lastMessageType = types.MsgTypeURL(message)
		if lastMessageType != recvPacketMsgType {
			continue
		}

		receivePacket := message.(*chantypes.MsgRecvPacket)
		appData := receivePacket.Packet.GetData()
		tokenTransfer := &transfertypes.FungibleTokenPacketData{}
		if err := tokenTransfer.Unmarshal(appData); err == nil {
			fmt.Printf("%+v\n", tokenTransfer)
		} else {
			fmt.Println(string(appData))
		}
	}
	return lastMessageType
}

func validateFeegrantSigners(
	t *testing.T,
	scenario *feegrantTestScenario,
	signersByChain map[string][]string,
) {
	for chainID, signers := range signersByChain {
		require.Equal(t, chainID, scenario.gaia.Config().ChainID)
		counts := countFeegrantSigners(signers)
		highestCount := highestFeegrantSignerCount(counts)

		require.GreaterOrEqual(t, highestCount, 1)
		expectedFeegrantInfo := scenario.feegrantedChains[chainID]
		require.Equal(t, len(counts), len(expectedFeegrantInfo.grantees))
		assertRoundRobinFeegrantSigners(t, counts, highestCount)
	}
}

func countFeegrantSigners(signers []string) map[string]int {
	counts := make(map[string]int, len(signers))
	for _, signer := range signers {
		counts[signer]++
	}
	return counts
}

func highestFeegrantSignerCount(counts map[string]int) int {
	highestCount := 0
	for _, count := range counts {
		if count > highestCount {
			highestCount = count
		}
	}
	return highestCount
}

func assertRoundRobinFeegrantSigners(t *testing.T, counts map[string]int, highestCount int) {
	for signer, count := range counts {
		fmt.Printf("signer %s signed %d feegranted TXs \n", signer, count)
		require.LessOrEqual(t, highestCount-count, 1)
	}
}

func validateFeegrantBalances(
	t *testing.T,
	scenario *feegrantTestScenario,
	result feegrantTransferResult,
) {
	gaiaDenomTrace := transfertypes.ExtractDenomFromPath(transfertypes.GetPrefixedDenom(
		scenario.counterpartyChan.PortID,
		scenario.counterpartyChan.ChannelID,
		scenario.gaia.Config().Denom,
	))
	gaiaIBCDenom := gaiaDenomTrace.IBCDenom()
	counterpartyDenomTrace := transfertypes.ExtractDenomFromPath(transfertypes.GetPrefixedDenom(
		scenario.gaiaChannel.PortID,
		scenario.gaiaChannel.ChannelID,
		scenario.counterparty.Config().Denom,
	))
	counterpartyIBCDenom := counterpartyDenomTrace.IBCDenom()

	gaiaIBCBalance, err := scenario.counterparty.GetBalance(
		scenario.ctx,
		result.gaiaDestination,
		gaiaIBCDenom,
	)
	require.NoError(t, err)
	require.True(t, result.amount.Equal(gaiaIBCBalance))

	counterpartyIBCBalance, err := scenario.gaia.GetBalance(
		scenario.ctx,
		result.counterpartyDest,
		counterpartyIBCDenom,
	)
	require.NoError(t, err)
	require.True(t, result.amount.MulRaw(3).Equal(counterpartyIBCBalance))

	granteeBalance, err := scenario.gaia.GetBalance(
		scenario.ctx,
		scenario.wallets.grantees[0].FormattedAddress(),
		scenario.gaia.Config().Denom,
	)
	require.NoError(t, err)
	require.True(t, granteeBalance.Equal(scenario.granteeFund))

	granterBalance, err := scenario.gaia.GetBalance(
		scenario.ctx,
		scenario.wallets.granter.FormattedAddress(),
		scenario.gaia.Config().Denom,
	)
	require.NoError(t, err)
	require.True(t, granterBalance.LT(scenario.fundAmount))
}

func TxWithRetry(ctx context.Context, client cclient.ConsensusClient, hash []byte) (*coretypes.ResultTx, error) {
	var err error
	var res *coretypes.ResultTx
	if err = retry.Do(func() error {
		res, err = client.GetTx(ctx, hash, true)
		return err
	},
		retry.Context(ctx),
		retry.Attempts(feegrantTxQueryAttempts),
		retry.Delay(feegrantTxQueryDelay),
		retry.DelayType(retry.FixedDelay),
		relayer.RtyErr,
	); err != nil {
		return res, err
	}

	return res, err
}

func buildUserUnfunded(
	ctx context.Context,
	keyNamePrefix, mnemonic string,
	chain ibc.Chain,
) (ibc.Wallet, error) {
	chainCfg := chain.Config()
	keyName := fmt.Sprintf("%s-%s-%s", keyNamePrefix, chainCfg.ChainID, randLowerCaseLetterString(3))
	user, err := chain.BuildWallet(ctx, keyName, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to get source user wallet: %w", err)
	}

	return user, nil
}

var chars = []byte("abcdefghijklmnopqrstuvwxyz")

// RandLowerCaseLetterString returns a lowercase letter string of given length
func randLowerCaseLetterString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func Feegrant(
	t *testing.T,
	chain *cosmosv8.CosmosChain,
	granterWallet ibc.Wallet,
	granter types.AccAddress,
	grantee types.AccAddress,
	granterAddr string,
	granteeAddr string,
) error {
	// attempt to update client with duplicate header
	b := cosmosv8.NewBroadcaster(t, chain)

	thirtyMin := time.Now().Add(30 * time.Minute)
	feeGrantBasic := &feegrant.BasicAllowance{
		Expiration: &thirtyMin,
	}
	msgGrantAllowance, err := feegrant.NewMsgGrantAllowance(feeGrantBasic, granter, grantee)
	if err != nil {
		fmt.Printf("Error: feegrant.NewMsgGrantAllowance: %s", err.Error())
		return err
	}

	// ensure correct bech32 prefix
	msgGrantAllowance.Grantee = granteeAddr
	msgGrantAllowance.Granter = granterAddr

	resp, err := cosmosv8.BroadcastTx(context.Background(), b, granterWallet, msgGrantAllowance)
	require.NoError(t, err)
	assertTransactionIsValid(t, resp)
	return nil
}

func assertTransactionIsValid(t *testing.T, resp types.TxResponse) {
	t.Helper()
	require.NotNil(t, resp)
	require.NotEqual(t, 0, resp.GasUsed)
	require.NotEqual(t, 0, resp.GasWanted)
	require.Equal(t, uint32(0), resp.Code)
	require.NotEmpty(t, resp.Data)
	require.NotEmpty(t, resp.TxHash)
	require.NotEmpty(t, resp.Events)
}
