package v2

import (
	"testing"

	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/stretchr/testify/require"
)

func TestContractPacketRoundTripMatchesV11_2_0(t *testing.T) {
	packet := ContractPacket{
		Sequence:          7,
		SourceClient:      "07-tendermint-0",
		DestinationClient: "07-tendermint-1",
		TimeoutTimestamp:  123,
		Payloads: []ContractPayload{{
			SourcePort: "transfer", DestinationPort: "transfer",
			Version: "ics20-1", Encoding: "application/json", Value: []byte{1, 2, 3},
		}},
	}

	envelope, err := FromContractPacket(packet)
	require.NoError(t, err)
	require.Equal(t, protocol.ProtocolV2, envelope.ID.Protocol)
	require.Equal(t, protocol.TimestampSeconds, envelope.Timeout.TimestampUnit)

	got, err := ToContractPacket(envelope)
	require.NoError(t, err)
	require.Equal(t, packet, got)

	packet.Payloads[0].Value[0] = 9
	require.Equal(t, byte(1), envelope.Payloads[0].Value[0])
}

func TestContractPacketRejectsInvalidPayloadCounts(t *testing.T) {
	packet := ContractPacket{
		Sequence: 1, SourceClient: "client-a", DestinationClient: "client-b", TimeoutTimestamp: 1,
	}
	_, err := FromContractPacket(packet)
	require.Error(t, err)

	payload := ContractPayload{SourcePort: "a", DestinationPort: "b", Version: "v", Encoding: "e", Value: []byte{1}}
	packet.Payloads = []ContractPayload{payload, payload}
	_, err = FromContractPacket(packet)
	require.Error(t, err)
}

func TestContractMsgSendPacketHasNoAssignedPacketID(t *testing.T) {
	message := ContractMsgSendPacket{
		SourceClient:     "07-tendermint-0",
		TimeoutTimestamp: 123,
		Signer:           "signer",
		Payloads: []ContractPayload{{
			SourcePort: "transfer", DestinationPort: "transfer",
			Version: "ics20-1", Encoding: "application/json", Value: []byte{1},
		}},
	}

	envelope, err := FromContractMsgSendPacket(message)
	require.NoError(t, err)
	require.Nil(t, envelope.Packet)
	require.NotNil(t, envelope.Send)
	require.Equal(t, message.SourceClient, envelope.Send.SourceClient)
}

func TestEventConstantsMatchIBCGoV11_2_0(t *testing.T) {
	require.Equal(t, "send_packet", EventTypeSendPacket)
	require.Equal(t, "recv_packet", EventTypeRecvPacket)
	require.Equal(t, "timeout_packet", EventTypeTimeoutPacket)
	require.Equal(t, "acknowledge_packet", EventTypeAcknowledgePacket)
	require.Equal(t, "write_acknowledgement", EventTypeWriteAck)
	require.Equal(t, "packet_source_client", AttributeKeySrcClient)
	require.Equal(t, "packet_dest_client", AttributeKeyDstClient)
	require.Equal(t, "packet_sequence", AttributeKeySequence)
	require.Equal(t, "packet_timeout_timestamp", AttributeKeyTimeoutTimestamp)
	require.Equal(t, "encoded_packet_hex", AttributeKeyEncodedPacketHex)
	require.Equal(t, "encoded_acknowledgement_hex", AttributeKeyEncodedAckHex)
}
