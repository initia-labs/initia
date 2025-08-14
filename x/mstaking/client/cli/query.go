package cli

import (
	"fmt"
	"strings"

	"cosmossdk.io/core/address"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/initia-labs/initia/x/mstaking/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(ac, vc address.Codec) *cobra.Command {
	stakingQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the multi-staking module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	stakingQueryCmd.AddCommand(
		GetCmdQueryDelegation(ac, vc),
		GetCmdQueryDelegations(ac),
		GetCmdQueryUnbondingDelegation(ac, vc),
		GetCmdQueryUnbondingDelegations(ac),
		GetCmdQueryRedelegation(ac, vc),
		GetCmdQueryRedelegations(ac),
		GetCmdQueryValidator(vc),
		GetCmdQueryValidators(),
		GetCmdQueryValidatorDelegations(vc),
		GetCmdQueryValidatorUnbondingDelegations(vc),
		GetCmdQueryValidatorRedelegations(vc),
		GetCmdQueryParams(),
		GetCmdQueryPool(),
		GetCmdQueryMigration(),
	)

	return stakingQueryCmd
}

// GetCmdQueryValidator implements the validator query command.
func GetCmdQueryValidator(vc address.Codec) *cobra.Command {
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "validator [validator-addr]",
		Short: "Query a validator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query details about an individual validator.

Example:
$ %s query staking validator %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			valAddrStr := args[0]
			if _, err := vc.StringToBytes(valAddrStr); err != nil {
				return err
			}

			params := &types.QueryValidatorRequest{ValidatorAddr: valAddrStr}
			res, err := queryClient.Validator(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Validator)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryValidators implements the query all validators command.
func GetCmdQueryValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Short: "Query for all validators",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query details about all validators on a network.

Example:
$ %s query staking validators
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			result, err := queryClient.Validators(cmd.Context(), &types.QueryValidatorsRequest{
				// Leaving status empty on purpose to query all validators.
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(result)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "validators")

	return cmd
}

// GetCmdQueryValidatorUnbondingDelegations implements the query all unbonding delegatations from a validator command.
func GetCmdQueryValidatorUnbondingDelegations(vc address.Codec) *cobra.Command {
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "unbonding-delegations-from [validator-addr]",
		Short: "Query all unbonding delegatations from a validator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query delegations that are unbonding _from_ a validator.

Example:
$ %s query staking unbonding-delegations-from %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			valAddrStr := args[0]
			if _, err := vc.StringToBytes(valAddrStr); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			params := &types.QueryValidatorUnbondingDelegationsRequest{
				ValidatorAddr: valAddrStr,
				Pagination:    pageReq,
			}

			res, err := queryClient.ValidatorUnbondingDelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "unbonding delegations")

	return cmd
}

// GetCmdQueryValidatorRedelegations implements the query all redelegatations
// from a validator command.
func GetCmdQueryValidatorRedelegations(vc address.Codec) *cobra.Command {
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "redelegations-from [validator-addr]",
		Short: "Query all outgoing redelegatations from a validator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query delegations that are redelegating _from_ a validator.

Example:
$ %s query staking redelegations-from %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			valAddrStr := args[0]
			if _, err := vc.StringToBytes(valAddrStr); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			params := &types.QueryRedelegationsRequest{
				SrcValidatorAddr: valAddrStr,
				Pagination:       pageReq,
			}

			res, err := queryClient.Redelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "validator redelegations")

	return cmd
}

