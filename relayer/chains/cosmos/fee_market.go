package cosmos

import (
	"context"
	"fmt"
	"regexp"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/zap"
)

const baseFeeQueryPath = "/osmosis.txfees.v1beta1.Query/GetEipBaseFee"

var gasPriceDenomPattern = regexp.MustCompile(`^0\.\d+([a-zA-Z]+)$`)

// queryEIPBaseFeeResponse mirrors the stable wire contract of the Osmosis txfees query without
// importing the full Osmosis application module into the relayer. The custom LegacyDec field is
// encoded as its scaled integer text, so it is decoded into a string before LegacyDec.Unmarshal.
type queryEIPBaseFeeResponse struct {
	BaseFee string `protobuf:"bytes,1,opt,name=base_fee,json=baseFee,proto3" json:"base_fee,omitempty"`
}

func (r *queryEIPBaseFeeResponse) Reset()         { *r = queryEIPBaseFeeResponse{} }
func (r *queryEIPBaseFeeResponse) String() string { return proto.CompactTextString(r) }
func (*queryEIPBaseFeeResponse) ProtoMessage()    {}

// DynamicFee queries the dynamic gas price base fee and returns a string with the base fee and token denom concatenated.
// If the chain does not have dynamic fees enabled in the config, nothing happens and an empty string is always returned.
func (cc *CosmosProvider) DynamicFee(ctx context.Context) string {
	if !cc.PCfg.DynamicGasPrice {
		return ""
	}

	dynamicFee, err := cc.QueryBaseFee(ctx)
	if err != nil {
		// If there was an error querying the dynamic base fee, do nothing and fall back to configured gas price.
		cc.log.Warn("Failed to query the dynamic gas price base fee", zap.Error(err))
		return ""
	}

	return dynamicFee
}

// QueryBaseFee attempts to make an ABCI query to retrieve the base fee on chains using the Osmosis EIP-1559 implementation.
// This is currently hardcoded to only work on Osmosis.
func (cc *CosmosProvider) QueryBaseFee(ctx context.Context) (string, error) {
	resp, err := cc.ConsensusClient.GetABCIQuery(ctx, baseFeeQueryPath, nil)
	if err != nil {
		return "", fmt.Errorf("query Osmosis EIP-1559 base fee: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("query Osmosis EIP-1559 base fee: empty ABCI response")
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("query Osmosis EIP-1559 base fee: ABCI code %d", resp.Code)
	}

	var queryResponse queryEIPBaseFeeResponse
	if err := proto.Unmarshal(resp.Value, &queryResponse); err != nil {
		return "", fmt.Errorf("decode Osmosis EIP-1559 base fee response: %w", err)
	}
	if queryResponse.BaseFee == "" {
		return "", fmt.Errorf("decode Osmosis EIP-1559 base fee response: missing base_fee")
	}

	var fee sdkmath.LegacyDec
	if err := fee.Unmarshal([]byte(queryResponse.BaseFee)); err != nil {
		return "", fmt.Errorf("decode Osmosis EIP-1559 base fee value: %w", err)
	}

	denom, err := parseTokenDenom(cc.PCfg.GasPrices)
	if err != nil {
		return "", err
	}

	return fee.String() + denom, nil
}

// parseTokenDenom takes a string in the format numericGasPrice + tokenDenom (e.g. 0.0025uosmo),
// and parses the tokenDenom portion (e.g. uosmo) before returning just the token denom.
func parseTokenDenom(gasPrice string) (string, error) {
	matches := gasPriceDenomPattern.FindStringSubmatch(gasPrice)

	if len(matches) != 2 {
		return "", fmt.Errorf("failed to parse token denom from string %s", gasPrice)
	}

	return matches[1], nil
}
