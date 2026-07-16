package ics29

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/v11/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v11/modules/core/errors"
)

const maximumCounterpartyPayeeLength = 2048

// ErrCounterpartyPayeeEmpty preserves the registered v8 ICS-29 ABCI contract.
var ErrCounterpartyPayeeEmpty = errorsmod.Register("feeibc", 7, "counterparty payee must not be empty")

// NewMsgRegisterCounterpartyPayee preserves the v8 ICS-29 constructor used by
// Classic paths while compiling the rest of the relayer against ibc-go v11.
func NewMsgRegisterCounterpartyPayee(
	portID, channelID, relayerAddress, counterpartyPayeeAddress string,
) *MsgRegisterCounterpartyPayee {
	return &MsgRegisterCounterpartyPayee{
		PortId:            portID,
		ChannelId:         channelID,
		Relayer:           relayerAddress,
		CounterpartyPayee: counterpartyPayeeAddress,
	}
}

// ValidateBasic performs the legacy stateless validation contract.
func (msg MsgRegisterCounterpartyPayee) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}
	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(msg.Relayer); err != nil {
		return errorsmod.Wrap(err, "failed to create sdk.AccAddress from relayer address")
	}
	if strings.TrimSpace(msg.CounterpartyPayee) == "" {
		return ErrCounterpartyPayeeEmpty
	}
	if len(msg.CounterpartyPayee) > maximumCounterpartyPayeeLength {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidAddress,
			"counterparty payee address must not exceed %d bytes",
			maximumCounterpartyPayeeLength,
		)
	}
	return nil
}

// GetSigners returns the legacy signer encoded in the message.
func (msg MsgRegisterCounterpartyPayee) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
