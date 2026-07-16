package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	comettypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	"github.com/stretchr/testify/require"
)

type matcherProvider struct {
	ChainProvider

	latestHeight    int64
	latestHeightErr error
	consensusState  *clienttypes.QueryConsensusStateResponse
	consensusErr    error
	ibcHeader       IBCHeader
	ibcHeaderErr    error

	calls                []string
	consensusChainHeight int64
	consensusClientID    string
	consensusHeight      ibcexported.Height
	ibcHeaderHeight      int64
}

func (p *matcherProvider) QueryLatestHeight(context.Context) (int64, error) {
	p.calls = append(p.calls, "latest-height")
	return p.latestHeight, p.latestHeightErr
}

func (p *matcherProvider) QueryClientConsensusState(
	_ context.Context,
	chainHeight int64,
	clientID string,
	clientHeight ibcexported.Height,
) (*clienttypes.QueryConsensusStateResponse, error) {
	p.calls = append(p.calls, "client-consensus-state")
	p.consensusChainHeight = chainHeight
	p.consensusClientID = clientID
	p.consensusHeight = clientHeight
	return p.consensusState, p.consensusErr
}

func (p *matcherProvider) QueryIBCHeader(_ context.Context, height int64) (IBCHeader, error) {
	p.calls = append(p.calls, "ibc-header")
	p.ibcHeaderHeight = height
	return p.ibcHeader, p.ibcHeaderErr
}

type matcherIBCHeader struct {
	height         uint64
	consensusState ibcexported.ConsensusState
	nextValsHash   []byte
}

func (h matcherIBCHeader) Height() uint64 {
	return h.height
}

func (h matcherIBCHeader) ConsensusState() ibcexported.ConsensusState {
	return h.consensusState
}

func (h matcherIBCHeader) NextValidatorsHash() []byte {
	return h.nextValsHash
}

func TestClientsMatchSkipsUnsupportedClientType(t *testing.T) {
	existingClient := clienttypes.IdentifiedClientState{
		ClientId: "09-localhost",
		ClientState: &codectypes.Any{
			TypeUrl: "/ibc.lightclients.localhost.v2.ClientState",
			Value:   []byte{},
		},
	}

	clientID, err := ClientsMatch(context.Background(), nil, nil, existingClient, &tmclient.ClientState{})
	require.NoError(t, err)
	require.Empty(t, clientID)
}

func TestClientsMatchRejectsMalformedTendermintClient(t *testing.T) {
	existingClient := clienttypes.IdentifiedClientState{
		ClientId: "07-tendermint-0",
		ClientState: &codectypes.Any{
			TypeUrl: tendermintClientStateTypeURL,
			Value:   []byte{0xff},
		},
	}

	clientID, err := ClientsMatch(context.Background(), nil, nil, existingClient, &tmclient.ClientState{})
	require.Error(t, err)
	require.Empty(t, clientID)
}

func TestCometMatcherRejectsUnexpectedClientStateTypes(t *testing.T) {
	validClient := &tmclient.ClientState{}
	tests := []struct {
		name           string
		existingClient ibcexported.ClientState
		newClient      ibcexported.ClientState
		wantErr        string
	}{
		{
			name:           "new client",
			existingClient: validClient,
			wantErr:        "got type(<nil>) expected type(*tmclient.ClientState)",
		},
		{
			name:      "existing client",
			newClient: validClient,
			wantErr:   "got type(<nil>) expected type(*tmclient.ClientState)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientID, err := cometMatcher(
				context.Background(), nil, nil, "07-tendermint-0", tt.existingClient, tt.newClient,
			)
			require.EqualError(t, err, tt.wantErr)
			require.Empty(t, clientID)
		})
	}
}

func TestCometMatcherSkipsMismatchedAndFrozenClients(t *testing.T) {
	existingClient, newClient := matchingClientStates(time.Hour)
	mismatchedClient := *newClient
	mismatchedClient.ChainId = "chain-2"

	frozenExisting := *existingClient
	frozenExisting.FrozenHeight = clienttypes.NewHeight(1, 3)
	frozenNew := *newClient
	frozenNew.FrozenHeight = frozenExisting.FrozenHeight

	tests := []struct {
		name           string
		existingClient *tmclient.ClientState
		newClient      *tmclient.ClientState
	}{
		{name: "different client state", existingClient: existingClient, newClient: &mismatchedClient},
		{name: "frozen client", existingClient: &frozenExisting, newClient: &frozenNew},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientID, err := cometMatcher(
				context.Background(), nil, nil, "07-tendermint-0", tt.existingClient, tt.newClient,
			)
			require.NoError(t, err)
			require.Empty(t, clientID)
		})
	}
}

