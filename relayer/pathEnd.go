package relayer

import (
	"fmt"
	"strings"

	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
)

// MerklePrefix is an ordered list of UTF-8 commitment key path segments.
type MerklePrefix []string

// PathEnd represents the local connection identifiers for a relay path
// The path is set on the chain before performing operations
type PathEnd struct {
	ChainID      string       `yaml:"chain-id,omitempty" json:"chain-id,omitempty"`
	ClientID     string       `yaml:"client-id,omitempty" json:"client-id,omitempty"`
	ConnectionID string       `yaml:"connection-id,omitempty" json:"connection-id,omitempty"`
	MerklePrefix MerklePrefix `yaml:"merkle-prefix,omitempty" json:"merkle-prefix,omitempty"`
}

// Validate rejects empty Merkle key path segments while preserving their order.
func (p MerklePrefix) Validate() error {
	for i, segment := range p {
		if segment == "" {
			return fmt.Errorf("contains an empty segment at index %d", i)
		}
	}
	return nil
}

// Bytes converts configured UTF-8 segments to independent byte slices.
func (p MerklePrefix) Bytes() [][]byte {
	if p == nil {
		return nil
	}
	out := make([][]byte, len(p))
	for i, segment := range p {
		out[i] = []byte(segment)
	}
	return out
}

// OrderFromString parses a string into a channel order byte
func OrderFromString(order string) chantypes.Order {
	switch strings.ToUpper(order) {
	case "UNORDERED":
		return chantypes.UNORDERED
	case "ORDERED":
		return chantypes.ORDERED
	default:
		return chantypes.NONE
	}
}

// StringFromOrder returns the string representation of a channel order.
func StringFromOrder(order chantypes.Order) string {
	switch order {
	case chantypes.UNORDERED:
		return "unordered"
	case chantypes.ORDERED:
		return "ordered"
	default:
		return ""
	}
}
