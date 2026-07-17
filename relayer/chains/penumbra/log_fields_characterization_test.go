package penumbra

import (
	"fmt"
	"testing"

	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestPenumbraGetChannelsCharacterizesAllChannelAndPortValueCombinations(t *testing.T) {
	for mask := 0; mask < 16; mask++ {
		t.Run(fmt.Sprintf("mask-%04b", mask), func(t *testing.T) {
			srcChannel := penumbraLogCombinationValue(mask, 1, "channel-src")
			dstChannel := penumbraLogCombinationValue(mask, 2, "channel-dst")
			srcPort := penumbraLogCombinationValue(mask, 4, "port-src")
			dstPort := penumbraLogCombinationValue(mask, 8, "port-dst")
			events := []provider.RelayerEvent{{Attributes: map[string]string{
				srcChanTag:        srcChannel,
				dstChanTag:        dstChannel,
				"packet_src_port": srcPort,
				"packet_dst_port": dstPort,
			}}}

			fields := getChannelsIfPresent(events)

			require.Equal(t, []zapcore.Field{
				zap.String(srcChanTag, srcChannel),
				zap.String(dstChanTag, dstChannel),
			}, fields)
			require.Len(t, fields, 2)
			require.Equal(t, 2, cap(fields))
		})
	}
}

func TestPenumbraGetChannelsCharacterizesMissingTagsAsNonNilEmpty(t *testing.T) {
	tests := []struct {
		name   string
		events []provider.RelayerEvent
	}{
		{name: "nil events"},
		{name: "empty events", events: []provider.RelayerEvent{}},
		{name: "nil attributes", events: []provider.RelayerEvent{{Attributes: nil}}},
		{name: "empty attributes", events: []provider.RelayerEvent{{Attributes: map[string]string{}}}},
		{name: "ports only", events: []provider.RelayerEvent{{Attributes: map[string]string{
			"packet_src_port": "transfer",
			"packet_dst_port": "",
		}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := getChannelsIfPresent(tt.events)
			require.NotNil(t, fields)
			require.Empty(t, fields)
			require.Zero(t, cap(fields))
		})
	}
}

func TestPenumbraGetChannelsCharacterizesFirstOccurrenceAndCrossEventOrder(t *testing.T) {
	events := []provider.RelayerEvent{
		{Attributes: map[string]string{dstChanTag: "dst-first"}},
		{Attributes: map[string]string{
			srcChanTag: "src-first",
			dstChanTag: "dst-second",
		}},
		{Attributes: map[string]string{srcChanTag: "src-second"}},
	}

	fields := getChannelsIfPresent(events)

	require.Equal(t, []zapcore.Field{
		zap.String(dstChanTag, "dst-first"),
		zap.String(srcChanTag, "src-first"),
	}, fields)
}

func TestPenumbraGetChannelsCharacterizesSameEventTagOrder(t *testing.T) {
	events := []provider.RelayerEvent{{Attributes: map[string]string{
		dstChanTag: "dst",
		srcChanTag: "src",
	}}}

	fields := getChannelsIfPresent(events)

	require.Equal(t, []zapcore.Field{
		zap.String(srcChanTag, "src"),
		zap.String(dstChanTag, "dst"),
	}, fields)
}

func TestPenumbraGetChannelsCharacterizesSingleFieldCapacityAndExactType(t *testing.T) {
	fields := getChannelsIfPresent([]provider.RelayerEvent{{Attributes: map[string]string{
		srcChanTag: "channel-7",
	}}})

	require.Equal(t, []zapcore.Field{zap.String(srcChanTag, "channel-7")}, fields)
	require.Len(t, fields, 1)
	require.Equal(t, 1, cap(fields))
	require.Equal(t, zapcore.StringType, fields[0].Type)
	require.Equal(t, srcChanTag, fields[0].Key)
	require.Equal(t, "channel-7", fields[0].String)
	require.Zero(t, fields[0].Integer)
	require.Nil(t, fields[0].Interface)
}

func TestPenumbraGetChannelsCharacterizesIndependentOutputAndInput(t *testing.T) {
	attributes := map[string]string{srcChanTag: "channel-original"}
	events := []provider.RelayerEvent{{Attributes: attributes}}
	first := getChannelsIfPresent(events)
	second := getChannelsIfPresent(events)

	first[0].String = "mutated-output"
	first = append(first, zap.String("extra", "field"))

	require.Equal(t, "channel-original", second[0].String)
	require.Equal(t, "channel-original", attributes[srcChanTag])
	require.Len(t, second, 1)
	require.Equal(t, 1, cap(second))
	require.Len(t, first, 2)
}

func penumbraLogCombinationValue(mask, bit int, value string) string {
	if mask&bit != 0 {
		return value
	}
	return ""
}
