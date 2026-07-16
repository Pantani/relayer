package ethermint

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

type traversalChildFixture struct {
	Label string `json:"label"`
}

type traversalFixture struct {
	Name        string                `json:"name"`
	Enabled     bool                  `json:"enabled"`
	Count       int32                 `json:"count"`
	IDs         []uint64              `json:"ids"`
	Matrix      [][]uint8             `json:"matrix"`
	Child       traversalChildFixture `json:"child"`
	Empty       string                `json:"empty,omitempty"`
	Unsupported map[string]string     `json:"unsupported"`
}

func TestWalkFieldsGolden(t *testing.T) {
	typeMap := apitypes.Types{"Root": {}}
	input := traversalFixture{
		Name:    "relayer",
		Enabled: true,
		Count:   7,
		IDs:     []uint64{10, 20},
		Matrix:  [][]uint8{{1, 2}, {3}},
		Child:   traversalChildFixture{Label: "nested"},
		Unsupported: map[string]string{
			"ignored": "map",
		},
	}

	require.NoError(t, walkFields(nil, typeMap, "Root", input))
	require.Equal(t, apitypes.Types{
		"Root": {
			{Name: "name", Type: "string"},
			{Name: "enabled", Type: "bool"},
			{Name: "count", Type: "int32"},
			{Name: "ids", Type: "uint64[]"},
			{Name: "matrix", Type: "uint8[]"},
			{Name: "child", Type: "TypeChild"},
		},
		"TypeChild": {
			{Name: "label", Type: "string"},
		},
	}, typeMap)
}

func TestWalkFieldsGoldenPointerField(t *testing.T) {
	type pointerFixture struct {
		Child *traversalChildFixture `json:"child"`
	}

	typeMap := apitypes.Types{"Root": {}}
	require.NoError(t, walkFields(nil, typeMap, "Root", pointerFixture{
		Child: &traversalChildFixture{Label: "nested"},
	}))
	require.Equal(t, apitypes.Types{
		"Root":      {{Name: "child", Type: "TypeChild"}},
		"TypeChild": {{Name: "label", Type: "string"}},
	}, typeMap)
}

func TestWalkFieldsReturnsEarlyForCompleteDefinition(t *testing.T) {
	type inputFixture struct {
		Name string `json:"name"`
	}

	existing := []apitypes.Type{{Name: "existing", Type: "bool"}}
	typeMap := apitypes.Types{"Root": existing}

	require.NoError(t, walkFields(nil, typeMap, "Root", inputFixture{Name: "ignored"}))
	require.Equal(t, existing, typeMap["Root"])
}

func TestTypToEth(t *testing.T) {
	tests := map[string]struct {
		typ  reflect.Type
		want string
	}{
		"string":             {typ: reflect.TypeOf(""), want: "string"},
		"bool":               {typ: reflect.TypeOf(false), want: "bool"},
		"int":                {typ: reflect.TypeOf(int(0)), want: "int64"},
		"int8":               {typ: reflect.TypeOf(int8(0)), want: "int8"},
		"int16":              {typ: reflect.TypeOf(int16(0)), want: "int16"},
		"int32":              {typ: reflect.TypeOf(int32(0)), want: "int32"},
		"int64":              {typ: reflect.TypeOf(int64(0)), want: "int64"},
		"uint":               {typ: reflect.TypeOf(uint(0)), want: "uint64"},
		"uint8":              {typ: reflect.TypeOf(uint8(0)), want: "uint8"},
		"uint16":             {typ: reflect.TypeOf(uint16(0)), want: "uint16"},
		"uint32":             {typ: reflect.TypeOf(uint32(0)), want: "uint32"},
		"uint64":             {typ: reflect.TypeOf(uint64(0)), want: "uint64"},
		"slice":              {typ: reflect.TypeOf([]uint64{}), want: "uint64[]"},
		"array":              {typ: reflect.TypeOf([2]bool{}), want: "bool[]"},
		"nested slice":       {typ: reflect.TypeOf([][]uint8{}), want: "uint8[][]"},
		"hash":               {typ: reflect.TypeOf(common.Hash{}), want: "uint8[]"},
		"address":            {typ: reflect.TypeOf(common.Address{}), want: "uint8[]"},
		"big int":            {typ: reflect.TypeOf(big.Int{}), want: "string"},
		"big int pointer":    {typ: reflect.TypeOf(&big.Int{}), want: "string"},
		"time":               {typ: reflect.TypeOf(time.Time{}), want: "string"},
		"time pointer":       {typ: reflect.TypeOf(&time.Time{}), want: "string"},
		"cosmos int":         {typ: reflect.TypeOf(sdkmath.Int{}), want: "string"},
		"cosmos decimal":     {typ: reflect.TypeOf(sdkmath.LegacyDec{}), want: "string"},
		"ed25519 public key": {typ: reflect.TypeOf(ed25519.PubKey{}), want: "string"},
		"unsupported map":    {typ: reflect.TypeOf(map[string]string{}), want: ""},
		"unsupported struct": {typ: reflect.TypeOf(struct{ Value string }{}), want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, typToEth(tc.typ))
		})
	}
}
