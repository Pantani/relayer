package v2

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/stretchr/testify/require"
)

const (
	packetGoldenV11_2_0 = "0807120f30372d74656e6465726d696e742d301a0f30372d74656e6465726d696e742d31207b2a340a087472616e7366657212087472616e736665721a0769637332302d3122106170706c69636174696f6e2f6a736f6e2a03010203"
	ackGoldenV11_2_0    = "0a02aabb"
)

func TestDecodePacketHexMatchesIBCGoV11_2_0Golden(t *testing.T) {
	packet, err := DecodePacketHex(packetGoldenV11_2_0)
	require.NoError(t, err)
	require.Equal(t, protocol.ProtocolV2, packet.ID.Protocol)
	require.Equal(t, uint64(7), packet.ID.Sequence)
	require.Equal(t, "07-tendermint-0", packet.ID.Source.ClientID)
	require.Equal(t, "07-tendermint-1", packet.ID.Destination.ClientID)
	require.Equal(t, uint64(123), packet.Timeout.Timestamp)
	require.Equal(t, protocol.TimestampSeconds, packet.Timeout.TimestampUnit)
	require.Equal(t, "transfer", packet.Payloads[0].SourcePort)
	require.Equal(t, "transfer", packet.Payloads[0].DestinationPort)
	require.Equal(t, "ics20-1", packet.Payloads[0].Version)
	require.Equal(t, "application/json", packet.Payloads[0].Encoding)
	require.Equal(t, []byte{1, 2, 3}, packet.Payloads[0].Value)
}

func TestWirePacketMarshalMatchesIBCGoV11_2_0Golden(t *testing.T) {
	encoded, err := proto.Marshal(validWirePacket())
	require.NoError(t, err)
	require.Equal(t, packetGoldenV11_2_0, hex.EncodeToString(encoded))
}

func TestDecodeAcknowledgementHexMatchesIBCGoV11_2_0Golden(t *testing.T) {
	ack, err := DecodeAcknowledgementHex(ackGoldenV11_2_0)
	require.NoError(t, err)
	require.Equal(t, protocol.ProtocolV2, ack.Protocol)
	require.Equal(t, [][]byte{{0xaa, 0xbb}}, ack.AppAcknowledgements)

	encoded, err := proto.Marshal(&wireAcknowledgement{AppAcknowledgements: [][]byte{{0xaa, 0xbb}}})
	require.NoError(t, err)
	require.Equal(t, ackGoldenV11_2_0, hex.EncodeToString(encoded))
}

func TestDecodePacketHexAcceptsUnknownFields(t *testing.T) {
	packet, err := DecodePacketHex(packetGoldenV11_2_0 + "980601")
	require.NoError(t, err)
	require.Equal(t, uint64(7), packet.ID.Sequence)
}

func TestDecodePacketHexRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		target error
	}{
		{name: "odd hex", value: "0", target: ErrInvalidHex},
		{name: "invalid hex", value: "zz", target: ErrInvalidHex},
		{name: "invalid protobuf", value: "2aff", target: ErrInvalidProtobuf},
		{name: "protobuf length overflow", value: "2affffffffffffffffff01", target: ErrInvalidProtobuf},
		{name: "empty packet", value: "", target: ErrInvalidPacketContract},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodePacketHex(test.value)
			require.ErrorIs(t, err, test.target)
		})
	}
}

func TestDecodeErrorsDoNotIncludeEncodedValue(t *testing.T) {
	const value = "deadbeeg"
	_, err := DecodePacketHex(value)
	require.ErrorIs(t, err, ErrInvalidHex)
	require.NotContains(t, err.Error(), value)
}

func TestDecodePacketHexRejectsOversizeBeforeDecode(t *testing.T) {
	value := strings.Repeat("00", MaximumEncodedPacketSize+1)
	_, err := DecodePacketHex(value)
	require.ErrorIs(t, err, ErrEncodedValueTooLarge)
	require.NotContains(t, err.Error(), value)
}

func TestDecodePacketHexRejectsOversizedOddBeforeHexValidation(t *testing.T) {
	value := strings.Repeat("0", MaximumEncodedPacketSize*2+1)
	_, err := DecodePacketHex(value)
	require.ErrorIs(t, err, ErrEncodedValueTooLarge)
}

func TestDecodeBoundedHexSizeBoundaries(t *testing.T) {
	const limit = 2

	below, err := decodeBoundedHex("00", limit)
	require.NoError(t, err)
	require.Len(t, below, limit-1)

	atLimit, err := decodeBoundedHex("0000", limit)
	require.NoError(t, err)
	require.Len(t, atLimit, limit)

	_, err = decodeBoundedHex("000000", limit)
	require.ErrorIs(t, err, ErrEncodedValueTooLarge)
}

