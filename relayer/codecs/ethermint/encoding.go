package ethermint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// WrapTxToTypedData is an ultimate method that wraps Amino-encoded Cosmos Tx JSON data
// into an EIP712-compatible TypedData request.
func WrapTxToTypedData(
	cdc codectypes.AnyUnpacker,
	chainID uint64,
	msg sdk.Msg,
	data []byte,
	feeDelegation *FeeDelegationOptions,
) (apitypes.TypedData, error) {
	txData := make(map[string]interface{})

	if err := json.Unmarshal(data, &txData); err != nil {
		return apitypes.TypedData{}, errorsmod.Wrap(legacyerrors.ErrJSONUnmarshal, "failed to JSON unmarshal data")
	}

	domain := apitypes.TypedDataDomain{
		Name:              "Cosmos Web3",
		Version:           "1.0.0",
		ChainId:           math.NewHexOrDecimal256(int64(chainID)),
		VerifyingContract: "cosmos",
		Salt:              "0",
	}

	msgTypes, err := extractMsgTypes(cdc, "MsgValue", msg)
	if err != nil {
		return apitypes.TypedData{}, err
	}

	if feeDelegation != nil {
		feeInfo, ok := txData["fee"].(map[string]interface{})
		if !ok {
			return apitypes.TypedData{}, errorsmod.Wrap(legacyerrors.ErrInvalidType, "cannot parse fee from tx data")
		}

		feeInfo["feePayer"] = feeDelegation.FeePayer.String()

		// also patching msgTypes to include feePayer
		msgTypes["Fee"] = []apitypes.Type{
			{Name: "feePayer", Type: "string"},
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		}
	}

	typedData := apitypes.TypedData{
		Types:       msgTypes,
		PrimaryType: "Tx",
		Domain:      domain,
		Message:     txData,
	}

	return typedData, nil
}

type FeeDelegationOptions struct {
	FeePayer sdk.AccAddress
}

func extractMsgTypes(cdc codectypes.AnyUnpacker, msgTypeName string, msg sdk.Msg) (apitypes.Types, error) {
	rootTypes := apitypes.Types{
		"EIP712Domain": {
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "version",
				Type: "string",
			},
			{
				Name: "chainId",
				Type: "uint256",
			},
			{
				Name: "verifyingContract",
				Type: "string",
			},
			{
				Name: "salt",
				Type: "string",
			},
		},
		"Tx": {
			{Name: "account_number", Type: "string"},
			{Name: "chain_id", Type: "string"},
			{Name: "fee", Type: "Fee"},
			{Name: "memo", Type: "string"},
			{Name: "msgs", Type: "Msg[]"},
			{Name: "sequence", Type: "string"},
			// Note timeout_height was removed because it was not getting filled with the legacyTx
			// {Name: "timeout_height", Type: "string"},
		},
		"Fee": {
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		},
		"Coin": {
			{Name: "denom", Type: "string"},
			{Name: "amount", Type: "string"},
		},
		"Msg": {
			{Name: "type", Type: "string"},
			{Name: "value", Type: msgTypeName},
		},
		msgTypeName: {},
	}

	if err := walkFields(cdc, rootTypes, msgTypeName, msg); err != nil {
		return nil, err
	}

	return rootTypes, nil
}

const typeDefPrefix = "_"

func walkFields(cdc codectypes.AnyUnpacker, typeMap apitypes.Types, rootType string, in interface{}) (err error) {
	defer doRecover(&err)

	t := reflect.TypeOf(in)
	v := reflect.ValueOf(in)

	for {
		if t.Kind() == reflect.Ptr ||
			t.Kind() == reflect.Interface {
			t = t.Elem()
			v = v.Elem()

			continue
		}

		break
	}

	return traverseFields(cdc, typeMap, rootType, typeDefPrefix, t, v)
}

type cosmosAnyWrapper struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type traversedField struct {
	name         string
	prefix       string
	typ          reflect.Type
	value        reflect.Value
	isCollection bool
}

func traverseFields(
	cdc codectypes.AnyUnpacker,
	typeMap apitypes.Types,
	rootType string,
	prefix string,
	t reflect.Type,
	v reflect.Value,
) error {
	if typeDefinitionComplete(typeMap, rootType, prefix, t.NumField()) {
		return nil
	}

	for i := 0; i < t.NumField(); i++ {
		if err := traverseField(cdc, typeMap, rootType, prefix, t.Field(i), valueField(v, i)); err != nil {
			return err
		}
	}

	return nil
}

