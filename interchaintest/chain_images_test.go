package interchaintest_test

import "github.com/cosmos/interchaintest/v11/ibc"

const gaiaHeighlinerRepository = "ghcr.io/strangelove-ventures/heighliner/gaia"

func gaiaChainConfig(version string, config ibc.ChainConfig) ibc.ChainConfig {
	config.Images = []ibc.DockerImage{ibc.NewDockerImage(gaiaHeighlinerRepository, version, "1025:1025")}
	return config
}
