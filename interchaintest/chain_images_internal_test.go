package interchaintest

import "github.com/cosmos/interchaintest/v11/ibc"

func gaiaChainConfig(version string, config ibc.ChainConfig) ibc.ChainConfig {
	config.Images = []ibc.DockerImage{ibc.NewDockerImage("ghcr.io/strangelove-ventures/heighliner/gaia", version, "1025:1025")}
	return config
}
