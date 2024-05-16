package interchaintest

import (
	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	ibctypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"

	"os"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

var (
	InitialICTestRepo = "ghcr.io/initia/initia-test"
	repo, version     = GetDockerImageInfo()

	InitialImage = ibc.DockerImage{
		Repository: repo,
		Version:    version,
		UidGid:     "1025:1025",
	}

	InitiaConfig = ibc.ChainConfig{
		Type:                "cosmos",
		Name:                "initial",
		ChainID:             "initiation-1",
		Images:              []ibc.DockerImage{InitialImage},
		Bin:                 "initiad",
		Bech32Prefix:        "init",
		Denom:               "uinit",
		CoinType:            "118",
		GasPrices:           "0.0uinit",
		GasAdjustment:       1.1,
		TrustingPeriod:      "112h",
		EncodingConfig:      initialEncoding(),
		NoHostMount:         false,
		ModifyGenesis:       nil,
		ConfigFileOverrides: nil,
	}

	genesisWalletAmount = sdkmath.NewInt(10_000_000)

	DefaultRelayer = ibc.DockerImage{
		Repository: "ghcr.io/cosmos/relayer",
		Version:    "main",
		UidGid:     "1025:1025",
	}
)

// GetDockerImageInfo returns the appropriate repo and branch version string for integration with the CI pipeline.
// The remote runner sets the BRANCH_CI env var. If present, interchaintest will use the docker image pushed up to the repo.
// If testing locally, user should run `make docker-build-debug` and interchaintest will use the local image.
func GetDockerImageInfo() (repo, version string) {
	branchVersion, found := os.LookupEnv("BRANCH_CI")
	repo = InitialICTestRepo
	if !found {
		// make local-image
		repo = "initial"
		branchVersion = "debug"
	}
	return repo, branchVersion
}

func initialEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	wasmtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	ibctypes.RegisterInterfaces(cfg.InterfaceRegistry)

	// register custom types
	// authtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	return &cfg
}
