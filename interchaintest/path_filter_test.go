package interchaintest_test

import (
	"context"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	"github.com/cosmos/interchaintest/v11"
	"github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
	"github.com/cosmos/interchaintest/v11/testreporter"
	"github.com/cosmos/interchaintest/v11/testutil"
	relayerinterchaintest "github.com/cosmos/relayer/v2/interchaintest"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
)

// TestScenarioPathFilterAllow tests the channel allowlist
func TestScenarioPathFilterAllow(t *testing.T) {
	ctx := context.Background()

	nv := 1
	nf := 0

	// Chain Factory
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{Name: "gaia", Version: "v14.1.0", NumValidators: &nv, NumFullNodes: &nf, ChainConfig: gaiaChainConfig("v14.1.0", ibc.ChainConfig{})},
		{Name: "osmosis", Version: "v22.0.0", NumValidators: &nv, NumFullNodes: &nf},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	gaia, osmosis := chains[0], chains[1]

	// Relayer Factory to construct relayer
	r := relayerinterchaintest.NewRelayerFactory(relayerinterchaintest.RelayerConfig{
		Processor:           relayer.ProcessorEvents,
		InitialBlockHistory: 100,
	}).Build(t, nil, "")

	t.Parallel()

	// Prep Interchain
	const ibcPath = "gaia-osmosis"
	ic := interchaintest.NewInterchain().
		AddChain(gaia).
		AddChain(osmosis).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  gaia,
			Chain2:  osmosis,
			Relayer: r,
			Path:    ibcPath,
		})

	// Reporter/logs
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	client, network := interchaintest.DockerSetup(t)

	// Build interchain
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,

		SkipPathCreation: false,
	}))

	// Get Channel ID
	gaiaChans, err := r.GetChannels(ctx, eRep, gaia.Config().ChainID)
	require.NoError(t, err)
	gaiaChannel := gaiaChans[0]
	osmosisChannel := gaiaChans[0].Counterparty

	require.NoError(t, r.UpdatePath(ctx, eRep, ibcPath, ibc.PathUpdateOptions{
		ChannelFilter: &ibc.ChannelFilter{
			Rule:        processor.RuleAllowList,
			ChannelList: []string{gaiaChannel.ChannelID},
		},
	}))

	// Create and Fund User Wallets
	initBal := sdkmath.NewInt(10_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", initBal, gaia, osmosis)

	gaiaUser, osmosisUser := users[0].(*cosmos.CosmosWallet), users[1].(*cosmos.CosmosWallet)

	require.NoError(t, r.StartRelayer(ctx, eRep, ibcPath))
	t.Cleanup(func() { stopRelayerForTest(t, ctx, r, eRep) })

	// Send Transaction
	amountToSend := sdkmath.NewInt(1_000_000)
	gaiaDstAddress := gaiaUser.FormattedAddressWithPrefix(osmosis.Config().Bech32Prefix)
	osmosisDstAddress := osmosisUser.FormattedAddressWithPrefix(gaia.Config().Bech32Prefix)

	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)

	osmosisHeight, err := osmosis.Height(ctx)
	require.NoError(t, err)

	var eg errgroup.Group
	eg.Go(func() error {
		return sendPathFilterTransfer(ctx, gaia, gaiaChannel.ChannelID, gaiaUser, gaiaDstAddress, amountToSend, gaiaHeight, true)
	})

	eg.Go(func() error {
		return sendPathFilterTransfer(ctx, osmosis, osmosisChannel.ChannelID, osmosisUser, osmosisDstAddress, amountToSend, osmosisHeight, true)
	})
	// Acks should exist
	require.NoError(t, eg.Wait())

	// Trace IBC Denom
	gaiaDenomTrace := transfertypes.ExtractDenomFromPath(transfertypes.GetPrefixedDenom(osmosisChannel.PortID, osmosisChannel.ChannelID, gaia.Config().Denom))
	gaiaIbcDenom := gaiaDenomTrace.IBCDenom()

	osmosisDenomTrace := transfertypes.ExtractDenomFromPath(transfertypes.GetPrefixedDenom(gaiaChannel.PortID, gaiaChannel.ChannelID, osmosis.Config().Denom))
	osmosisIbcDenom := osmosisDenomTrace.IBCDenom()

	// Test destination wallets have increased funds
	gaiaIBCBalance, err := osmosis.GetBalance(ctx, gaiaDstAddress, gaiaIbcDenom)
	require.NoError(t, err)
	require.True(t, amountToSend.Equal(gaiaIBCBalance))

	osmosisIBCBalance, err := gaia.GetBalance(ctx, osmosisDstAddress, osmosisIbcDenom)
	require.NoError(t, err)
	require.True(t, amountToSend.Equal(osmosisIBCBalance))
}