func typeDefinitionComplete(typeMap apitypes.Types, rootType, prefix string, fields int) bool {
	return len(typeMap[typeDefinitionName(rootType, prefix)]) == fields
}

func typeDefinitionName(rootType, prefix string) string {
	if prefix == typeDefPrefix {
		return rootType
	}

	return sanitizeTypedef(prefix)
}

func valueField(v reflect.Value, index int) reflect.Value {
	if v.IsValid() {
		return v.Field(index)
	}

	return reflect.Value{}
}

func traverseField(
	cdc codectypes.AnyUnpacker,
	typeMap apitypes.Types,
	rootType string,
	prefix string,
	structField reflect.StructField,
	value reflect.Value,
) error {
	field, skip, err := prepareTraversedField(cdc, prefix, structField, value)
	if err != nil || skip {
		return err
	}

	if appendEthereumField(typeMap, rootType, prefix, field) {
		return nil
	}

	if field.typ.Kind() != reflect.Struct {
		return nil
	}

	appendTypedField(typeMap, rootType, prefix, field.name, structFieldType(field))
	return traverseFields(cdc, typeMap, rootType, field.prefix, field.typ, field.value)
}

func prepareTraversedField(
	cdc codectypes.AnyUnpacker,
	prefix string,
	structField reflect.StructField,
	value reflect.Value,
) (traversedField, bool, error) {
	fieldType, fieldValue, err := unpackFieldAny(cdc, structField.Type, value)
	if err != nil {
		return traversedField{}, false, err
	}

	if fieldValue.IsZero() {
		return traversedField{}, true, nil
	}

	fieldType, fieldValue = dereferenceField(fieldType, fieldValue)
	return prepareCollectionField(cdc, prefix, jsonNameFromTag(structField.Tag), fieldType, fieldValue)
}

func unpackFieldAny(
	cdc codectypes.AnyUnpacker,
	fieldType reflect.Type,
	fieldValue reflect.Value,
) (reflect.Type, reflect.Value, error) {
	if fieldType != cosmosAnyType {
		return fieldType, fieldValue, nil
	}

	return unpackAny(cdc, fieldValue)
}

func prepareCollectionField(
	cdc codectypes.AnyUnpacker,
	prefix string,
	name string,
	fieldType reflect.Type,
	fieldValue reflect.Value,
) (traversedField, bool, error) {
	isCollection := fieldType.Kind() == reflect.Array || fieldType.Kind() == reflect.Slice
	if isCollection && fieldValue.Len() == 0 {
		return traversedField{}, true, nil
	}

	var err error
	if isCollection {
		fieldType, fieldValue, err = collectionElement(cdc, fieldType, fieldValue)
	}
	if err != nil {
		return traversedField{}, false, err
	}

	fieldType, fieldValue = dereferenceField(fieldType, fieldValue)
	return traversedField{
		name:         name,
		prefix:       fmt.Sprintf("%s.%s", prefix, name),
		typ:          fieldType,
		value:        fieldValue,
		isCollection: isCollection,
	}, false, nil
}

func collectionElement(
	cdc codectypes.AnyUnpacker,
	fieldType reflect.Type,
	fieldValue reflect.Value,
) (reflect.Type, reflect.Value, error) {
	fieldType = fieldType.Elem()
	fieldValue = fieldValue.Index(0)
	return unpackFieldAny(cdc, fieldType, fieldValue)
}

func dereferenceField(fieldType reflect.Type, fieldValue reflect.Value) (reflect.Type, reflect.Value) {
	for {
		switch fieldType.Kind() {
		case reflect.Ptr:
			fieldType = fieldType.Elem()
			fieldValue = dereferenceValue(fieldValue)
			continue
		case reflect.Interface:
			fieldType = reflect.TypeOf(fieldValue.Interface())
			continue
		}

		if fieldValue.Kind() != reflect.Ptr {
			return fieldType, fieldValue
		}

		fieldValue = fieldValue.Elem()
	}
}

func dereferenceValue(value reflect.Value) reflect.Value {
	if value.IsValid() {
		return value.Elem()
	}

	return value
}

func appendEthereumField(typeMap apitypes.Types, rootType, prefix string, field traversedField) bool {
	ethType := ethereumFieldType(field.typ, field.isCollection)
	if ethType == "" {
		return false
	}

	appendTypedField(typeMap, rootType, prefix, field.name, ethType)
	return true
}

func ethereumFieldType(fieldType reflect.Type, isCollection bool) string {
	ethType := typToEth(fieldType)
	if ethType == "" {
		return ""
	}

	if isCollection && fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array {
		return ethType + "[]"
	}

	return ethType
}

