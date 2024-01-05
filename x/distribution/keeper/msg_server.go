package keeper

import (
	"context"
	"time"

	"github.com/hashicorp/go-metrics"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the distribution MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) SetWithdrawAddress(ctx context.Context, msg *types.MsgSetWithdrawAddress) (*types.MsgSetWithdrawAddressResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "distribution", "msg", "set-withdraw-address")

	delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(msg.DelegatorAddress)
	if err != nil {
		return nil, err
	}
	withdrawAddress, err := k.authKeeper.AddressCodec().StringToBytes(msg.WithdrawAddress)
	if err != nil {
		return nil, err
	}
	err = k.SetWithdrawAddr(ctx, delegatorAddress, withdrawAddress)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetWithdrawAddressResponse{}, nil
}

func (k msgServer) WithdrawDelegatorReward(ctx context.Context, msg *types.MsgWithdrawDelegatorReward) (*types.MsgWithdrawDelegatorRewardResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "distribution", "msg", "withdraw-delegator-reward")

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(msg.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	rewards, err := k.WithdrawDelegationRewards(ctx, delegatorAddress, valAddr)
	if err != nil {
		return nil, err
	}

	defer func() {
		for _, a := range rewards.Sum() {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "withdraw_reward"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()

	return &types.MsgWithdrawDelegatorRewardResponse{}, nil
}

func (k msgServer) WithdrawValidatorCommission(ctx context.Context, msg *types.MsgWithdrawValidatorCommission) (*types.MsgWithdrawValidatorCommissionResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "distribution", "msg", "withdraw-validator-commission")

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	commissions, err := k.Keeper.WithdrawValidatorCommission(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	amount := commissions.Sum()
	defer func() {
		for _, a := range amount {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "withdraw_commission"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()

	return &types.MsgWithdrawValidatorCommissionResponse{Amount: amount}, nil
}

func (k msgServer) FundCommunityPool(ctx context.Context, msg *types.MsgFundCommunityPool) (*types.MsgFundCommunityPoolResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "distribution", "msg", "fund-community-pool")

	depositor, err := k.authKeeper.AddressCodec().StringToBytes(msg.Depositor)
	if err != nil {
		return nil, err
	}
	if err := k.Keeper.FundCommunityPool(ctx, msg.Amount, depositor); err != nil {
		return nil, err
	}

	return &types.MsgFundCommunityPoolResponse{}, nil
}

func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not supported")
}

func (k msgServer) CommunityPoolSpend(ctx context.Context, req *types.MsgCommunityPoolSpend) (*types.MsgCommunityPoolSpendResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "distribution", "msg", "community-pool-spend")
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	recipient, err := k.authKeeper.AddressCodec().StringToBytes(req.Recipient)
	if err != nil {
		return nil, err
	}

	if k.bankKeeper.BlockedAddr(recipient) {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive external funds", req.Recipient)
	}

	if err := k.DistributeFromFeePool(ctx, req.Amount, recipient); err != nil {
		return nil, err
	}

	logger := k.Logger(ctx)
	logger.Info("transferred from the community pool to recipient", "amount", req.Amount.String(), "recipient", req.Recipient)

	return &types.MsgCommunityPoolSpendResponse{}, nil
}

func (k msgServer) DepositValidatorRewardsPool(ctx context.Context, msg *types.MsgDepositValidatorRewardsPool) (*types.MsgDepositValidatorRewardsPoolResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not supported")
}
