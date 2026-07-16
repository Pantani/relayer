package relayer

import (
	"testing"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
)

func TestParseClientIDFromEvents(t *testing.T) {
	events := []provider.RelayerEvent{
		{EventType: "irrelevant", Attributes: map[string]string{clienttypes.AttributeKeyClientID: "ignored"}},
		{EventType: clienttypes.EventTypeCreateClient, Attributes: map[string]string{clienttypes.AttributeKeyClientID: "07-tendermint-0"}},
	}

	clientID, err := ParseClientIDFromEvents(events)
	require.NoError(t, err)
	require.Equal(t, "07-tendermint-0", clientID)

	emptyClientID, err := ParseClientIDFromEvents([]provider.RelayerEvent{{
		EventType:  clienttypes.EventTypeCreateClient,
		Attributes: map[string]string{clienttypes.AttributeKeyClientID: ""},
	}})
	require.NoError(t, err)
	require.Empty(t, emptyClientID)

	_, err = ParseClientIDFromEvents([]provider.RelayerEvent{{EventType: clienttypes.EventTypeCreateClient}})
	require.EqualError(t, err, "client identifier event attribute not found")
}

func TestParseConnectionIDFromEvents(t *testing.T) {
	for _, eventType := range []string{
		connectiontypes.EventTypeConnectionOpenInit,
		connectiontypes.EventTypeConnectionOpenTry,
	} {
		t.Run(eventType, func(t *testing.T) {
			connectionID, err := ParseConnectionIDFromEvents([]provider.RelayerEvent{{
				EventType:  eventType,
				Attributes: map[string]string{connectiontypes.AttributeKeyConnectionID: "connection-0"},
			}})
			require.NoError(t, err)
			require.Equal(t, "connection-0", connectionID)
		})
	}

	_, err := ParseConnectionIDFromEvents([]provider.RelayerEvent{{
		EventType:  "irrelevant",
		Attributes: map[string]string{connectiontypes.AttributeKeyConnectionID: "connection-0"},
	}})
	require.EqualError(t, err, "connection identifier event attribute not found")
}

func TestParseChannelIDFromEvents(t *testing.T) {
	for _, eventType := range []string{
		channeltypes.EventTypeChannelOpenInit,
		channeltypes.EventTypeChannelOpenTry,
	} {
		t.Run(eventType, func(t *testing.T) {
			channelID, err := ParseChannelIDFromEvents([]provider.RelayerEvent{{
				EventType:  eventType,
				Attributes: map[string]string{channeltypes.AttributeKeyChannelID: "channel-0"},
			}})
			require.NoError(t, err)
			require.Equal(t, "channel-0", channelID)
		})
	}

	_, err := ParseChannelIDFromEvents([]provider.RelayerEvent{{
		EventType:  channeltypes.EventTypeChannelOpenInit,
		Attributes: map[string]string{"irrelevant": "channel-0"},
	}})
	require.EqualError(t, err, "channel identifier event attribute not found")
}