func TestCometMatcherPropagatesLatestHeightError(t *testing.T) {
	wantErr := errors.New("latest height unavailable")
	src := &matcherProvider{latestHeightErr: wantErr}
	existingClient, newClient := matchingClientStates(time.Hour)

	clientID, err := cometMatcher(
		context.Background(), src, nil, "07-tendermint-0", existingClient, newClient,
	)

	require.ErrorIs(t, err, wantErr)
	require.Empty(t, clientID)
	require.Equal(t, []string{"latest-height"}, src.calls)
}

func TestCometMatcherPropagatesConsensusStateQueryError(t *testing.T) {
	wantErr := errors.New("consensus state unavailable")
	src := &matcherProvider{latestHeight: 44, consensusErr: wantErr}
	existingClient, newClient := matchingClientStates(time.Hour)

	clientID, err := cometMatcher(
		context.Background(), src, nil, "07-tendermint-0", existingClient, newClient,
	)

	require.ErrorIs(t, err, wantErr)
	require.Empty(t, clientID)
	require.Equal(t, []string{"latest-height", "client-consensus-state"}, src.calls)
	require.Equal(t, int64(44), src.consensusChainHeight)
	require.Equal(t, "07-tendermint-0", src.consensusClientID)
	require.Equal(t, existingClient.LatestHeight, src.consensusHeight)
}

func TestCometMatcherReturnsMatchingClient(t *testing.T) {
	existingClient, newClient := matchingClientStates(time.Hour)
	consensusState := &tmclient.ConsensusState{Timestamp: time.Now()}
	src := &matcherProvider{
		latestHeight:   44,
		consensusState: matcherConsensusResponse(t, consensusState),
	}
	dst := &matcherProvider{ibcHeader: matcherIBCHeader{consensusState: consensusState}}

	clientID, err := cometMatcher(
		context.Background(), src, dst, "07-tendermint-0", existingClient, newClient,
	)

	require.NoError(t, err)
	require.Equal(t, "07-tendermint-0", clientID)
	require.Equal(t, []string{"latest-height", "client-consensus-state"}, src.calls)
	require.Equal(t, []string{"ibc-header"}, dst.calls)
	require.Equal(t, int64(existingClient.LatestHeight.RevisionHeight), dst.ibcHeaderHeight)
}

func TestCometMatcherRejectsExpiredClient(t *testing.T) {
	existingClient, newClient := matchingClientStates(time.Second)
	consensusState := &tmclient.ConsensusState{Timestamp: time.Now().Add(-2 * time.Second)}
	src := &matcherProvider{
		latestHeight:   44,
		consensusState: matcherConsensusResponse(t, consensusState),
	}
	dst := &matcherProvider{}

	clientID, err := cometMatcher(
		context.Background(), src, dst, "07-tendermint-0", existingClient, newClient,
	)

	require.ErrorIs(t, err, tmclient.ErrTrustingPeriodExpired)
	require.Empty(t, clientID)
	require.Empty(t, dst.calls)
}

func TestCometMatcherPropagatesCounterpartyHeaderError(t *testing.T) {
	wantErr := errors.New("counterparty header unavailable")
	existingClient, newClient := matchingClientStates(time.Hour)
	consensusState := &tmclient.ConsensusState{Timestamp: time.Now()}
	src := &matcherProvider{
		latestHeight:   44,
		consensusState: matcherConsensusResponse(t, consensusState),
	}
	dst := &matcherProvider{ibcHeaderErr: wantErr}

	clientID, err := cometMatcher(
		context.Background(), src, dst, "07-tendermint-0", existingClient, newClient,
	)

	require.ErrorIs(t, err, wantErr)
	require.Empty(t, clientID)
}

