package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// NewUpdateAdmin returns the command to create a MsgUpdateAdmin transaction
func NewUpdateAdminCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-admin [port] [channel] [admin]",
		Short:   "Transfer a ownership of a channel to a new admin",
		Example: fmt.Sprintf("%s tx ibc-perm update-admin [port] [channel] [admin]", version.AppName),
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sender, err := ac.BytesToString(clientCtx.GetFromAddress())
			if err != nil {
				return err
			}

			port := args[0]
			channel := args[1]
			admin := args[2]

			if _, err := ac.StringToBytes(admin); err != nil {
				return err
			}

			msg := types.NewMsgUpdateAdmin(sender, port, channel, admin)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdatePermissionedRelayersCmd returns the command to create a MsgUpdatePermissionedRelayers transaction
func NewUpdatePermissionedRelayersCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-relayers [port] [channel] [relayer],...,[relayer]",
		Short:   "Give a list of relayers permission to relay packets on a channel",
		Example: fmt.Sprintf("%s tx ibc-perm update-relayers [port] [channel] [relayer],...,[relayer]", version.AppName),
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sender, err := ac.BytesToString(clientCtx.GetFromAddress())
			if err != nil {
				return err
			}

			port := args[0]
			channel := args[1]
			relayers := strings.Split(args[2], ",")

			for _, relayer := range relayers {
				if _, err := ac.StringToBytes(relayer); err != nil {
					return err
				}
			}

			msg := types.NewMsgUpdatePermissionedRelayers(sender, port, channel, relayers)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