// GetCmdQueryDelegation the query delegation command.
func GetCmdQueryDelegation(ac, vc address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "delegation [delegator-addr] [validator-addr]",
		Short: "Query a delegation based on address and validator address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query delegations for an individual delegator on an individual validator.

Example:
$ %s query staking delegation %s1gghjut3ccd8ay0zduzj64hwre2fxs9ld75ru9p %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixAccAddr, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			delAddrStr := args[0]
			if _, err := ac.StringToBytes(delAddrStr); err != nil {
				return err
			}

			valAddrStr := args[1]
			if _, err := vc.StringToBytes(valAddrStr); err != nil {
				return err
			}

			params := &types.QueryDelegationRequest{
				DelegatorAddr: delAddrStr,
				ValidatorAddr: valAddrStr,
			}

			res, err := queryClient.Delegation(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.DelegationResponse)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryDelegations implements the command to query all the delegations
// made from one delegator.
func GetCmdQueryDelegations(ac address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()

	cmd := &cobra.Command{
		Use:   "delegations [delegator-addr]",
		Short: "Query all delegations made by one delegator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query delegations for an individual delegator on all validators.

Example:
$ %s query staking delegations %s1gghjut3ccd8ay0zduzj64hwre2fxs9ld75ru9p
`,
				version.AppName, bech32PrefixAccAddr,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			delAddrStr := args[0]
			if _, err := ac.StringToBytes(delAddrStr); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			params := &types.QueryDelegatorDelegationsRequest{
				DelegatorAddr: delAddrStr,
				Pagination:    pageReq,
			}

			res, err := queryClient.DelegatorDelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "delegations")

	return cmd
}

// GetCmdQueryValidatorDelegations implements the command to query all the
// delegations to a specific validator.
func GetCmdQueryValidatorDelegations(vc address.Codec) *cobra.Command {
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "delegations-to [validator-addr]",
		Short: "Query all delegations made to one validator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query delegations on an individual validator.

Example:
$ %s query staking delegations-to %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			valAddrStr := args[0]
			if _, err := vc.StringToBytes(valAddrStr); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			params := &types.QueryValidatorDelegationsRequest{
				ValidatorAddr: valAddrStr,
				Pagination:    pageReq,
			}

			res, err := queryClient.ValidatorDelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "validator delegations")

	return cmd
}

// GetCmdQueryUnbondingDelegation implements the command to query a single
// unbonding-delegation record.
func GetCmdQueryUnbondingDelegation(ac, vc address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "unbonding-delegation [delegator-addr] [validator-addr]",
		Short: "Query an unbonding-delegation record based on delegator and validator address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query unbonding delegations for an individual delegator on an individual validator.

Example:
$ %s query staking unbonding-delegation %s1gghjut3ccd8ay0zduzj64hwre2fxs9ld75ru9p %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixAccAddr, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			delAddrStr := args[0]
			if _, err := ac.StringToBytes(delAddrStr); err != nil {
				return err
			}

			valAddrStr := args[1]
			if _, err := vc.StringToBytes(valAddrStr); err != nil {
				return err
			}

			params := &types.QueryUnbondingDelegationRequest{
				DelegatorAddr: delAddrStr,
				ValidatorAddr: valAddrStr,
			}

			res, err := queryClient.UnbondingDelegation(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Unbond)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryUnbondingDelegations implements the command to query all the
// unbonding-delegation records for a delegator.
func GetCmdQueryUnbondingDelegations(ac address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()

	cmd := &cobra.Command{
		Use:   "unbonding-delegations [delegator-addr]",
		Short: "Query all unbonding-delegations records for one delegator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query unbonding delegations for an individual delegator.

Example:
$ %s query staking unbonding-delegations %s1gghjut3ccd8ay0zduzj64hwre2fxs9ld75ru9p
`,
				version.AppName, bech32PrefixAccAddr,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			delAddrStr := args[0]
			if _, err := ac.StringToBytes(delAddrStr); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			params := &types.QueryDelegatorUnbondingDelegationsRequest{
				DelegatorAddr: delAddrStr,
				Pagination:    pageReq,
			}

			res, err := queryClient.DelegatorUnbondingDelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "unbonding delegations")

	return cmd
}

// GetCmdQueryRedelegation implements the command to query a single
// redelegation record.
func GetCmdQueryRedelegation(ac, vc address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()
	bech32PrefixValAddr := sdk.GetConfig().GetBech32ValidatorAddrPrefix()

	cmd := &cobra.Command{
		Use:   "redelegation [delegator-addr] [src-validator-addr] [dst-validator-addr]",
		Short: "Query a redelegation record based on delegator and a source and destination validator address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query a redelegation record for an individual delegator between a source and destination validator.

Example:
$ %s query staking redelegation %s1gghjut3ccd8ay0zduzj64hwre2fxs9ld75ru9p %s1l2rsakp388kuv9k8qzq6lrm9taddae7fpx59wm %s1gghjut3ccd8ay0zduzj64hwre2fxs9ldmqhffj
`,
				version.AppName, bech32PrefixAccAddr, bech32PrefixValAddr, bech32PrefixValAddr,
			),
		),
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			delAddrStr := args[0]
			if _, err := ac.StringToBytes(delAddrStr); err != nil {
				return err
			}

			valSrcAddrStr := args[1]
			if _, err := vc.StringToBytes(valSrcAddrStr); err != nil {
				return err
			}

			valDstAddrStr := args[2]
			if _, err := vc.StringToBytes(valDstAddrStr); err != nil {
				return err
			}

			params := &types.QueryRedelegationsRequest{
				DelegatorAddr:    delAddrStr,
				DstValidatorAddr: valDstAddrStr,
				SrcValidatorAddr: valSrcAddrStr,
			}

			res, err := queryClient.Redelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryRedelegations implements the command to query all the
// redelegation records for a delegator.
func GetCmdQueryRedelegations(ac address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()

	cmd := &cobra.Command{
		Use:   "redelegations [delegator-addr]",
		Args:  cobra.ExactArgs(1),
		Short: "Query all redelegations records for one delegator",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query all redelegation records for an individual delegator.

Example:
$ %s query staking redelegations %s1gghjut3ccd8ay0zduzj64hwre2fxs9ld75ru9p
`,
				version.AppName, bech32PrefixAccAddr,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			delAddrStr := args[0]
			if _, err := ac.StringToBytes(delAddrStr); err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			params := &types.QueryRedelegationsRequest{
				DelegatorAddr: delAddrStr,
				Pagination:    pageReq,
			}

			res, err := queryClient.Redelegations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "delegator redelegations")

	return cmd
}

// GetCmdQueryPool implements the pool query command.
func GetCmdQueryPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Args:  cobra.NoArgs,
		Short: "Query the current staking pool values",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query values for amounts stored in the staking pool.

Example:
$ %s query staking pool
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Pool(cmd.Context(), &types.QueryPoolRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Pool)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryParams implements the params query command.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Query the current staking parameters information",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query values set as staking parameters.

Example:
$ %s query staking params
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryMigration implements the migration query command.
func GetCmdQueryMigration() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration [lp-denom-in] [lp-denom-out]",
		Args:  cobra.ExactArgs(2),
		Short: "Query the migration info",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the migration info.

Example:
$ %s query staking migration USDC iUSD
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Migration(cmd.Context(), &types.QueryMigrationRequest{
				LpDenomIn:  args[0],
				LpDenomOut: args[1],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Migration)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