func TestCometMatcherRejectsUnexpectedCounterpartyConsensusState(t *testing.T) {
	existingClient, newClient := matchingClientStates(time.Hour)
	consensusState := &tmclient.ConsensusState{Timestamp: time.Now()}
	src := &matcherProvider{
		latestHeight:   44,
		consensusState: matcherConsensusResponse(t, consensusState),
	}
	dst := &matcherProvider{ibcHeader: matcherIBCHeader{}}

	clientID, err := cometMatcher(
		context.Background(), src, dst, "07-tendermint-0", existingClient, newClient,
	)

	require.EqualError(t, err, "got type(*tendermint.ConsensusState) expected type(*tmclient.ConsensusState)")
	require.Empty(t, clientID)
}

func TestCometMatcherSkipsDifferentCounterpartyConsensusState(t *testing.T) {
	existingClient, newClient := matchingClientStates(time.Hour)
	existingConsensusState := &tmclient.ConsensusState{Timestamp: time.Now()}
	counterpartyConsensusState := &tmclient.ConsensusState{Timestamp: existingConsensusState.Timestamp.Add(time.Second)}
	src := &matcherProvider{
		latestHeight:   44,
		consensusState: matcherConsensusResponse(t, existingConsensusState),
	}
	dst := &matcherProvider{ibcHeader: matcherIBCHeader{consensusState: counterpartyConsensusState}}

	clientID, err := cometMatcher(
		context.Background(), src, dst, "07-tendermint-0", existingClient, newClient,
	)

	require.NoError(t, err)
	require.Empty(t, clientID)
}

func TestCheckForMisbehaviourRejectsMalformedMessage(t *testing.T) {
	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), nil, "07-tendermint-0", []byte{0xff}, nil,
	)

	require.Error(t, err)
	require.Nil(t, misbehaviour)
}

func TestCheckForMisbehaviourIgnoresNonHeaderMessage(t *testing.T) {
	message := tmclient.NewMisbehaviour("07-tendermint-0", nil, nil)
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, message)

	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), nil, "07-tendermint-0", messageBytes, nil,
	)

	require.NoError(t, err)
	require.Nil(t, misbehaviour)
}

func TestCheckForMisbehaviourUsesCachedHeader(t *testing.T) {
	cachedHeader := newMatcherTendermintIBCHeader(matcherTimestamp(), 12, []byte("app-hash"))
	proposedHeader := matcherTMHeader(t, cachedHeader)
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, proposedHeader)
	counterparty := &matcherProvider{ibcHeaderErr: errors.New("must not query counterparty")}

	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), counterparty, "07-tendermint-0", messageBytes, cachedHeader,
	)

	require.NoError(t, err)
	require.Nil(t, misbehaviour)
	require.Empty(t, counterparty.calls)
}

func TestCheckForMisbehaviourQueriesHeaderAtProposedHeight(t *testing.T) {
	counterpartyHeader := newMatcherTendermintIBCHeader(matcherTimestamp(), 12, []byte("app-hash"))
	proposedHeader := matcherTMHeader(t, counterpartyHeader)
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, proposedHeader)
	counterparty := &matcherProvider{ibcHeader: counterpartyHeader}

	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), counterparty, "07-tendermint-0", messageBytes, nil,
	)

	require.NoError(t, err)
	require.Nil(t, misbehaviour)
	require.Equal(t, []string{"ibc-header"}, counterparty.calls)
	require.Equal(t, proposedHeader.Header.Height, counterparty.ibcHeaderHeight)
}

func TestCheckForMisbehaviourPropagatesHeaderQueryError(t *testing.T) {
	wantErr := errors.New("header unavailable")
	proposedHeader := matcherTMHeader(t, newMatcherTendermintIBCHeader(matcherTimestamp(), 12, []byte("app-hash")))
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, proposedHeader)
	counterparty := &matcherProvider{ibcHeaderErr: wantErr}

	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), counterparty, "07-tendermint-0", messageBytes, nil,
	)

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, misbehaviour)
}

