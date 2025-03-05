package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cosmossdk.io/core/address"
	"github.com/spf13/cobra"

	movetypes "github.com/initia-labs/initia/v1/x/move/types"
	mstaking "github.com/initia-labs/initia/v1/x/mstaking/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/cosmos/cosmos-sdk/x/authz"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Flag names and valuesi
const (
	FlagSpendLimit        = "spend-limit"
	FlagMsgType           = "msg-type"
	FlagExpiration        = "expiration"
	FlagAllowedValidators = "allowed-validators"
	FlagDenyValidators    = "deny-validators"
	FlagAllowList         = "allow-list"
	FlagType              = "type"
	FlagItems             = "items"
	FlagModules           = "modules"
	generic               = "generic"
	delegate              = "delegate"
	redelegate            = "redelegate"
	unbond                = "unbond"
	move                  = "move"
	publish               = "publish"
	execute               = "execute"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd(ac, vc address.Codec) *cobra.Command {
	AuthorizationTxCmd := &cobra.Command{
		Use:                        authz.ModuleName,
		Short:                      "Authorization transactions subcommands",
		Long:                       "Authorize and revoke access to execute transactions on behalf of your address",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	AuthorizationTxCmd.AddCommand(
		NewCmdGrantAuthorization(ac, vc),
		NewCmdRevokeAuthorization(ac),
		NewCmdExecAuthorization(),
	)

	return AuthorizationTxCmd
}

func NewCmdGrantAuthorization(ac, vc address.Codec) *cobra.Command {
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
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee, err := ac.StringToBytes(args[0])
			if err != nil {
				return err
			}

			var authorization authz.Authorization
			switch args[1] {
			case "send":
				limit, err := cmd.Flags().GetString(FlagSpendLimit)
				if err != nil {
					return err
				}

				spendLimit, err := sdk.ParseCoinsNormalized(limit)
				if err != nil {
					return err
				}

				if !spendLimit.IsAllPositive() {
					return fmt.Errorf("spend-limit should be greater than zero")
				}

				allowList, err := cmd.Flags().GetStringSlice(FlagAllowList)
				if err != nil {
					return err
				}

				allowed, err := bech32toAccAddresses(allowList, ac)
				if err != nil {
					return err
				}

				authorization = bank.NewSendAuthorization(spendLimit, allowed)
			case generic:
				msgType, err := cmd.Flags().GetString(FlagMsgType)
				if err != nil {
					return err
				}

				authorization = authz.NewGenericAuthorization(msgType)
			case delegate, unbond, redelegate:
				limit, err := cmd.Flags().GetString(FlagSpendLimit)
				if err != nil {
					return err
				}

				allowValidators, err := cmd.Flags().GetStringSlice(FlagAllowedValidators)
				if err != nil {
					return err
				}

				denyValidators, err := cmd.Flags().GetStringSlice(FlagDenyValidators)
				if err != nil {
					return err
				}

				var delegateLimit sdk.Coins
				if limit != "" {
					spendLimit, err := sdk.ParseCoinsNormalized(limit)
					if err != nil {
						return err
					}

					if !spendLimit.IsAllPositive() {
						return fmt.Errorf("spend-limit should be greater than zero")
					}
					delegateLimit = spendLimit
				}

				_, err = bech32toValidatorAddresses(allowValidators, vc)
				if err != nil {
					return err
				}

				_, err = bech32toValidatorAddresses(denyValidators, vc)
				if err != nil {
					return err
				}

				switch args[1] {
				case delegate:
					authorization, err = mstaking.NewStakeAuthorization(allowValidators, denyValidators, mstaking.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE, delegateLimit)
				case unbond:
					authorization, err = mstaking.NewStakeAuthorization(allowValidators, denyValidators, mstaking.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE, delegateLimit)
				default:
					authorization, err = mstaking.NewStakeAuthorization(allowValidators, denyValidators, mstaking.AuthorizationType_AUTHORIZATION_TYPE_REDELEGATE, delegateLimit)
				}
				if err != nil {
					return err
				}
			case move:
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

			default:
				return fmt.Errorf("invalid authorization type, %s", args[1])
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
	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(FlagMsgType, "", "The Msg method name for which we are creating a GenericAuthorization")
	cmd.Flags().String(FlagSpendLimit, "", "SpendLimit for Send Authorization, an array of Coins allowed spend")
	cmd.Flags().StringSlice(FlagAllowedValidators, []string{}, "Allowed validators addresses separated by ,")
	cmd.Flags().StringSlice(FlagDenyValidators, []string{}, "Deny validators addresses separated by ,")
	cmd.Flags().StringSlice(FlagAllowList, []string{}, "Allowed addresses grantee is allowed to send funds separated by ,")
	cmd.Flags().Int64(FlagExpiration, time.Now().UTC().AddDate(1, 0, 0).Unix(), "The Unix timestamp. Default is one year.")
	cmd.Flags().String(FlagType, "", "The type of move authorization, {publish|execute}")
	cmd.Flags().String(FlagItems, "", "The items of move execute authorization, a json file path.")
	cmd.Flags().String(FlagModules, "", "The items of move publish authorization, a comma-separated string of module names.")
	return cmd
}

func getExpireTime(cmd *cobra.Command) (*time.Time, error) {
	exp, err := cmd.Flags().GetInt64(FlagExpiration)
	if err != nil {
		return nil, err
	}
	if exp == 0 {
		return nil, nil
	}
	e := time.Unix(exp, 0)
	return &e, nil
}

func NewCmdRevokeAuthorization(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke [grantee] [msg_type] --from=[granter]",
		Short: "revoke authorization",
		Long: strings.TrimSpace(
			fmt.Sprintf(`revoke authorization from a granter to a grantee:
Example:
 $ %s tx %s revoke init1vrit.. %s --from=init1vrit..
            `, version.AppName, authz.ModuleName, bank.SendAuthorization{}.MsgTypeURL()),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee, err := ac.StringToBytes(args[0])
			if err != nil {
				return err
			}

			granter := clientCtx.GetFromAddress()
			msgAuthorized := args[1]
			msg := authz.NewMsgRevoke(granter, grantee, msgAuthorized)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCmdExecAuthorization() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [msg_tx_json_file] --from [grantee]",
		Short: "execute tx on behalf of granter account",
		Long: strings.TrimSpace(
			fmt.Sprintf(`execute tx on behalf of granter account:
Example:
 $ %s tx %s exec tx.json --from grantee
 $ %s tx bank send <granter> <recipient> --from <granter> --chain-id <chain-id> --generate-only > tx.json && %s tx %s exec tx.json --from grantee
            `, version.AppName, authz.ModuleName, version.AppName, version.AppName, authz.ModuleName),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			grantee := clientCtx.GetFromAddress()

			if offline, _ := cmd.Flags().GetBool(flags.FlagOffline); offline {
				return errors.New("cannot broadcast tx during offline mode")
			}

			theTx, err := authclient.ReadTxFromFile(clientCtx, args[0])
			if err != nil {
				return err
			}
			msg := authz.NewMsgExec(grantee, theTx.GetMsgs())

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func bech32toValidatorAddresses(validators []string, vc address.Codec) ([]sdk.ValAddress, error) {
	vals := make([]sdk.ValAddress, len(validators))
	for i, addr := range validators {
		valAddr, err := vc.StringToBytes(addr)
		if err != nil {
			return nil, err
		}
		vals[i] = valAddr
	}
	return vals, nil
}

// bech32toAccAddresses returns []AccAddress from a list of Bech32 string addresses.
func bech32toAccAddresses(accAddrs []string, ac address.Codec) ([]sdk.AccAddress, error) {
	addrs := make([]sdk.AccAddress, len(accAddrs))
	for i, addr := range accAddrs {
		accAddr, err := ac.StringToBytes(addr)
		if err != nil {
			return nil, err
		}
		addrs[i] = accAddr
	}
	return addrs, nil
}
