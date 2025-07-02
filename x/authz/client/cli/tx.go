package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"cosmossdk.io/core/address"
	"github.com/spf13/cobra"

	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzcli "github.com/cosmos/cosmos-sdk/x/authz/client/cli"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Flag names and valuesi
const (
	FlagType    = "type"
	FlagItems   = "items"
	FlagModules = "modules"
	move        = "move"
	publish     = "publish"
	execute     = "execute"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd(ac, vc address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        authz.ModuleName,
		Short:                      "Authorization transactions subcommands",
		Long:                       "Authorize and revoke access to execute transactions on behalf of your address",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCmdGrantAuthorization(ac, vc),
		authzcli.NewCmdRevokeAuthorization(ac),
		authzcli.NewCmdExecAuthorization(),
	)

	return txCmd
}

func NewCmdGrantAuthorization(ac, vc address.Codec) *cobra.Command {
	originCmd := authzcli.NewCmdGrantAuthorization(ac)
	cmd := &cobra.Command{
		Use:   "grant <grantee> <authorization_type=\"send\"|\"generic\"|\"delegate\"|\"unbond\"|\"redelegate\"|\"move\"> --from <granter>",
		Short: "Grant authorization to an address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`grant authorization to an address to execute a transaction on your behalf:

Examples:
 $ %s tx %s grant init1vrit.. send %s --spend-limit=1000uinit --from=init1vrit..
 $ %s tx %s grant init1vrit.. generic --msg-type=/cosmos.gov.v1beta1.MsgVote --from=init1vrit..
 $ %s tx %s grant init1vrit.. move --type publish --modules "secp256*1,ed*"  --from=init1vrit..
 $ %s tx %s grant init1vrit.. move --type execute --items ./authzItems.json --from=init1vrit..

Where authzItems.json contains:
[
    {
        "module_address": "init1vr...",
        "module_name": "foo",
        "function_names": ["ba*"]
    },
  	{
        "module_address": "init1vr...",
        "module_name": "bar",
        "function_names": ["baz"]
    }
]
    `, version.AppName, authz.ModuleName, bank.SendAuthorization{}.MsgTypeURL(), version.AppName, authz.ModuleName, version.AppName, authz.ModuleName, version.AppName, authz.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If not a Move authorization, delegate to standard authz CLI handler
			if args[1] != move {
				return originCmd.RunE(cmd, args)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee, err := ac.StringToBytes(args[0])
			if err != nil {
				return err
			}

			var authorization authz.Authorization
			typ, err := cmd.Flags().GetString(FlagType)
			if err != nil {
				return err
			}

			switch typ {
			case publish:
				var modpatstr string
				modpatstr, err = cmd.Flags().GetString(FlagModules)
				if err != nil {
					return err
				}
				if modpatstr == "" {
					return fmt.Errorf("module pattern is empty")
				}
				modulePatterns := strings.Split(modpatstr, ",")
				authorization, err = movetypes.NewPublishAuthorization(modulePatterns)
			case execute:
				var items string
				items, err = cmd.Flags().GetString(FlagItems)
				if err != nil {
					return err
				}
				if items == "" {
					return fmt.Errorf("items file path is empty")
				}
				var itemsBytes []byte
				itemsBytes, err = os.ReadFile(items)
				if err != nil {
					return err
				}
				var authzItems []movetypes.ExecuteAuthorizationItem
				if err = json.Unmarshal(itemsBytes, &authzItems); err != nil {
					return fmt.Errorf("invalid authorization item, %s", items)
				}
				authorization, err = movetypes.NewExecuteAuthorization(ac, authzItems)
			default:
				return fmt.Errorf("invalid type, %s", typ)
			}
			if err != nil {
				return err
			}

			expire, err := getExpireTime(cmd)
			if err != nil {
				return err
			}

			msg, err := authz.NewMsgGrant(clientCtx.GetFromAddress(), grantee, authorization, expire)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(originCmd.Flags())
	cmd.Flags().String(FlagType, "", "The type of move authorization, {publish|execute}")
	cmd.Flags().String(FlagItems, "", "The items of move execute authorization, a json file path.")
	cmd.Flags().String(FlagModules, "", "The items of move publish authorization, a comma-separated string of module names.")
	return cmd
}

func getExpireTime(cmd *cobra.Command) (*time.Time, error) {
	exp, err := cmd.Flags().GetInt64(authzcli.FlagExpiration)
	if err != nil {
		return nil, err
	}
	if exp == 0 {
		return nil, nil
	}
	e := time.Unix(exp, 0)
	return &e, nil
}