func TestCheckForMisbehaviourRejectsUnexpectedQueriedHeader(t *testing.T) {
	proposedHeader := matcherTMHeader(t, newMatcherTendermintIBCHeader(matcherTimestamp(), 12, []byte("app-hash")))
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, proposedHeader)
	unexpectedHeader := matcherIBCHeader{}
	counterparty := &matcherProvider{ibcHeader: unexpectedHeader}

	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), counterparty, "07-tendermint-0", messageBytes, nil,
	)

	wantErr := fmt.Sprintf(
		"failed to check for misbehaviour, expected %T, got %T",
		(*TendermintIBCHeader)(nil), unexpectedHeader,
	)
	require.EqualError(t, err, wantErr)
	require.Nil(t, misbehaviour)
}

func TestCheckForMisbehaviourPreservesUnexpectedCachedHeaderPanic(t *testing.T) {
	proposedHeader := matcherTMHeader(t, newMatcherTendermintIBCHeader(matcherTimestamp(), 12, []byte("app-hash")))
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, proposedHeader)

	require.Panics(t, func() {
		_, _ = CheckForMisbehaviour(
			context.Background(), nil, "07-tendermint-0", messageBytes, matcherIBCHeader{},
		)
	})
}

func TestCheckForMisbehaviourCopiesTrustedFields(t *testing.T) {
	proposedIBCHeader := newMatcherTendermintIBCHeader(matcherTimestamp(), 12, []byte("proposed-app-hash"))
	proposedHeader := matcherTMHeader(t, proposedIBCHeader)
	proposedHeader.TrustedHeight = clienttypes.NewHeight(1, 9)
	messageBytes := clienttypes.MustMarshalClientMessage(tendermintClientCodec, proposedHeader)
	cachedHeader := newMatcherTendermintIBCHeader(
		proposedIBCHeader.SignedHeader.Time.Add(-time.Second), 12, []byte("trusted-app-hash"),
	)

	misbehaviour, err := CheckForMisbehaviour(
		context.Background(), nil, "07-tendermint-0", messageBytes, cachedHeader,
	)

	require.NoError(t, err)
	tmMisbehaviour, ok := misbehaviour.(*tmclient.Misbehaviour)
	require.True(t, ok)
	require.Equal(t, "07-tendermint-0", tmMisbehaviour.ClientId)
	require.Equal(t, proposedHeader.TrustedHeight, tmMisbehaviour.Header2.TrustedHeight)
	require.Equal(t, proposedHeader.TrustedValidators, tmMisbehaviour.Header2.TrustedValidators)
}

func matchingClientStates(trustingPeriod time.Duration) (*tmclient.ClientState, *tmclient.ClientState) {
	existingClient := &tmclient.ClientState{
		ChainId:        "chain-1",
		TrustingPeriod: trustingPeriod,
		LatestHeight:   clienttypes.NewHeight(1, 7),
	}
	newClient := *existingClient
	newClient.LatestHeight = clienttypes.NewHeight(1, 8)
	return existingClient, &newClient
}

func matcherConsensusResponse(
	t *testing.T,
	consensusState ibcexported.ConsensusState,
) *clienttypes.QueryConsensusStateResponse {
	t.Helper()
	packedState, err := codectypes.NewAnyWithValue(consensusState)
	require.NoError(t, err)
	return &clienttypes.QueryConsensusStateResponse{ConsensusState: packedState}
}

func newMatcherTendermintIBCHeader(
	timestamp time.Time,
	height int64,
	appHash []byte,
) TendermintIBCHeader {
	validatorSet := comettypes.NewValidatorSet(nil)
	return TendermintIBCHeader{
		SignedHeader: &comettypes.SignedHeader{
			Header: &comettypes.Header{
				ChainID:            "chain-1",
				Height:             height,
				Time:               timestamp,
				AppHash:            appHash,
				NextValidatorsHash: []byte("next-validators-hash"),
			},
			Commit: &comettypes.Commit{Height: height},
		},
		ValidatorSet:      validatorSet,
		TrustedValidators: validatorSet,
		TrustedHeight:     clienttypes.NewHeight(1, uint64(height-1)),
	}
}

func matcherTMHeader(t *testing.T, ibcHeader TendermintIBCHeader) *tmclient.Header {
	t.Helper()
	header, err := ibcHeader.TMHeader()
	require.NoError(t, err)
	return header
}

func matcherTimestamp() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}