func structFieldType(field traversedField) string {
	fieldType := sanitizeTypedef(field.prefix)
	if field.isCollection {
		return fieldType + "[]"
	}

	return fieldType
}

func appendTypedField(typeMap apitypes.Types, rootType, prefix, name, fieldType string) {
	typeName := typeDefinitionName(rootType, prefix)
	typeMap[typeName] = append(typeMap[typeName], apitypes.Type{Name: name, Type: fieldType})
}

func jsonNameFromTag(tag reflect.StructTag) string {
	jsonTags := tag.Get("json")
	parts := strings.Split(jsonTags, ",")
	return parts[0]
}

// Unpack the given Any value with Type/Value deconstruction
func unpackAny(cdc codectypes.AnyUnpacker, field reflect.Value) (reflect.Type, reflect.Value, error) {
	any, ok := field.Interface().(*codectypes.Any)
	if !ok {
		return nil, reflect.Value{}, errorsmod.Wrapf(legacyerrors.ErrPackAny, "%T", field.Interface())
	}

	anyWrapper := &cosmosAnyWrapper{
		Type: any.TypeUrl,
	}

	if err := cdc.UnpackAny(any, &anyWrapper.Value); err != nil {
		return nil, reflect.Value{}, errorsmod.Wrap(err, "failed to unpack Any in msg struct")
	}

	fieldType := reflect.TypeOf(anyWrapper)
	field = reflect.ValueOf(anyWrapper)

	return fieldType, field, nil
}

// _.foo_bar.baz -> TypeFooBarBaz
//
// this is needed for Geth's own signing code which doesn't
// tolerate complex type names
func sanitizeTypedef(str string) string {
	buf := new(bytes.Buffer)
	parts := strings.Split(str, ".")
	caser := cases.Title(language.English, cases.NoLower)

	for _, part := range parts {
		if part == "_" {
			buf.WriteString("Type")
			continue
		}

		subparts := strings.Split(part, "_")
		for _, subpart := range subparts {
			buf.WriteString(caser.String(subpart))
		}
	}

	return buf.String()
}

var (
	hashType      = reflect.TypeOf(common.Hash{})
	addressType   = reflect.TypeOf(common.Address{})
	bigIntType    = reflect.TypeOf(big.Int{})
	cosmIntType   = reflect.TypeOf(sdkmath.Int{})
	cosmDecType   = reflect.TypeOf(sdkmath.LegacyDec{})
	cosmosAnyType = reflect.TypeOf(&codectypes.Any{})
	timeType      = reflect.TypeOf(time.Time{})

	edType = reflect.TypeOf(ed25519.PubKey{})
)

var basicEthereumTypes = map[reflect.Kind]string{
	reflect.String: "string",
	reflect.Bool:   "bool",
	reflect.Int:    "int64",
	reflect.Int8:   "int8",
	reflect.Int16:  "int16",
	reflect.Int32:  "int32",
	reflect.Int64:  "int64",
	reflect.Uint:   "uint64",
	reflect.Uint8:  "uint8",
	reflect.Uint16: "uint16",
	reflect.Uint32: "uint32",
	reflect.Uint64: "uint64",
}

var ethereumStringTypes = []reflect.Type{
	hashType,
	addressType,
	bigIntType,
	edType,
	timeType,
	cosmDecType,
	cosmIntType,
}

var ethereumPointerStringTypes = []reflect.Type{
	bigIntType,
	edType,
	timeType,
	cosmDecType,
	cosmIntType,
}

// typToEth supports only basic types and arrays of basic types.
// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-712.md
func typToEth(typ reflect.Type) string {
	if ethType := basicEthereumTypes[typ.Kind()]; ethType != "" {
		return ethType
	}

	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		return ethereumCollectionType(typ)
	case reflect.Ptr:
		return ethereumConvertibleType(typ.Elem(), ethereumPointerStringTypes)
	case reflect.Struct:
		return ethereumConvertibleType(typ, ethereumStringTypes)
	}

	return ""
}

func ethereumCollectionType(typ reflect.Type) string {
	elementType := typToEth(typ.Elem())
	if elementType == "" {
		return ""
	}

	return elementType + "[]"
}

func ethereumConvertibleType(typ reflect.Type, supported []reflect.Type) string {
	for _, candidate := range supported {
		if typ.ConvertibleTo(candidate) {
			return "string"
		}
	}

	return ""
}

func doRecover(err *error) {
	if r := recover(); r != nil {
		if e, ok := r.(error); ok {
			e = errorsmod.Wrap(e, "panicked with error")
			*err = e
			return
		}

		*err = fmt.Errorf("%v", r)
	}
}
