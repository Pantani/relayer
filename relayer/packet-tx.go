package relayer

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
)

const defaultTimeoutOffset = 1000

// SendTransferMsg initiates an ics20 transfer from src to dst with the specified args.
func (c *Chain) SendTransferMsg(
	ctx context.Context,
	log *zap.Logger,
	dst *Chain,
	amount sdk.Coin,
	dstAddr, memo string,
	toHeightOffset uint64,
	toTimeOffset time.Duration,
	srcChannel *chantypes.IdentifiedChannel,
) error {
	if err := validateTransferTimeoutOffset(toTimeOffset); err != nil {
		return err
	}

	// get header representing dst to check timeouts
	srch, dsth, err := QueryLatestHeights(ctx, c, dst)
	if err != nil {
		return err
	}
	clientLatestHeight, err := transferClientLatestHeight(ctx, c, srch)
	if err != nil {
		return err
	}

	timeoutTimestamp, err := transferTimeoutTimestamp(ctx, dst, dsth, toTimeOffset)
	if err != nil {
		return err
	}

	timeoutHeight := transferTimeoutHeight(clientLatestHeight.GetRevisionHeight(), toHeightOffset, toTimeOffset)

	// MsgTransfer will call SendPacket on src chain
	pi := provider.PacketInfo{
		SourceChannel: srcChannel.ChannelId,
		SourcePort:    srcChannel.PortId,
		TimeoutHeight: clienttypes.Height{
			RevisionNumber: clientLatestHeight.GetRevisionNumber(),
			RevisionHeight: timeoutHeight,
		},
		TimeoutTimestamp: timeoutTimestamp,
	}

	msg, err := c.ChainProvider.MsgTransfer(dstAddr, amount, pi)
	if err != nil {
		return err
	}

	txs := RelayMsgs{
		Src: []provider.RelayerMessage{msg},
	}

	result := txs.Send(ctx, log, AsRelayMsgSender(c), AsRelayMsgSender(dst), memo)
	return logTransferResult(c, dst, result)
}

func validateTransferTimeoutOffset(offset time.Duration) error {
	if offset < 0 {
		return fmt.Errorf("transfer timeout time offset cannot be negative: %s", offset)
	}
	return nil
}

func transferTimeoutTimestamp(ctx context.Context, dst *Chain, queryHeight int64, offset time.Duration) (uint64, error) {
	if offset <= 0 {
		return 0, nil
	}
	referenceTimestamp, err := transferReferenceTimestamp(ctx, dst, queryHeight)
	if err != nil {
		return 0, err
	}
	return max(uint64(time.Now().UnixNano()), referenceTimestamp) + uint64(offset), nil
}

func transferTimeoutHeight(clientHeight, heightOffset uint64, timeOffset time.Duration) uint64 {
	if heightOffset > 0 {
		return clientHeight + heightOffset
	}
	if timeOffset > 0 {
		return 0
	}
	return clientHeight + defaultTimeoutOffset
}

func logTransferResult(src, dst *Chain, result SendMsgsResult) error {
	if err := result.Error(); err != nil {
		if result.PartiallySent() {
			src.log.Info(
				"Partial success when sending transfer",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.Object("send_result", result),
			)
		}
		return err
	}
	if result.SuccessfullySent() {
		src.log.Info(
			"Successfully sent a transfer",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.Object("send_result", result),
		)
	}

	return nil
}

func transferClientLatestHeight(ctx context.Context, chain *Chain, queryHeight int64) (clienttypes.Height, error) {
	if chain.ClientID() == ibcexported.LocalhostClientID {
		if queryHeight < 0 {
			return clienttypes.Height{}, fmt.Errorf("localhost query height cannot be negative: %d", queryHeight)
		}
		return clienttypes.NewHeight(clienttypes.ParseChainID(chain.ChainID()), uint64(queryHeight)), nil
	}

	state, err := chain.ChainProvider.QueryClientState(ctx, queryHeight, chain.ClientID())
	if err != nil {
		return clienttypes.Height{}, err
	}
	return provider.ClientStateLatestHeight(state)
}

func transferReferenceTimestamp(ctx context.Context, chain *Chain, queryHeight int64) (uint64, error) {
	if chain.ClientID() == ibcexported.LocalhostClientID {
		blockTime, err := chain.ChainProvider.BlockTime(ctx, queryHeight)
		if err != nil {
			return 0, fmt.Errorf("failed to query localhost block time: %w", err)
		}
		return uint64(blockTime.UnixNano()), nil
	}

	stateResponse, err := chain.ChainProvider.QueryClientStateResponse(ctx, queryHeight, chain.ClientID())
	if err != nil {
		return 0, fmt.Errorf("failed to query the client state response: %w", err)
	}
	state, err := clienttypes.UnpackClientState(stateResponse.ClientState)
	if err != nil {
		return 0, fmt.Errorf("failed to unpack client state: %w", err)
	}
	stateHeight, err := provider.ClientStateLatestHeight(state)
	if err != nil {
		return 0, fmt.Errorf("failed to extract client state height: %w", err)
	}
	consensusResponse, err := chain.ChainProvider.QueryClientConsensusState(ctx, queryHeight, chain.ClientID(), stateHeight)
	if err != nil {
		return 0, fmt.Errorf("failed to query client consensus state: %w", err)
	}
	consensusState, err := clienttypes.UnpackConsensusState(consensusResponse.ConsensusState)
	if err != nil {
		return 0, fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	return consensusState.GetTimestamp(), nil
}