// TestScenarioPathFilterDeny tests the channel denylist
func TestScenarioPathFilterDeny(t *testing.T) {
	ctx := context.Background()

	nv := 1
	nf := 0

	// Chain Factory
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{Name: "gaia", Version: "v14.1.0", NumValidators: &nv, NumFullNodes: &nf, ChainConfig: gaiaChainConfig("v14.1.0", ibc.ChainConfig{})},
		{Name: "osmosis", Version: "v22.0.0", NumValidators: &nv, NumFullNodes: &nf},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	gaia, osmosis := chains[0], chains[1]

	// Relayer Factory to construct relayer
	r := relayerinterchaintest.NewRelayerFactory(relayerinterchaintest.RelayerConfig{
		Processor:           relayer.ProcessorEvents,
		InitialBlockHistory: 100,
	}).Build(t, nil, "")

	// Prep Interchain
	const ibcPath = "gaia-osmosis"
	ic := interchaintest.NewInterchain().
		AddChain(gaia).
		AddChain(osmosis).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  gaia,
			Chain2:  osmosis,
			Relayer: r,
			Path:    ibcPath,
		})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	client, network := interchaintest.DockerSetup(t)

	// Build interchain
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,

		SkipPathCreation: false,
	}))

	t.Parallel()

	// Get Channel ID
	gaiaChans, err := r.GetChannels(ctx, eRep, gaia.Config().ChainID)
	require.NoError(t, err)
	gaiaChannel := gaiaChans[0]
	osmosisChannel := gaiaChans[0].Counterparty

	require.NoError(t, r.UpdatePath(ctx, eRep, ibcPath, ibc.PathUpdateOptions{
		ChannelFilter: &ibc.ChannelFilter{
			Rule:        processor.RuleDenyList,
			ChannelList: []string{gaiaChannel.ChannelID},
		},
	}))

	// Create and Fund User Wallets
	initBal := sdkmath.NewInt(10_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", initBal, gaia, osmosis)

	gaiaUser, osmosisUser := users[0].(*cosmos.CosmosWallet), users[1].(*cosmos.CosmosWallet)

	require.NoError(t, r.StartRelayer(ctx, eRep, ibcPath))
	t.Cleanup(func() { stopRelayerForTest(t, ctx, r, eRep) })

	// Send Transaction
	amountToSend := sdkmath.NewInt(1_000_000)
	gaiaDstAddress := gaiaUser.FormattedAddressWithPrefix(osmosis.Config().Bech32Prefix)
	osmosisDstAddress := osmosisUser.FormattedAddressWithPrefix(gaia.Config().Bech32Prefix)

	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)

	osmosisHeight, err := osmosis.Height(ctx)
	require.NoError(t, err)

	var eg errgroup.Group
	eg.Go(func() error {
		return sendPathFilterTransfer(ctx, gaia, gaiaChannel.ChannelID, gaiaUser, gaiaDstAddress, amountToSend, gaiaHeight, false)
	})

	eg.Go(func() error {
		return sendPathFilterTransfer(ctx, osmosis, osmosisChannel.ChannelID, osmosisUser, osmosisDstAddress, amountToSend, osmosisHeight, false)
	})
	// Test that acks do not show up
	require.NoError(t, eg.Wait())

	// Trace IBC Denom
	gaiaDenomTrace := transfertypes.ExtractDenomFromPath(transfertypes.GetPrefixedDenom(osmosisChannel.PortID, osmosisChannel.ChannelID, gaia.Config().Denom))
	gaiaIbcDenom := gaiaDenomTrace.IBCDenom()

	osmosisDenomTrace := transfertypes.ExtractDenomFromPath(transfertypes.GetPrefixedDenom(gaiaChannel.PortID, gaiaChannel.ChannelID, osmosis.Config().Denom))
	osmosisIbcDenom := osmosisDenomTrace.IBCDenom()

	// Test destination wallets do not have increased funds
	gaiaIBCBalance, err := osmosis.GetBalance(ctx, gaiaDstAddress, gaiaIbcDenom)
	require.NoError(t, err)
	require.True(t, sdkmath.ZeroInt().Equal(gaiaIBCBalance))

	osmosisIBCBalance, err := gaia.GetBalance(ctx, osmosisDstAddress, osmosisIbcDenom)
	require.NoError(t, err)
	require.True(t, sdkmath.ZeroInt().Equal(osmosisIBCBalance))
}

func stopRelayerForTest(t *testing.T, ctx context.Context, r ibc.Relayer, rep ibc.RelayerExecReporter) {
	t.Helper()
	if err := r.StopRelayer(ctx, rep); err != nil {
		t.Logf("an error occurred while stopping the relayer: %s", err)
	}
}

func sendPathFilterTransfer(
	ctx context.Context,
	chain ibc.Chain,
	channelID string,
	user ibc.Wallet,
	destination string,
	amount sdkmath.Int,
	startHeight int64,
	expectAck bool,
) error {
	tx, err := chain.SendIBCTransfer(ctx, channelID, user.KeyName(), ibc.WalletAmount{
		Address: destination,
		Denom:   chain.Config().Denom,
		Amount:  amount,
	}, ibc.TransferOptions{})
	if err != nil {
		return err
	}
	if err := tx.Validate(); err != nil {
		return err
	}

	ack, pollErr := testutil.PollForAck(ctx, chain, startHeight, startHeight+10, tx.Packet)
	if expectAck {
		return pollErr
	}
	if pollErr == nil {
		return fmt.Errorf("no error when error was expected when polling for ack: %+v", ack)
	}
	return nil
}
