package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
)

// GetCmdQueryClassTrace defines the command to query a a class id trace from a given trace hash or ibc denom.
func GetCmdQueryClassTrace() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "class-id-trace [hash/class-id]",
		Short:   "Query the class-id trace info from a given trace hash or ibc class-id",
		Long:    "Query the class-id trace info from a given trace hash or ibc class-id",
		Example: fmt.Sprintf("%s query ibc-nft-transfer class-id-trace 27A6394C3F9FF9C9DCF5DFFADF9BB5FE9A37C7E92B006199894CF1824DF9AC7C", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryClassTraceRequest{
				Hash: args[0],
			}

			res, err := queryClient.ClassTrace(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryClassTraces defines the command to query all the class id trace infos
// that this chain maintains.
func GetCmdQueryClassTraces() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "class-id-traces",
		Short:   "Query the trace info for all class ids",
		Long:    "Query the trace info for all class ids",
		Example: fmt.Sprintf("%s query ibc-nft-transfer class-id-traces", version.AppName),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &types.QueryClassTracesRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.ClassTraces(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "class id trace")

	return cmd
}

// GetCmdParams returns the command handler for ibc-nft-transfer parameter querying.
func GetCmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "params",
		Short:   "Query the current ibc-nft-transfer parameters",
		Long:    "Query the current ibc-nft-transfer parameters",
		Args:    cobra.NoArgs,
		Example: fmt.Sprintf("%s query ibc-nft-transfer params", version.AppName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdParams returns the command handler for ibc-nft-transfer parameter querying.
func GetCmdQueryEscrowAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "escrow-address",
		Short:   "Get the escrow address for a channel",
		Long:    "Get the escrow address for a channel",
		Args:    cobra.ExactArgs(2),
		Example: fmt.Sprintf("%s query ibc-nft-transfer escrow-address [port] [channel-id]", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			port := args[0]
			channel := args[1]
			addr := types.GetEscrowAddress(port, channel)
			return clientCtx.PrintString(fmt.Sprintf("%s\n", addr.String()))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryClassHash defines the command to query a class id hash from a given trace.
func GetCmdQueryClassHash() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "class-id-hash [trace]",
		Short:   "Query the class-id hash info from a given class-id trace",
		Long:    "Query the class-id hash info from a given class-id trace",
		Example: fmt.Sprintf("%s query ibc-nft-transfer class-id-hash nft-transfer/channel-0/0x123::nft::Extension", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryClassHashRequest{
				Trace: args[0],
			}

			res, err := queryClient.ClassHash(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