func TestDecodePacketHexEnforcesPacketSemantics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*wirePacket)
	}{
		{name: "zero sequence", mutate: func(packet *wirePacket) { packet.Sequence = 0 }},
		{name: "zero timeout", mutate: func(packet *wirePacket) { packet.TimeoutTimestamp = 0 }},
		{name: "invalid client", mutate: func(packet *wirePacket) { packet.SourceClient = "bad/id" }},
		{name: "no payload", mutate: func(packet *wirePacket) { packet.Payloads = nil }},
		{name: "multiple payloads", mutate: duplicatePayload},
		{name: "invalid port", mutate: func(packet *wirePacket) { packet.Payloads[0].SourcePort = "a" }},
		{name: "empty version", mutate: func(packet *wirePacket) { packet.Payloads[0].Version = " " }},
		{name: "empty encoding", mutate: func(packet *wirePacket) { packet.Payloads[0].Encoding = "" }},
		{name: "empty value", mutate: func(packet *wirePacket) { packet.Payloads[0].Value = nil }},
		{name: "oversize payload", mutate: oversizePayload},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packet := validWirePacket()
			test.mutate(packet)
			_, err := DecodePacketHex(marshalHex(t, packet))
			require.ErrorIs(t, err, ErrInvalidPacketContract)
		})
	}
}

func TestDecodePacketHexAllowsMaximumPayload(t *testing.T) {
	packet := validWirePacket()
	packet.Payloads[0].Value = make([]byte, protocol.MaximumPayloadsSize)
	decoded, err := DecodePacketHex(marshalHex(t, packet))
	require.NoError(t, err)
	require.Len(t, decoded.Payloads[0].Value, protocol.MaximumPayloadsSize)
}

func TestDecodeAcknowledgementHexRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		target error
	}{
		{name: "odd hex", value: "0", target: ErrInvalidHex},
		{name: "invalid hex", value: "gg", target: ErrInvalidHex},
		{name: "invalid protobuf", value: "0aff", target: ErrInvalidProtobuf},
		{name: "empty acknowledgement", value: "", target: ErrInvalidAcknowledgement},
		{name: "empty app acknowledgement", value: "0a00", target: ErrInvalidAcknowledgement},
		{name: "multiple acknowledgements", value: "0a01010a0102", target: ErrInvalidAcknowledgement},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeAcknowledgementHex(test.value)
			require.ErrorIs(t, err, test.target)
		})
	}
}

func TestDecodeAcknowledgementHexRejectsOversizeBeforeDecode(t *testing.T) {
	value := strings.Repeat("00", MaximumEncodedAcknowledgementSize+1)
	_, err := DecodeAcknowledgementHex(value)
	require.ErrorIs(t, err, ErrEncodedValueTooLarge)
	require.NotContains(t, err.Error(), value)
}

func TestWireDecodeReturnsDefensiveCopies(t *testing.T) {
	ack, err := DecodeAcknowledgementHex(ackGoldenV11_2_0)
	require.NoError(t, err)
	clone := ack.Clone()
	ack.AppAcknowledgements[0][0] = 0
	require.Equal(t, byte(0xaa), clone.AppAcknowledgements[0][0])

	contract, err := ToContractPacket(mustDecodePacket(t))
	require.NoError(t, err)
	packet := mustDecodePacket(t)
	contract.Payloads[0].Value[0] = 0
	require.Equal(t, byte(1), packet.Payloads[0].Value[0])
}

func TestWireErrorsSupportErrorsIs(t *testing.T) {
	_, packetErr := DecodePacketHex("zz")
	_, ackErr := DecodeAcknowledgementHex("")
	require.True(t, errors.Is(packetErr, ErrInvalidHex))
	require.True(t, errors.Is(ackErr, ErrInvalidAcknowledgement))
}

func validWirePacket() *wirePacket {
	return &wirePacket{
		Sequence:          7,
		SourceClient:      "07-tendermint-0",
		DestinationClient: "07-tendermint-1",
		TimeoutTimestamp:  123,
		Payloads: []*wirePayload{{
			SourcePort:      "transfer",
			DestinationPort: "transfer",
			Version:         "ics20-1",
			Encoding:        "application/json",
			Value:           []byte{1, 2, 3},
		}},
	}
}

func duplicatePayload(packet *wirePacket) {
	duplicate := *packet.Payloads[0]
	duplicate.Value = append([]byte(nil), duplicate.Value...)
	packet.Payloads = append(packet.Payloads, &duplicate)
}

func oversizePayload(packet *wirePacket) {
	packet.Payloads[0].Value = make([]byte, protocol.MaximumPayloadsSize+1)
}

func marshalHex(t *testing.T, message proto.Message) string {
	t.Helper()
	encoded, err := proto.Marshal(message)
	require.NoError(t, err)
	return hex.EncodeToString(encoded)
}

func mustDecodePacket(t *testing.T) protocol.PacketEnvelope {
	t.Helper()
	packet, err := DecodePacketHex(packetGoldenV11_2_0)
	require.NoError(t, err)
	return packet
}
