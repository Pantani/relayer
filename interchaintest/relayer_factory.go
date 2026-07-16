package interchaintest

import (
	"testing"

	"github.com/cosmos/interchaintest/v11"
	"github.com/cosmos/interchaintest/v11/ibc"
	interchaintestrelayer "github.com/cosmos/interchaintest/v11/relayer"
	"github.com/moby/moby/client"
)

// RelayerFactory implements the interchaintest RelayerFactory interface.
type RelayerFactory struct {
	config RelayerConfig
}

// RelayerConfig defines parameters for customizing a LocalRelayer.
type RelayerConfig struct {
	Processor           string
	Memo                string
	InitialBlockHistory uint64
}

func NewRelayerFactory(config RelayerConfig) RelayerFactory {
	return RelayerFactory{
		config: config,
	}
}

// Build returns a relayer interface
func (rf RelayerFactory) Build(interchaintest.TestName, *client.Client, string) ibc.Relayer {
	tst := &testing.T{}
	return NewRelayer(tst, rf.config)
}

func (RelayerFactory) Capabilities() map[interchaintestrelayer.Capability]bool {
	// It is currently expected that the main branch of the relayer supports all tested features.
	return interchaintestrelayer.FullCapabilities()
}

func (RelayerFactory) Name() string { return "github.com/cosmos/relayer" }
