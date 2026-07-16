package relayer

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPathEffectiveProtocol(t *testing.T) {
	t.Parallel()

	require.Equal(t, protocol.ProtocolClassic, (&Path{}).EffectiveProtocol())
	require.Equal(t, protocol.ProtocolClassic, (&Path{Protocol: protocol.ProtocolClassic}).EffectiveProtocol())
	require.Equal(t, protocol.ProtocolV2, (&Path{Protocol: protocol.ProtocolV2}).EffectiveProtocol())
}

func TestLegacyPathSerializationDoesNotAddProtocolFields(t *testing.T) {
	t.Parallel()

	legacy := `src:
  chain-id: chain-a-1
  client-id: 07-tendermint-0
  connection-id: connection-0
dst:
  chain-id: chain-b-1
  client-id: 07-tendermint-1
  connection-id: connection-1
src-channel-filter:
  rule: allowlist
  channel-list:
    - channel-0
`

	var path Path
	require.NoError(t, yaml.Unmarshal([]byte(legacy), &path))
	require.Equal(t, protocol.ProtocolClassic, path.EffectiveProtocol())

	yamlOutput, err := yaml.Marshal(path)
	require.NoError(t, err)
	require.NotContains(t, string(yamlOutput), "protocol:")
	require.NotContains(t, string(yamlOutput), "merkle-prefix:")
	require.Contains(t, string(yamlOutput), "src-channel-filter:")

	jsonOutput, err := json.Marshal(path)
	require.NoError(t, err)
	require.NotContains(t, string(jsonOutput), `"protocol"`)
	require.NotContains(t, string(jsonOutput), `"merkle-prefix"`)
}

func TestExplicitPathProtocolRoundTrip(t *testing.T) {
	t.Parallel()

	paths := []*Path{
		validClassicPath(protocol.ProtocolClassic),
		validV2Path(),
	}
	for _, original := range paths {
		yamlOutput, err := yaml.Marshal(original)
		require.NoError(t, err)
		require.Contains(t, string(yamlOutput), "protocol: "+string(original.Protocol))

		var fromYAML Path
		require.NoError(t, yaml.Unmarshal(yamlOutput, &fromYAML))
		requireSamePath(t, original, &fromYAML)

		jsonOutput, err := json.Marshal(original)
		require.NoError(t, err)
		var fromJSON Path
		require.NoError(t, json.Unmarshal(jsonOutput, &fromJSON))
		requireSamePath(t, original, &fromJSON)
	}
}

func TestPathValidateProtocolCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    *Path
		wantErr string
	}{
		{name: "implicit classic", path: validClassicPath(protocol.ProtocolUnspecified)},
		{name: "explicit classic", path: validClassicPath(protocol.ProtocolClassic)},
		{name: "v2", path: validV2Path()},
		{name: "nil path", wantErr: "path is nil"},
		{name: "nil source", path: &Path{Dst: &PathEnd{}}, wantErr: "path source is nil"},
		{name: "nil destination", path: &Path{Src: &PathEnd{}}, wantErr: "path destination is nil"},
		{name: "unknown protocol", path: pathWithProtocol(protocol.Protocol("future")), wantErr: "protocol \"future\" is not supported"},
		{name: "classic prefix", path: classicPathWithPrefix(), wantErr: "path protocol classic cannot set source merkle-prefix"},
		{name: "v2 source connection", path: v2PathWithSourceConnection(), wantErr: "path protocol v2 cannot set source connection-id"},
		{name: "v2 destination connection", path: v2PathWithDestinationConnection(), wantErr: "path protocol v2 cannot set destination connection-id"},
		{name: "v2 filter rule", path: v2PathWithFilter(ChannelFilter{Rule: "allowlist"}), wantErr: "path protocol v2 cannot set src-channel-filter"},
		{name: "v2 filter list", path: v2PathWithFilter(ChannelFilter{ChannelList: []string{"channel-0"}}), wantErr: "path protocol v2 cannot set src-channel-filter"},
		{name: "v2 missing source chain", path: v2PathWithoutSourceChain(), wantErr: "path protocol v2 requires source chain-id"},
		{name: "v2 missing destination client", path: v2PathWithoutDestinationClient(), wantErr: "path protocol v2 requires destination client-id"},
		{name: "v2 missing source prefix", path: v2PathWithoutSourcePrefix(), wantErr: "path protocol v2 requires source merkle-prefix"},
		{name: "v2 empty prefix segment", path: v2PathWithEmptyPrefixSegment(), wantErr: "source merkle-prefix: contains an empty segment at index 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.path.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestPathEnsureClassicRuntime(t *testing.T) {
	t.Parallel()

	require.NoError(t, validClassicPath(protocol.ProtocolUnspecified).EnsureClassicRuntime())
	require.NoError(t, validClassicPath(protocol.ProtocolClassic).EnsureClassicRuntime())
	require.EqualError(t, (*Path)(nil).EnsureClassicRuntime(), "path is nil")
	require.ErrorIs(t, validV2Path().EnsureClassicRuntime(), ErrV2RuntimeNotImplemented)
}

func TestMerklePrefixBytes(t *testing.T) {
	t.Parallel()

	prefix := MerklePrefix{"ibc", "clients", "ibc"}
	got := prefix.Bytes()
	require.Equal(t, [][]byte{[]byte("ibc"), []byte("clients"), []byte("ibc")}, got)

	got[0][0] = 'x'
	require.Equal(t, "ibc", prefix[0])
	require.NoError(t, prefix.Validate())
	require.EqualError(t, MerklePrefix{"ibc", ""}.Validate(), "contains an empty segment at index 1")
	require.Nil(t, MerklePrefix(nil).Bytes())
}

func TestGenPathRemainsImplicitClassic(t *testing.T) {
	t.Parallel()

	path := GenPath("chain-a-1", "chain-b-1")
	require.Equal(t, protocol.ProtocolUnspecified, path.Protocol)
	require.Equal(t, protocol.ProtocolClassic, path.EffectiveProtocol())
	require.False(t, strings.Contains(path.MustYAML(), "protocol:"))
}

func validClassicPath(value protocol.Protocol) *Path {
	return &Path{
		Protocol: value,
		Src: &PathEnd{
			ChainID:      "chain-a-1",
			ClientID:     "07-tendermint-0",
			ConnectionID: "connection-0",
		},
		Dst: &PathEnd{
			ChainID:      "chain-b-1",
			ClientID:     "07-tendermint-1",
			ConnectionID: "connection-1",
		},
	}
}

func requireSamePath(t *testing.T, want, got *Path) {
	t.Helper()
	require.Equal(t, want.Protocol, got.Protocol)
	require.Equal(t, want.Src, got.Src)
	require.Equal(t, want.Dst, got.Dst)
	require.Equal(t, want.Filter.Rule, got.Filter.Rule)
	require.ElementsMatch(t, want.Filter.ChannelList, got.Filter.ChannelList)
}

func validV2Path() *Path {
	return &Path{
		Protocol: protocol.ProtocolV2,
		Src: &PathEnd{
			ChainID:      "chain-a-1",
			ClientID:     "07-tendermint-0",
			MerklePrefix: MerklePrefix{"ibc"},
		},
		Dst: &PathEnd{
			ChainID:      "chain-b-1",
			ClientID:     "07-tendermint-1",
			MerklePrefix: MerklePrefix{"ibc"},
		},
	}
}

func pathWithProtocol(value protocol.Protocol) *Path {
	path := validClassicPath(protocol.ProtocolUnspecified)
	path.Protocol = value
	return path
}

func classicPathWithPrefix() *Path {
	path := validClassicPath(protocol.ProtocolClassic)
	path.Src.MerklePrefix = MerklePrefix{"ibc"}
	return path
}

func v2PathWithSourceConnection() *Path {
	path := validV2Path()
	path.Src.ConnectionID = "connection-0"
	return path
}

func v2PathWithDestinationConnection() *Path {
	path := validV2Path()
	path.Dst.ConnectionID = "connection-1"
	return path
}

func v2PathWithFilter(filter ChannelFilter) *Path {
	path := validV2Path()
	path.Filter = filter
	return path
}

func v2PathWithoutSourceChain() *Path {
	path := validV2Path()
	path.Src.ChainID = ""
	return path
}

func v2PathWithoutDestinationClient() *Path {
	path := validV2Path()
	path.Dst.ClientID = ""
	return path
}

func v2PathWithoutSourcePrefix() *Path {
	path := validV2Path()
	path.Src.MerklePrefix = nil
	return path
}

func v2PathWithEmptyPrefixSegment() *Path {
	path := validV2Path()
	path.Src.MerklePrefix = MerklePrefix{"ibc", ""}
	return path
}
