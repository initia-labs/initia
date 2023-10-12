package cli

import (
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for IBC connections
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "ibc-perm",
		Short:                      "IBC permissioned channel relayer query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdQueryChannelRelayer(),
	)

	return queryCmd
}
