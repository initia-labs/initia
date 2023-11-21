package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// GetCmdQueryChannelRelayer defines the command to query a a channel relayer with channel id.
func GetCmdQueryChannelRelayer() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel-relayer [channel-id]",
		Short:   "Query the permissioned channel relayer of the channel id",
		Long:    "Query the permissioned channel relayer of the channel id",
		Example: fmt.Sprintf("%s query ibc-perm channel-relayer channel-123", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryChannelRelayerRequest{
				Channel: args[0],
			}

			res, err := queryClient.ChannelRelayer(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
