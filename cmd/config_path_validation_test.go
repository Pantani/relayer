package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type pathValidationProvider struct {
	provider.ChainProvider
	chainID           string
	height            int64
	heightErr         error
	clientErr         error
	connectionErr     error
	connectionClient  string
	heightQueries     int
	clientQueries     int
	connectionQueries int
}

func (p *pathValidationProvider) ChainId() string {
	return p.chainID
}

func (p *pathValidationProvider) QueryLatestHeight(context.Context) (int64, error) {
	p.heightQueries++
	return p.height, p.heightErr
}

func (p *pathValidationProvider) QueryClientStateResponse(context.Context, int64, string) (*clienttypes.QueryClientStateResponse, error) {
	p.clientQueries++
	return &clienttypes.QueryClientStateResponse{}, p.clientErr
}

func (p *pathValidationProvider) QueryConnection(context.Context, int64, string) (*conntypes.QueryConnectionResponse, error) {
	p.connectionQueries++
	return &conntypes.QueryConnectionResponse{
		Connection: &conntypes.ConnectionEnd{ClientId: p.connectionClient},
	}, p.connectionErr
}

func TestValidatePathEndMissingChainIsWarning(t *testing.T) {
	t.Parallel()

	config := Config{Chains: make(relayer.Chains)}
	var stderr bytes.Buffer
	err := config.ValidatePathEnd(context.Background(), &stderr, &relayer.PathEnd{ChainID: "missing-1"})

	require.NoError(t, err)
	require.Equal(t, "Chain missing-1 is not currently configured.\n", stderr.String())
}

func TestValidatePathEndSkipsEmptyIdentifiers(t *testing.T) {
	t.Parallel()

	config, mock := validationConfig()
	err := config.ValidatePathEnd(context.Background(), &bytes.Buffer{}, &relayer.PathEnd{ChainID: mock.chainID})

	require.NoError(t, err)
	require.Zero(t, mock.heightQueries)
	require.Zero(t, mock.clientQueries)
	require.Zero(t, mock.connectionQueries)
}

func TestValidatePathEndValidatesClientAndConnection(t *testing.T) {
	t.Parallel()

	config, mock := validationConfig()
	mock.connectionClient = "07-tendermint-0"
	pathEnd := &relayer.PathEnd{
		ChainID:      mock.chainID,
		ClientID:     "07-tendermint-0",
		ConnectionID: "connection-0",
	}
	err := config.ValidatePathEnd(context.Background(), &bytes.Buffer{}, pathEnd)

	require.NoError(t, err)
	require.Equal(t, 1, mock.heightQueries)
	require.Equal(t, 1, mock.clientQueries)
	require.Equal(t, 1, mock.connectionQueries)
}

func TestValidatePathEndRequiresClientForConnection(t *testing.T) {
	t.Parallel()

	config, mock := validationConfig()
	pathEnd := &relayer.PathEnd{ChainID: mock.chainID, ConnectionID: "connection-7"}
	err := config.ValidatePathEnd(context.Background(), &bytes.Buffer{}, pathEnd)

	require.EqualError(t, err, "clientID is not configured for the connection: connection-7")
	require.Equal(t, 1, mock.heightQueries)
	require.Zero(t, mock.clientQueries)
	require.Zero(t, mock.connectionQueries)
}

func TestValidatePathEndReturnsProviderErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("client query failed")
	config, mock := validationConfig()
	mock.clientErr = wantErr
	pathEnd := &relayer.PathEnd{ChainID: mock.chainID, ClientID: "07-tendermint-0"}
	err := config.ValidatePathEnd(context.Background(), &bytes.Buffer{}, pathEnd)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 1, mock.heightQueries)
	require.Equal(t, 1, mock.clientQueries)
	require.Zero(t, mock.connectionQueries)
}

func validationConfig() (Config, *pathValidationProvider) {
	mock := &pathValidationProvider{chainID: "chain-a-1", height: 42}
	chain := relayer.NewChain(zap.NewNop(), mock, false)
	return Config{Chains: relayer.Chains{"chain-a": chain}}, mock
}
