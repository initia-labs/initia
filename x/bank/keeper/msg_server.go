package keeper

import (
	"context"

	"github.com/hashicorp/go-metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	cosmosbank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	cosmosbank.Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper cosmosbank.Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) Send(goCtx context.Context, msg *types.MsgSend) (*types.MsgSendResponse, error) {
	var (
		from, to []byte
		err      error
	)

	if base, ok := k.Keeper.(BaseKeeper); ok {
		from, err = base.ak.AddressCodec().StringToBytes(msg.FromAddress)
		if err != nil {
			return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid from address: %s", err)
		}
		to, err = base.ak.AddressCodec().StringToBytes(msg.ToAddress)
		if err != nil {
			return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid to address: %s", err)
		}
	} else {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("invalid keeper type: %T", k.Keeper)
	}

	if !msg.Amount.IsValid() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	if !msg.Amount.IsAllPositive() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.IsSendEnabledCoins(ctx, msg.Amount...); err != nil {
		return nil, err
	}

	if k.BlockedAddr(to) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.ToAddress)
	}

	err = k.SendCoins(ctx, from, to, msg.Amount)
	if err != nil {
		return nil, err
	}

	defer func() {
		for _, a := range msg.Amount {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "send"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()

	return &types.MsgSendResponse{}, nil
}

func (k msgServer) MultiSend(goCtx context.Context, msg *types.MsgMultiSend) (*types.MsgMultiSendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not supported")
}

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := req.Params.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) SetSendEnabled(goCtx context.Context, msg *types.MsgSetSendEnabled) (*types.MsgSetSendEnabledResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	seen := map[string]bool{}
	for _, se := range msg.SendEnabled {
		if _, alreadySeen := seen[se.Denom]; alreadySeen {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("duplicate denom entries found for %q", se.Denom)
		}

		seen[se.Denom] = true

		if err := se.Validate(); err != nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("invalid SendEnabled denom %q: %s", se.Denom, err)
		}
	}

	for _, denom := range msg.UseDefaultFor {
		if err := sdk.ValidateDenom(denom); err != nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrapf("invalid UseDefaultFor denom %q: %s", denom, err)
		}
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if len(msg.SendEnabled) > 0 {
		k.SetAllSendEnabled(ctx, msg.SendEnabled)
	}
	if len(msg.UseDefaultFor) > 0 {
		k.DeleteSendEnabled(ctx, msg.UseDefaultFor...)
	}

	return &types.MsgSetSendEnabledResponse{}, nil
}
