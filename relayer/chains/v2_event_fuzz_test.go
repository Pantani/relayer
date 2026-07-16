package chains

import (
	"strconv"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	protocolv2 "github.com/cosmos/relayer/v2/relayer/protocol/v2"
	"go.uber.org/zap"
)

func FuzzDecodeV2PacketHexNeverPanics(f *testing.F) {
	for _, seed := range []string{v11PacketHex, "", "0", "xyz", "0a", "00"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		packet, err := protocolv2.DecodePacketHex(value)
		if err != nil {
			return
		}
		if err := packet.Validate(); err != nil {
			t.Fatalf("successful decode produced invalid packet: %v", err)
		}
	})
}

func FuzzCorrelateMessageActionsNeverPanics(f *testing.F) {
	f.Add("/fixture.Msg", "0", false)
	f.Add("", "0", false)
	f.Add("/fixture.Msg", "x", false)
	f.Add("/fixture.MsgA", "0", true)
	f.Add("/fixture.Legacy", "", false)

	f.Fuzz(func(t *testing.T, action, messageIndex string, conflict bool) {
		batch := fuzzMessageActionBatch(action, messageIndex, conflict)
		expectedIndex, valid := expectedFuzzActionIndex(action, messageIndex, conflict)
		validateFuzzSidecars(t, batch, boundedFuzzString(action, 128), expectedIndex, valid)
	})
}

func fuzzMessageActionBatch(action, messageIndex string, conflict bool) IBCEventBatch {
	action = boundedFuzzString(action, 128)
	messageIndex = boundedFuzzString(messageIndex, 32)
	actionEvent := messageActionEvent(action, messageIndex)
	if conflict {
		actionEvent.Attributes = append(actionEvent.Attributes, abci.EventAttribute{Key: sdk.AttributeKeyAction, Value: action + "x"})
	}
	events := []abci.Event{actionEvent, keeperModuleEvent(messageIndex), v2PacketEvent(protocolv2.EventTypeSendPacket, messageIndex)}
	return ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())
}

func validateFuzzSidecars(t *testing.T, batch IBCEventBatch, action string, expectedIndex uint32, valid bool) {
	t.Helper()
	if !valid {
		if len(batch.V2Packets) != 0 {
			t.Fatalf("invalid correlation published %d sidecars", len(batch.V2Packets))
		}
		return
	}
	if len(batch.V2Packets) != 1 {
		t.Fatalf("valid correlation published %d sidecars", len(batch.V2Packets))
	}
	validateFuzzSidecar(t, batch.V2Packets[0], action, expectedIndex)
}

func validateFuzzSidecar(t *testing.T, packetEvent V2PacketEvent, action string, expectedIndex uint32) {
	t.Helper()
	if err := packetEvent.Event.Validate(); err != nil {
		t.Fatalf("successful sidecar has invalid event: %v", err)
	}
	if err := packetEvent.Observation.Validate(); err != nil {
		t.Fatalf("successful sidecar has invalid observation: %v", err)
	}
	if packetEvent.Event.Action.Index != expectedIndex || packetEvent.Event.Action.Type != action {
		t.Fatalf("sidecar action mismatch: %+v", packetEvent.Event.Action)
	}
}

func expectedFuzzActionIndex(action, messageIndex string, conflict bool) (uint32, bool) {
	action = boundedFuzzString(action, 128)
	messageIndex = boundedFuzzString(messageIndex, 32)
	if action == "" || conflict {
		return 0, false
	}
	if messageIndex == "" {
		return 0, true
	}
	parsed, err := strconv.ParseUint(messageIndex, 10, 32)
	return uint32(parsed), err == nil
}

func boundedFuzzString(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}
