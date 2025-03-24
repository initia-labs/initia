package keeper

import (
	"context"
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (k Keeper) HandleVMStargateMsg(ctx context.Context, req *vmtypes.CosmosMessage) (proto.Message, error) {
	var sdkMsg sdk.Msg
	err := k.cdc.UnmarshalInterfaceJSON(req.Data, &sdkMsg)
	if err != nil {
		return nil, err
	}

	if m, ok := sdkMsg.(sdk.HasValidateBasic); ok {
		if err := m.ValidateBasic(); err != nil {
			return nil, err
		}
	}

	// make sure this account can send it
	signer := types.ConvertVMAddressToSDKAddress(req.Sender)
	signers, _, err := k.cdc.GetMsgV1Signers(sdkMsg)
	if err != nil {
		return nil, err
	}
	for _, acct := range signers {
		if !signer.Equals(sdk.AccAddress(acct)) {
			return nil, errorsmod.Wrapf(
				sdkerrors.ErrUnauthorized,
				"required signer: `%s`, given signer: `%s`",
				hex.EncodeToString(acct),
				hex.EncodeToString(signer),
			)
		}
	}

	return sdkMsg, nil
}
