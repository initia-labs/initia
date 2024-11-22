package flags

import (
	"github.com/spf13/cobra"

	cfg "github.com/cometbft/cometbft/config"
)

var (
	defaultCometConfig = cfg.DefaultConfig()
)

// AddCometBFTFlags adds flags for cometbft to the provided command
func AddCometBFTFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("statesync.enable", defaultCometConfig.StateSync.Enable, "enable state sync")
	cmd.Flags().StringArray("statesync.rpc_servers", defaultCometConfig.StateSync.RPCServers, "state sync rpc servers")
	cmd.Flags().Int64("statesync.trust_height", defaultCometConfig.StateSync.TrustHeight, "state sync trust height")
	cmd.Flags().String("statesync.trust_hash", defaultCometConfig.StateSync.TrustHash, "state sync trust hash")
	cmd.Flags().Duration("statesync.trust_period", defaultCometConfig.StateSync.TrustPeriod, "state sync trust period")
	cmd.Flags().Duration("statesync.discovery_time", defaultCometConfig.StateSync.DiscoveryTime, "state sync discovery time")
	cmd.Flags().String("statesync.temp_dir", defaultCometConfig.StateSync.TempDir, "state sync temp dir")
	cmd.Flags().Duration("statesync.chunk_request_timeout", defaultCometConfig.StateSync.ChunkRequestTimeout, "state sync chunk request timeout")
	cmd.Flags().Int32("statesync.chunk_fetchers", defaultCometConfig.StateSync.ChunkFetchers, "state sync chunk fetchers")
}
