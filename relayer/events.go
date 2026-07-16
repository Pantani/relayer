package relayer

import (
	"errors"
	"slices"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/provider"
)

// ParseClientIDFromEvents parses events emitted from a MsgCreateClient and returns the
// client identifier.
func ParseClientIDFromEvents(events []provider.RelayerEvent) (string, error) {
	return parseIdentifierFromEvents(
		events,
		clienttypes.AttributeKeyClientID,
		"client identifier event attribute not found",
		clienttypes.EventTypeCreateClient,
	)
}

// ParseConnectionIDFromEvents parses events emitted from a MsgConnectionOpenInit or
// MsgConnectionOpenTry and returns the connection identifier.
func ParseConnectionIDFromEvents(events []provider.RelayerEvent) (string, error) {
	return parseIdentifierFromEvents(
		events,
		connectiontypes.AttributeKeyConnectionID,
		"connection identifier event attribute not found",
		connectiontypes.EventTypeConnectionOpenInit,
		connectiontypes.EventTypeConnectionOpenTry,
	)
}

// ParseChannelIDFromEvents parses events emitted from a MsgChannelOpenInit or
// MsgChannelOpenTry and returns the channel identifier.
func ParseChannelIDFromEvents(events []provider.RelayerEvent) (string, error) {
	return parseIdentifierFromEvents(
		events,
		channeltypes.AttributeKeyChannelID,
		"channel identifier event attribute not found",
		channeltypes.EventTypeChannelOpenInit,
		channeltypes.EventTypeChannelOpenTry,
	)
}

func parseIdentifierFromEvents(
	events []provider.RelayerEvent,
	attributeKey,
	errorMessage string,
	eventTypes ...string,
) (string, error) {
	for _, event := range events {
		if !slices.Contains(eventTypes, event.EventType) {
			continue
		}
		if identifier, ok := event.Attributes[attributeKey]; ok {
			return identifier, nil
		}
	}
	return "", errors.New(errorMessage)
}
