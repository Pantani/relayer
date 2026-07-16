package classic

import (
	"testing"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
)

func TestPacketInfoRoundTripAndDefensiveCopies(t *testing.T) {
	original := provider.PacketInfo{
		Height:           42,
		Sequence:         7,
		SourcePort:       "transfer",
		SourceChannel:    "channel-1",
		DestPort:         "transfer",
		DestChannel:      "channel-9",
		ChannelOrder:     "ORDER_ORDERED",
		Data:             []byte{1, 2, 3},
		TimeoutHeight:    clienttypes.NewHeight(2, 100),
		TimeoutTimestamp: 123456789,
		Ack:              []byte{4, 5, 6},
	}

	observation, err := FromPacketInfo("send_packet", original)
	require.NoError(t, err)
	require.Equal(t, protocol.ProtocolClassic, observation.Packet.ID.Protocol)
	require.Equal(t, "send_packet", observation.EventType)
	require.Equal(t, protocol.TimestampNanoseconds, observation.Packet.Timeout.TimestampUnit)

	original.Data[0] = 99
	original.Ack[0] = 99
	require.Equal(t, byte(1), observation.Packet.Payloads[0].Value[0])
	require.Equal(t, byte(4), observation.Acknowledgement[0])

	want := original
	want.Data = []byte{1, 2, 3}
	want.Ack = []byte{4, 5, 6}
	roundTrip, err := ToPacketInfo(observation)
	require.NoError(t, err)
	require.Equal(t, want, roundTrip)

	observation.Packet.Payloads[0].Value[0] = 88
	observation.Acknowledgement[0] = 88
	require.Equal(t, byte(1), roundTrip.Data[0])
	require.Equal(t, byte(4), roundTrip.Ack[0])
}

func TestPacketInfoAdapterRejectsForeignProtocol(t *testing.T) {
	observation := validObservation(t)
	observation.Protocol = protocol.ProtocolV2
	_, err := ToPacketInfo(observation)
	require.ErrorContains(t, err, "Classic adapter")

	observation = validObservation(t)
	observation.Packet.ID.Protocol = protocol.ProtocolV2
	_, err = ToPacketInfo(observation)
	require.ErrorContains(t, err, "Classic adapter")
}

func TestPacketInfoHeightOnlyTimeoutRoundTrip(t *testing.T) {
	original := provider.PacketInfo{
		Sequence:      9,
		SourcePort:    "transfer",
		SourceChannel: "channel-2",
		DestPort:      "transfer",
		DestChannel:   "channel-3",
		TimeoutHeight: clienttypes.NewHeight(4, 120),
	}

	observation, err := FromPacketInfo("send_packet", original)
	require.NoError(t, err)
	require.Equal(t, protocol.TimestampUnitUnspecified, observation.Packet.Timeout.TimestampUnit)

	roundTrip, err := ToPacketInfo(observation)
	require.NoError(t, err)
	require.Equal(t, original, roundTrip)
}

func TestPacketInfoPreservesNilAndEmptyBytes(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		ack  []byte
	}{
		{name: "nil", data: nil, ack: nil},
		{name: "empty", data: []byte{}, ack: []byte{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			original := validPacketInfo()
			original.Data = testCase.data
			original.Ack = testCase.ack

			observation, err := FromPacketInfo(chantypes.EventTypeSendPacket, original)
			require.NoError(t, err)
			roundTrip, err := ToPacketInfo(observation)
			require.NoError(t, err)
			require.Equal(t, original, roundTrip)
		})
	}
}

func TestClassicPacketEventTypes(t *testing.T) {
	eventTypes := []string{
		chantypes.EventTypeSendPacket,
		chantypes.EventTypeRecvPacket,
		chantypes.EventTypeWriteAck,
		chantypes.EventTypeAcknowledgePacket,
		chantypes.EventTypeTimeoutPacket,
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			observation, err := FromPacketInfo(eventType, validPacketInfo())
			require.NoError(t, err)
			require.Equal(t, eventType, observation.EventType)

			_, err = ToPacketInfo(observation)
			require.NoError(t, err)
		})
	}

	_, err := FromPacketInfo("unknown_packet_event", validPacketInfo())
	require.ErrorContains(t, err, "not a Classic packet event")

	observation := validObservation(t)
	observation.EventType = "unknown_packet_event"
	_, err = ToPacketInfo(observation)
	require.ErrorContains(t, err, "not a Classic packet event")
}

func TestPacketProofRoundTripAndDefensiveCopies(t *testing.T) {
	kinds := []protocol.ProofKind{
		protocol.ProofPacketCommitment,
		protocol.ProofAcknowledgement,
		protocol.ProofReceiptAbsence,
		protocol.ProofNextSequenceRecv,
	}

	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			original := provider.PacketProof{
				Proof:       []byte{0, 255, 1},
				ProofHeight: clienttypes.NewHeight(3, 77),
			}

			envelope, err := FromPacketProof(kind, original)
			require.NoError(t, err)
			original.Proof[0] = 9
			require.Equal(t, byte(0), envelope.Data[0])

			roundTrip, err := ToPacketProof(envelope)
			require.NoError(t, err)
			require.Equal(t, []byte{0, 255, 1}, roundTrip.Proof)
			require.Equal(t, clienttypes.NewHeight(3, 77), roundTrip.ProofHeight)

			envelope.Data[0] = 8
			require.Equal(t, byte(0), roundTrip.Proof[0])
		})
	}
}

func TestPacketProofAdapterRejectsForeignProtocolAndKind(t *testing.T) {
	proof := provider.PacketProof{
		Proof:       []byte{1},
		ProofHeight: clienttypes.NewHeight(1, 1),
	}

	_, err := FromPacketProof(protocol.ProofKind("client_state"), proof)
	require.ErrorContains(t, err, "not a Classic packet proof")

	envelope, err := FromPacketProof(protocol.ProofPacketCommitment, proof)
	require.NoError(t, err)
	envelope.Protocol = protocol.ProtocolV2
	_, err = ToPacketProof(envelope)
	require.ErrorContains(t, err, "Classic adapter")

	envelope.Protocol = protocol.ProtocolClassic
	envelope.Kind = protocol.ProofKind("client_state")
	_, err = ToPacketProof(envelope)
	require.ErrorContains(t, err, "not a Classic packet proof")
}

func validObservation(t *testing.T) protocol.PacketObservation {
	t.Helper()
	observation, err := FromPacketInfo("send_packet", validPacketInfo())
	require.NoError(t, err)
	return observation
}

func validPacketInfo() provider.PacketInfo {
	return provider.PacketInfo{
		Height:           1,
		Sequence:         1,
		SourcePort:       "transfer",
		SourceChannel:    "channel-0",
		DestPort:         "transfer",
		DestChannel:      "channel-1",
		Data:             []byte{1},
		TimeoutTimestamp: 1,
	}
}
