package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/initia-labs/initia/v1/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
)

func GetQueryCmd(ac address.Codec) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the move module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		GetCmdModule(ac),
		GetCmdModules(ac),
		GetCmdResource(ac),
		GetCmdResources(ac),
		GetCmdTableEntry(ac),
		GetCmdTableEntries(ac),
		GetCmdQueryViewFunction(ac),
		GetCmdQueryViewJSONFunction(ac),
		GetCmdQueryParams(),
	)
	return queryCmd
}

func GetCmdModule(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "module [module owner] [module name]",
		Short:   "Get published move module info",
		Long:    "Get published move module info",
		Aliases: []string{"m"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Module(
				context.Background(),
				&types.QueryModuleRequest{
					Address:    args[0],
					ModuleName: args[1],
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdModules(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "modules [module owner]",
		Short:   "Get all published move module infos of an account",
		Long:    "Get all published move module infos of an account",
		Aliases: []string{"ms"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Modules(
				context.Background(),
				&types.QueryModulesRequest{
					Address: args[0],
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "list modules")
	return cmd
}

func GetCmdResource(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resource [resource owner] [struct tag]",
		Short:   "Get store raw resource data",
		Long:    "Get store raw resource data",
		Aliases: []string{"r"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			_, err = vmapi.ParseStructTag(args[1])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Resource(
				context.Background(),
				&types.QueryResourceRequest{
					Address:   args[0],
					StructTag: args[1],
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdResources(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resources [resource owner]",
		Short:   "Get all raw resource data of an account",
		Long:    "Get all raw resource data of an account",
		Aliases: []string{"rs"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Resources(
				context.Background(),
				&types.QueryResourcesRequest{
					Address: args[0],
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "list modules")
	return cmd
}

func GetCmdTableEntry(ac address.Codec) *cobra.Command {
	decoder := newArgDecoder(asciiDecodeString)
	cmd := &cobra.Command{
		Use:     "table-entry [table addr] [key_bytes]",
		Short:   "Get store raw table entry",
		Long:    "Get store raw table entry",
		Aliases: []string{"entry"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			keyBz, err := decoder.DecodeString(args[1])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.TableEntry(
				context.Background(),
				&types.QueryTableEntryRequest{
					Address:  args[0],
					KeyBytes: keyBz,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	decoder.RegisterFlags(cmd.PersistentFlags(), "key bytes")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetCmdTableEntries(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "table-entries [table addr]",
		Short:   "Get all table entries",
		Long:    "Get all table entries",
		Aliases: []string{"entries"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.TableEntries(
				context.Background(),
				&types.QueryTableEntriesRequest{
					Address: args[0],
				},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "table entries")

	return cmd
}

func GetCmdQueryViewFunction(ac address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()
	cmd := &cobra.Command{
		Use:   "view [module owner] [module name] [function name]",
		Short: "Get view function execution result",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
Get an view function execution result

Supported types : u8, u16, u32, u64, u128, u256, bool, string, address, raw_hex, raw_base64,
	vector<inner_type>, option<inner_type>, biguint, bigdecimal, fixed_point32, fixed_point64
Example of args: address:0x1 bool:true u8:0 string:hello vector<u32>:a,b,c,d

Example:
$ %s query move view \
    %s1lwjmdnks33xwnmfayc64ycprww49n33mtm92ne \
	ManagedCoin \
	get_balance \
	--type-args '["0x1::BasicCoin::getBalance<u8>", "0x1::BasicCoin::getBalance<u64>"]' \
 	--args '["address:0x1", "bool:true", "u8:0x01", "u128:1234", "vector<u32>:a,b,c,d", "string:hello world"]'
`, version.AppName, bech32PrefixAccAddr,
			),
		),
		Aliases: []string{"e"},
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			tyArgs, err := ReadAndDecodeJSONStringArray[string](cmd, FlagTypeArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to read type args")
			}

			flagArgs, err := ReadAndDecodeJSONStringArray[string](cmd, FlagArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to read move args")
			}

			bcsArgs, err := BCSEncode(ac, flagArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to encode move args")
			}

			queryClient := types.NewQueryClient(clientCtx)

			//nolint
			res, err := queryClient.View(
				context.Background(),
				&types.QueryViewRequest{
					Address:      args[0],
					ModuleName:   args[1],
					FunctionName: args[2],
					TypeArgs:     tyArgs,
					Args:         bcsArgs,
				},
			)

			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().AddFlagSet(FlagSetTypeArgs())
	cmd.Flags().AddFlagSet(FlagSetArgs())
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryViewJSONFunction(ac address.Codec) *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()
	cmd := &cobra.Command{
		Use:   "view-json [module owner] [module name] [function name]",
		Short: "Get view json function execution result",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
Get an view function execution result

Supported types : u8, u16, u32, u64, u128, u256, bool, string, address, raw_hex, raw_base64,
	vector<inner_type>, option<inner_type>, biguint, bigdecimal, fixed_point32, fixed_point64
Example of args: "0x1" "true" "0" "hello vector" ["a","b","c","d"]

Example:
$ %s query move view_json \
    %s1lwjmdnks33xwnmfayc64ycprww49n33mtm92ne \
	ManagedCoin \
	get_balance \
	--type-args '["0x1::BasicCoin::getBalance<u8>", "0x1::BasicCoin::getBalance<u64>"]' \
 	--args '[0, true, "0x1", "1234", ["a","b","c","d"]]'

Note that there should be no spaces within the arguments, since each argument is separated by a space.
`, version.AppName, bech32PrefixAccAddr,
			),
		),
		Aliases: []string{"ej"},
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			_, err = types.AccAddressFromString(ac, args[0])
			if err != nil {
				return err
			}

			tyArgs, err := ReadAndDecodeJSONStringArray[string](cmd, FlagTypeArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to read type args")
			}

			moveArgs, err := ReadJSONStringArray(cmd, FlagArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to read move args")
			}

			queryClient := types.NewQueryClient(clientCtx)

			//nolint
			res, err := queryClient.ViewJSON(
				context.Background(),
				&types.QueryViewJSONRequest{
					Address:      args[0],
					ModuleName:   args[1],
					FunctionName: args[2],
					TypeArgs:     tyArgs,
					Args:         moveArgs,
				},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().AddFlagSet(FlagSetTypeArgs())
	cmd.Flags().AddFlagSet(FlagSetJSONArgs())
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryParams implements the params query command.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Query the current move parameters information",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query values set as move parameters.

Example:
$ %s query move params
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
