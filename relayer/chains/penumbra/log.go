package penumbra

import (
	"reflect"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typestx "github.com/cosmos/cosmos-sdk/types/tx"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	feetypes "github.com/cosmos/relayer/v2/relayer/codecs/ics29"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// getChannelsIfPresent scans the events for channel tags
func getChannelsIfPresent(events []provider.RelayerEvent) []zapcore.Field {
	channelTags := []string{srcChanTag, dstChanTag}
	fields := []zap.Field{}

	// While a transaction may have multiple messages, we just need to first
	// pair of channels
	foundTag := map[string]struct{}{}

	for _, event := range events {
		for _, tag := range channelTags {
			for attributeKey, attributeValue := range event.Attributes {
				if attributeKey == tag {
					// Only append the tag once
					// TODO: what if they are different?
					if _, ok := foundTag[tag]; !ok {
						fields = append(fields, zap.String(tag, attributeValue))
						foundTag[tag] = struct{}{}
					}
				}
			}
		}
	}
	return fields
}

// LogFailedTx takes the transaction and the messages to create it and logs the appropriate data
func (cc *PenumbraProvider) LogFailedTx(res *provider.RelayerTxResponse, err error, msgs []provider.RelayerMessage) {
	// Include the chain_id
	fields := []zapcore.Field{zap.String("chain_id", cc.ChainId())}

	// Extract the channels from the events, if present
	if res != nil {
		channels := getChannelsIfPresent(res.Events)
		fields = append(fields, channels...)
	}
	fields = append(fields, msgTypesField(msgs))

	if err != nil {
		// Make a copy since we may continue to the warning
		errorFields := append(fields, zap.Error(err))
		cc.log.Error(
			"Failed sending cosmos transaction",
			errorFields...,
		)

		if res == nil {
			return
		}
	}

	if res.Code != 0 && res.Data != "" {
		fields = append(fields, zap.Object("response", res))
		cc.log.Warn(
			"Sent transaction but received failure response",
			fields...,
		)
	}
}

// LogSuccessTx take the transaction and the messages to create it and logs the appropriate data
func (cc *PenumbraProvider) LogSuccessTx(res *sdk.TxResponse, msgs []provider.RelayerMessage) {
	if res == nil {
		cc.log.Debug("Cannot log successful transaction without a response")
		return
	}

	// Include the chain_id
	fields := []zapcore.Field{zap.String("chain_id", cc.ChainId())}

	// Extract the channels from the events, if present.
	events := parseEventsFromTxResponse(res)
	fields = append(fields, getChannelsIfPresent(events)...)

	// Include the gas used
	fields = append(fields, zap.Int64("gas_used", res.GasUsed))

	// Extract fees and fee_payer if present
	ir := cc.Codec.InterfaceRegistry
	cdc := codec.NewProtoCodec(ir)

	var m sdk.Msg
	if err := ir.UnpackAny(res.Tx, &m); err == nil {
		if tx, ok := m.(*typestx.Tx); ok {
			fields = append(fields, zap.Stringer("fees", tx.GetFee()))
			if feePayer := getFeePayer(cc.log, cdc, tx); feePayer != "" {
				fields = append(fields, zap.String("fee_payer", feePayer))
			}
		} else {
			cc.log.Debug(
				"Failed to convert message to Tx type",
				zap.Stringer("type", reflect.TypeOf(m)),
			)
		}
	} else {
		cc.log.Debug("Failed to unpack response Tx into sdk.Msg", zap.Error(err))
	}

	// Include the height, msgType, and tx_hash
	fields = append(fields,
		zap.Int64("height", res.Height),
		msgTypesField(msgs),
		zap.String("tx_hash", res.TxHash),
	)

	// Log the successful transaction with fields
	cc.log.Info(
		"Successful transaction",
		fields...,
	)
}

func msgTypesField(msgs []provider.RelayerMessage) zap.Field {
	msgTypes := make([]string, len(msgs))
	for i, m := range msgs {
		msgTypes[i] = m.Type()
	}
	return zap.Strings("msg_types", msgTypes)
}

// getFeePayer returns the bech32 address of the fee payer of a transaction.
// This uses the fee payer field if set,
// otherwise falls back to the address of whoever signed the first message.
func getFeePayer(log *zap.Logger, cdc *codec.ProtoCodec, tx *typestx.Tx) string {
	if payer := explicitFeePayer(tx); payer != "" {
		return payer
	}
	firstMsg, ok := firstTxMessage(log, tx)
	if !ok {
		return ""
	}
	if payer, known := knownMessageFeePayer(firstMsg); known {
		return payer
	}
	return derivedMessageFeePayer(log, cdc, firstMsg)
}

func explicitFeePayer(tx *typestx.Tx) string {
	if tx == nil || tx.AuthInfo == nil || tx.AuthInfo.Fee == nil {
		return ""
	}
	return tx.AuthInfo.Fee.Payer
}

func firstTxMessage(log *zap.Logger, tx *typestx.Tx) (sdk.Msg, bool) {
	if tx == nil || tx.Body == nil {
		return nil, false
	}
	messages, err := typestx.GetMsgs(tx.Body.Messages, "transaction")
	if err != nil {
		log.Info("Could not unpack first msg when attempting to get the fee payer", zap.Error(err))
		return nil, false
	}
	if len(messages) == 0 || messages[0] == nil {
		return nil, false
	}
	return messages[0], true
}

func knownMessageFeePayer(firstMsg sdk.Msg) (string, bool) {
	switch firstMsg := firstMsg.(type) {
	case *transfertypes.MsgTransfer:
		// There is a possible data race around concurrent map access
		// in the cosmos sdk when it converts the address from bech32.
		// We don't need the address conversion; just the sender is all that
		// GetSigners is doing under the hood anyway.
		return firstMsg.Sender, true
	case *clienttypes.MsgCreateClient:
		// Without this particular special case, there is a panic in ibc-go
		// due to the sdk config singleton expecting one bech32 prefix but seeing another.
		return firstMsg.Signer, true
	case *clienttypes.MsgUpdateClient:
		// Same failure mode as MsgCreateClient.
		return firstMsg.Signer, true
	case *clienttypes.MsgUpgradeClient:
		return firstMsg.Signer, true
	case *feetypes.MsgRegisterPayee:
		return firstMsg.Relayer, true
	case *feetypes.MsgRegisterCounterpartyPayee:
		return firstMsg.Relayer, true
	case *feetypes.MsgPayPacketFee:
		return firstMsg.Signer, true
	case *feetypes.MsgPayPacketFeeAsync:
		return firstMsg.PacketFee.RefundAddress, true
	}
	return "", false
}

func derivedMessageFeePayer(log *zap.Logger, cdc *codec.ProtoCodec, message sdk.Msg) string {
	if cdc == nil {
		return ""
	}
	signers, _, err := cdc.GetMsgV1Signers(message)
	if err != nil {
		log.Info("Could not get signers for msg when attempting to get the fee payer", zap.Error(err))
		return ""
	}
	if len(signers) == 0 {
		return ""
	}
	payer, err := cdc.InterfaceRegistry().SigningContext().AddressCodec().BytesToString(signers[0])
	if err != nil {
		log.Info("Could not encode signer when attempting to get the fee payer", zap.Error(err))
		return ""
	}
	return payer
}
