package module

import (
	proto "github.com/cosmos/gogoproto/proto"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
)

const legacyLocalhostClientStateTypeURL = "ibc.lightclients.localhost.v2.ClientState"

// legacyLocalhostClientState is a read-only wire compatibility type for chains
// that still return the stateful localhost client removed from IBC-Go v11.
// The relayer never creates or updates this client; registration only lets query
// responses containing the sentinel coexist with supported client types.
type legacyLocalhostClientState struct {
	LatestHeight clienttypes.Height `protobuf:"bytes,1,opt,name=latest_height,json=latestHeight,proto3" json:"latest_height"`
}

func (state *legacyLocalhostClientState) Reset() {
	*state = legacyLocalhostClientState{}
}

func (state *legacyLocalhostClientState) String() string {
	return proto.CompactTextString(state)
}

func (*legacyLocalhostClientState) ProtoMessage() {}

func (*legacyLocalhostClientState) XXX_MessageName() string {
	return legacyLocalhostClientStateTypeURL
}

func (*legacyLocalhostClientState) ClientType() string {
	return ibcexported.Localhost
}

func (*legacyLocalhostClientState) Validate() error {
	return nil
}
