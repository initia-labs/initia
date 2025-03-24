package movecmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"

	"cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	movecli "github.com/initia-labs/initia/x/move/client/cli"
	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

func moveDeployCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [flags]",
		Short: "deploy a whole move package",
		Long:  "deploy a whole move package. This command occurs a tx to publish module bundle.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// build package
			flagBuild, err := cmd.Flags().GetBool(flagBuild)
			if err != nil {
				return err
			}

			if flagBuild {
				arg, err := getCompilerArgument(cmd)
				if err != nil {
					return err
				}

				err = buildContract(arg)
				if err != nil {
					return err
				}
			}

			flagVerify, err := cmd.Flags().GetBool(flagVerify)
			if err != nil {
				return err
			}

			return deploy(cmd, ac, false, flagVerify)
		},
	}

	// add flat set for upgrade policy
	cmd.Flags().AddFlagSet(movecli.FlagSetUpgradePolicy())

	addMoveDeployFlags(cmd)
	addMoveBuildFlags(cmd)
	addMoveVerifyFlags(cmd, false)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// OBJECT_CODE_DEPLOYMENT_DOMAIN_SEPARATOR is the domain separator used for object code deployment
var OBJECT_CODE_DEPLOYMENT_DOMAIN_SEPARATOR string = "initia_std::object_code_deployment"

func deployObjectCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy-object [target-name] [flags]",
		Short: "build and deploy a move package via @std::object_code_deployment",
		Long: `
Build and deploy a move package via @std::object_code_deployment.

This command adds the named address to the build config, so the user must set 
the target name to '_' in the Move.toml file.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			// compute object address
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, clientCtx.FromAddress)
			if err != nil {
				return err
			}

			// Add 2 here because the sequence number will be incremented by 1
			// during the deployment transaction.
			seq += 2

			var buf []byte
			bz1, err := vmtypes.SerializeString(OBJECT_CODE_DEPLOYMENT_DOMAIN_SEPARATOR)
			if err != nil {
				return err
			}
			bz2, err := vmtypes.SerializeUint64(seq)
			if err != nil {
				return err
			}

			// compute vm address
			vmAddr, err := vmtypes.NewAccountAddressFromBytes(clientCtx.FromAddress.Bytes())
			if err != nil {
				return err
			}

			// derive object address
			// address + domain separator + sequence + 0xFE
			buf = append(buf, vmAddr[:]...)
			buf = append(buf, bz1...)
			buf = append(buf, bz2...)
			buf = append(buf, 0xFE)

			objectAddrBz := sha3.Sum256(buf)
			objectAccAddr := sdk.AccAddress(objectAddrBz[:])
			objectVmAddr := vmtypes.AccountAddress(objectAddrBz[:])
			objectAddressName := args[0]

			if !clientCtx.SkipConfirm {
				if ok, err := input.GetConfirmation(
					fmt.Sprintf("Do you want to publish this package at object address 0x%x", objectAccAddr),
					bufio.NewReader(clientCtx.Input),
					os.Stderr,
				); err != nil {
					return err
				} else if !ok {
					return nil
				}
			} else {
				fmt.Printf("Publishing package at object address 0x%x\n", objectAccAddr)
			}

			// add object address's named address
			arg.BuildConfig.AdditionalNamedAddresses = append(arg.BuildConfig.AdditionalNamedAddresses,
				struct {
					Field0 string
					Field1 vmtypes.AccountAddress
				}{
					Field0: objectAddressName,
					Field1: objectVmAddr,
				},
			)

			err = buildContract(arg)
			if err != nil {
				return err
			}

			return deploy(cmd, ac, true, false)
		},
	}

	addMoveBuildFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func buildContract(arg *vmtypes.CompilerArguments) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	_, err = api.BuildContract(*arg)
	if err != nil {
		return err
	}

	// move build api changes the working directory
	// so we need to change it back
	if newPwd, err := os.Getwd(); err != nil {
		return err
	} else if pwd != newPwd {
		if err := os.Chdir(pwd); err != nil {
			return err
		}
	}

	return nil
}

func deploy(cmd *cobra.Command, ac address.Codec, isObjectDeployment, verify bool) error {
	// deploy package
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	packagePath, err := cmd.Flags().GetString(flagPackagePath)
	if err != nil {
		return err
	}

	moduleBundle, err := getModuleBundle(packagePath)
	if err != nil {
		return err
	}

	if len(moduleBundle) == 0 {
		return fmt.Errorf("module bundle is empty")
	}

	senderAddr := clientCtx.FromAddress
	senderStrAddr, err := ac.BytesToString(senderAddr.Bytes())
	if err != nil {
		return err
	}

	var msg sdk.Msg
	if isObjectDeployment {
		moduleBundleStr := marshalBytesArrayToHexArray(moduleBundle)
		executeMsg := &movetypes.MsgExecuteJSON{
			Sender:        senderStrAddr,
			ModuleAddress: vmtypes.StdAddress.String(),
			ModuleName:    "object_code_deployment",
			FunctionName:  "publish_v2",
			Args: []string{
				moduleBundleStr, // code bytes array
			},
		}

		if err = executeMsg.Validate(ac); err != nil {
			return err
		}

		msg = executeMsg
	} else {
		upgradePolicyStr, err := cmd.Flags().GetString(movecli.FlagUpgradePolicy)
		if err != nil {
			return err
		}

		upgradePolicy, found := movetypes.UpgradePolicy_value[upgradePolicyStr]
		if !found {
			return fmt.Errorf("invalid upgrade-policy `%s`", upgradePolicyStr)
		}

		publishMsg := &movetypes.MsgPublish{
			Sender:        senderStrAddr,
			CodeBytes:     moduleBundle,
			UpgradePolicy: movetypes.UpgradePolicy(upgradePolicy),
		}

		if err = publishMsg.Validate(ac); err != nil {
			return err
		}

		msg = publishMsg
	}

	err = tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
	if err != nil {
		return err
	}

	// request contract verify
	if verify {
		vc, err := getVerifyConfig(cmd)
		if err != nil {
			return err
		}

		if err := verifyContract(*vc); err != nil {
			return errorsmod.Wrap(err, "failed to verify published package")
		}
	}

	return nil
}

func marshalBytesArrayToHexArray(data [][]byte) string {
	str := "["
	for _, b := range data {
		str += fmt.Sprintf("\"%02x\",", b)
	}
	str = str[:len(str)-1]
	str += "]"
	return str
}
