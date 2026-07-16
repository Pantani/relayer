package cosmos

import (
	"context"
	"errors"
	"testing"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/relayer/v2/cclient"
	"github.com/stretchr/testify/require"
)

var (
	coinType = 118

	testCfg = CosmosProviderConfig{
		KeyDirectory:     "",
		Key:              "default",
		ChainName:        "osmosis",
		ChainID:          "osmosis-1",
		RPCAddr:          "https://osmosis-rpc.polkachu.com:443",
		AccountPrefix:    "osmo",
		KeyringBackend:   "test",
		DynamicGasPrice:  true,
		GasAdjustment:    1.2,
		GasPrices:        "0.0025uosmo",
		MinGasAmount:     1,
		MaxGasAmount:     0,
		Debug:            false,
		Timeout:          "30s",
		BlockTimeout:     "30s",
		OutputFormat:     "json",
		SignModeStr:      "direct",
		ExtraCodecs:      nil,
		Modules:          nil,
		Slip44:           &coinType,
		SigningAlgorithm: "",
		Broadcast:        "batch",
		MinLoopDuration:  0,
		ExtensionOptions: nil,
		FeeGrants:        nil,
	}
)

type baseFeeConsensusClient struct {
	cclient.ConsensusClient
	response  *cclient.ABCIQueryResponse
	err       error
	queryPath string
}

func (c *baseFeeConsensusClient) GetABCIQuery(
	_ context.Context,
	queryPath string,
	_ tmbytes.HexBytes,
) (*cclient.ABCIQueryResponse, error) {
	c.queryPath = queryPath
	return c.response, c.err
}

func TestQueryBaseFee(t *testing.T) {
	// This wire fixture intentionally has a 17-byte field. The old implementation stripped only
	// a 0x10 length byte and therefore broke as soon as the encoded decimal grew past 16 bytes.
	fixture := append([]byte{0x0a, 0x11}, []byte("12500000000000000")...)

	consensusClient := &baseFeeConsensusClient{
		response: &cclient.ABCIQueryResponse{Value: fixture},
	}
	provider := CosmosProvider{
		PCfg:            testCfg,
		ConsensusClient: consensusClient,
	}

	baseFee, err := provider.QueryBaseFee(context.Background())
	require.NoError(t, err)
	require.Equal(t, "0.012500000000000000uosmo", baseFee)
	require.Equal(t, baseFeeQueryPath, consensusClient.queryPath)
}

func TestQueryBaseFeeErrors(t *testing.T) {
	validFixture := append([]byte{0x0a, 0x11}, []byte("12500000000000000")...)

	invalidFeeFixture, err := proto.Marshal(&queryEIPBaseFeeResponse{BaseFee: "not-a-decimal"})
	require.NoError(t, err)
	emptyFeeFixture, err := proto.Marshal(&queryEIPBaseFeeResponse{})
	require.NoError(t, err)

	tests := []struct {
		name      string
		response  *cclient.ABCIQueryResponse
		queryErr  error
		gasPrices string
		wantError string
	}{
		{name: "transport", queryErr: errors.New("RPC unavailable"), wantError: "RPC unavailable"},
		{name: "empty response", wantError: "empty ABCI response"},
		{name: "ABCI failure", response: &cclient.ABCIQueryResponse{Code: 7}, wantError: "ABCI code 7"},
		{name: "invalid protobuf", response: &cclient.ABCIQueryResponse{Value: []byte{0xff}}, wantError: "decode Osmosis"},
		{name: "missing fee", response: &cclient.ABCIQueryResponse{Value: emptyFeeFixture}, wantError: "missing base_fee"},
		{name: "invalid fee", response: &cclient.ABCIQueryResponse{Value: invalidFeeFixture}, wantError: "base fee value"},
		{name: "invalid denom", response: &cclient.ABCIQueryResponse{Value: validFixture}, gasPrices: "uosmo", wantError: "token denom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testCfg
			if tt.gasPrices != "" {
				config.GasPrices = tt.gasPrices
			}
			provider := CosmosProvider{
				PCfg: config,
				ConsensusClient: &baseFeeConsensusClient{
					response: tt.response,
					err:      tt.queryErr,
				},
			}

			baseFee, err := provider.QueryBaseFee(context.Background())
			require.ErrorContains(t, err, tt.wantError)
			require.Empty(t, baseFee)
		})
	}
}

func TestParseDenom(t *testing.T) {
	tests := []struct {
		name     string
		gasPrice string
		want     string
		wantErr  bool
	}{
		{name: "osmosis", gasPrice: "0.0025uosmo", want: "uosmo"},
		{name: "letters are preserved", gasPrice: "0.1uOSMO", want: "uOSMO"},
		{name: "missing amount", gasPrice: "uosmo", wantErr: true},
		{name: "missing denom", gasPrice: "0.0025", wantErr: true},
		{name: "multiple prices", gasPrice: "0.0025uosmo,0.1uatom", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			denom, err := parseTokenDenom(tt.gasPrice)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, denom)
		})
	}
}
