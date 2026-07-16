package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolCapabilitiesAreIsolated(t *testing.T) {
	classic, err := CapabilitiesFor(ProtocolClassic)
	require.NoError(t, err)
	require.True(t, classic.ConnectionHandshake)
	require.True(t, classic.ChannelHandshake)
	require.True(t, classic.OrderedDelivery)
	require.True(t, classic.TimeoutHeight)
	require.False(t, classic.ClientRouting)

	v2, err := CapabilitiesFor(ProtocolV2)
	require.NoError(t, err)
	require.True(t, v2.ClientRouting)
	require.True(t, v2.PerClientRelayerAllowlist)
	require.False(t, v2.ConnectionHandshake)
	require.False(t, v2.ChannelHandshake)
	require.False(t, v2.OrderedDelivery)
	require.False(t, v2.TimeoutHeight)
	require.Equal(t, 1, v2.MaxPayloads)

	_, err = CapabilitiesFor(ProtocolUnspecified)
	require.Error(t, err)
}

func TestPacketKeysAreProtocolIsolated(t *testing.T) {
	classic := PacketID{
		Protocol:    ProtocolClassic,
		Source:      Endpoint{PortID: "transfer", ChannelID: "channel-0"},
		Destination: Endpoint{PortID: "transfer", ChannelID: "channel-1"},
		Sequence:    7,
	}
	v2A := PacketID{
		Protocol:    ProtocolV2,
		Source:      Endpoint{ClientID: "07-tendermint-0"},
		Destination: Endpoint{ClientID: "07-tendermint-1"},
		Sequence:    7,
	}
	v2B := v2A
	v2B.Destination.ClientID = "07-tendermint-2"

	keys := map[PacketID]struct{}{classic: {}, v2A: {}, v2B: {}}
	require.Len(t, keys, 3)
}

func TestEventEnvelopePreservesOrderAndDuplicates(t *testing.T) {
	event := EventEnvelope{
		Protocol:   ProtocolV2,
		Type:       "send_packet",
		TxHash:     "ABC123",
		EventIndex: 2,
		Action:     MessageAction{Present: true, Index: 0, Type: "/ibc.core.channel.v2.MsgSendPacket"},
		Attributes: []EventAttribute{
			{Key: "message.action", Value: "A"},
			{Key: "packet_sequence", Value: "7"},
			{Key: "message.action", Value: "B"},
			{Key: "packet_sequence", Value: "8"},
		},
	}

	require.NoError(t, event.Validate())
	require.Equal(t, []string{"A", "B"}, event.AttributeValues("message.action"))
	require.Equal(t, []string{"7", "8"}, event.AttributeValues("packet_sequence"))

	clone := event.Clone()
	clone.Attributes[0].Value = "changed"
	require.Equal(t, "A", event.Attributes[0].Value)
	require.Equal(t, uint32(2), clone.EventIndex)
	require.Equal(t, uint32(0), clone.Action.Index)
}

func TestPacketEnvelopeRejectsMixedRoutingAndTimeoutUnits(t *testing.T) {
	classic := validClassicPacket()
	require.NoError(t, classic.Validate())

	classic.ID.Source.ClientID = "07-tendermint-0"
	require.Error(t, classic.Validate())

	v2 := validV2Packet()
	require.NoError(t, v2.Validate())
	v2.Timeout.TimestampUnit = TimestampNanoseconds
	require.Error(t, v2.Validate())
}

func TestV2PacketPayloadContract(t *testing.T) {
	packet := validV2Packet()
	require.NoError(t, packet.Validate())

	packet.Payloads = nil
	require.Error(t, packet.Validate())
	packet.Payloads = []Payload{{}, {}}
	require.Error(t, packet.Validate())

	packet = validV2Packet()
	packet.Payloads[0].Value = make([]byte, MaximumPayloadsSize+1)
	require.Error(t, packet.Validate())
}

func TestMessageValidationMatrix(t *testing.T) {
	packet := validV2Packet()
	proof := ProofEnvelope{
		Protocol: ProtocolV2,
		Kind:     ProofPacketCommitment,
		Height:   Height{RevisionNumber: 1, RevisionHeight: 10},
		Data:     []byte{1},
	}
	recv := MessageEnvelope{
		Protocol: ProtocolV2,
		Kind:     MessageRecvPacket,
		Packet:   &packet,
		Proof:    &proof,
		Signer:   "signer",
	}
	require.NoError(t, recv.Validate())
	recv.Proof = nil
	require.Error(t, recv.Validate())

	sendRequest := SendPacketRequest{
		Protocol:     ProtocolV2,
		SourceClient: packet.ID.Source.ClientID,
		Timeout:      packet.Timeout,
		Payloads:     packet.Payloads,
	}
	send := MessageEnvelope{Protocol: ProtocolV2, Kind: MessageSendPacket, Send: &sendRequest, Signer: "signer"}
	require.NoError(t, send.Validate())
	send.Proof = &proof
	require.Error(t, send.Validate())
}

func validClassicPacket() PacketEnvelope {
	return PacketEnvelope{
		ID: PacketID{
			Protocol:    ProtocolClassic,
			Source:      Endpoint{PortID: "transfer", ChannelID: "channel-0"},
			Destination: Endpoint{PortID: "transfer", ChannelID: "channel-1"},
			Sequence:    1,
		},
		Timeout: Timeout{
			Height:        Height{RevisionNumber: 1, RevisionHeight: 10},
			Timestamp:     100,
			TimestampUnit: TimestampNanoseconds,
		},
		Payloads: []Payload{{SourcePort: "transfer", DestinationPort: "transfer", Value: []byte{1}}},
	}
}

func validV2Packet() PacketEnvelope {
	return PacketEnvelope{
		ID: PacketID{
			Protocol:    ProtocolV2,
			Source:      Endpoint{ClientID: "07-tendermint-0"},
			Destination: Endpoint{ClientID: "07-tendermint-1"},
			Sequence:    1,
		},
		Timeout: Timeout{Timestamp: 100, TimestampUnit: TimestampSeconds},
		Payloads: []Payload{{
			SourcePort: "transfer", DestinationPort: "transfer",
			Version: "ics20-1", Encoding: "application/json", Value: []byte{1},
		}},
	}
}
