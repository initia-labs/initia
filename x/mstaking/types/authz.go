package types

import (
	"context"
	fmt "fmt"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// TODO: Revisit this once we have propoer gas fee framework.
// Tracking issues https://github.com/cosmos/cosmos-sdk/issues/9054, https://github.com/cosmos/cosmos-sdk/discussions/9072
const gasCostPerIteration = uint64(10)

// Normalized Msg type URLs
var (
	_ authz.Authorization = &StakeAuthorization{}
)

// NewStakeAuthorization creates a new StakeAuthorization object.
func NewStakeAuthorization(allowedValidators []string, deniedValidators []string, authzType AuthorizationType, amount sdk.Coins) (*StakeAuthorization, error) {
	a := StakeAuthorization{}
	if allowedValidators != nil {
		a.Validators = &StakeAuthorization_AllowList{AllowList: &StakeAuthorization_Validators{Address: allowedValidators}}
	} else {
		a.Validators = &StakeAuthorization_DenyList{DenyList: &StakeAuthorization_Validators{Address: deniedValidators}}
	}

	if amount != nil {
		a.MaxTokens = amount
	}
	a.AuthorizationType = authzType

	return &a, nil
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a StakeAuthorization) MsgTypeURL() string {
	authzType, err := normalizeAuthzType(a.AuthorizationType)
	if err != nil {
		panic(err)
	}
	return authzType
}

func (a StakeAuthorization) ValidateBasic() error {
	if a.MaxTokens != nil && !a.MaxTokens.IsAllPositive() {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "negative coin amount: %v", a.MaxTokens)
	}
	if a.AuthorizationType == AuthorizationType_AUTHORIZATION_TYPE_UNSPECIFIED {
		return errors.Wrapf(sdkerrors.ErrInvalidType, "unknown authorization type")
	}

	return nil
}

// Accept implements Authorization.Accept.
func (a StakeAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	var validatorAddress string
	var amount sdk.Coins

	switch msg := msg.(type) {
	case *MsgDelegate:
		validatorAddress = msg.ValidatorAddress
		amount = msg.Amount
	case *MsgUndelegate:
		validatorAddress = msg.ValidatorAddress
		amount = msg.Amount
	case *MsgBeginRedelegate:
		validatorAddress = msg.ValidatorDstAddress
		amount = msg.Amount
	case *MsgCancelUnbondingDelegation:
		validatorAddress = msg.ValidatorAddress
		amount = msg.Amount
	default:
		return authz.AcceptResponse{}, sdkerrors.ErrInvalidRequest.Wrap("unknown msg type")
	}

	isValidatorExists := false
	allowedList := a.GetAllowList().GetAddress()
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, validator := range allowedList {
		sdkCtx.GasMeter().ConsumeGas(gasCostPerIteration, "stake authorization")
		if validator == validatorAddress {
			isValidatorExists = true
			break
		}
	}

	denyList := a.GetDenyList().GetAddress()
	for _, validator := range denyList {
		sdkCtx.GasMeter().ConsumeGas(gasCostPerIteration, "stake authorization")
		if validator == validatorAddress {
			return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrapf(" cannot delegate/undelegate to %s validator", validator)
		}
	}

	if len(allowedList) > 0 && !isValidatorExists {
		return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrapf("cannot delegate/undelegate to %s validator", validatorAddress)
	}

	if a.MaxTokens == nil {
		return authz.AcceptResponse{
			Accept: true, Delete: false,
			Updated: &StakeAuthorization{
				Validators:        a.GetValidators(),
				AuthorizationType: a.GetAuthorizationType(),
			},
		}, nil
	}

	limitLeft, ok := a.MaxTokens.SafeSub(amount...)
	if !ok {
		return authz.AcceptResponse{}, fmt.Errorf("negative coins: %s", limitLeft)
	}

	if limitLeft.IsZero() {
		return authz.AcceptResponse{Accept: true, Delete: true}, nil
	}

	return authz.AcceptResponse{
		Accept: true,
		Delete: false,
		Updated: &StakeAuthorization{
			Validators:        a.GetValidators(),
			AuthorizationType: a.GetAuthorizationType(),
			MaxTokens:         limitLeft,
		},
	}, nil
}

func normalizeAuthzType(authzType AuthorizationType) (string, error) {
	switch authzType {
	case AuthorizationType_AUTHORIZATION_TYPE_DELEGATE:
		return sdk.MsgTypeURL(&MsgDelegate{}), nil
	case AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE:
		return sdk.MsgTypeURL(&MsgUndelegate{}), nil
	case AuthorizationType_AUTHORIZATION_TYPE_REDELEGATE:
		return sdk.MsgTypeURL(&MsgBeginRedelegate{}), nil
	case AuthorizationType_AUTHORIZATION_TYPE_CANCEL_UNBONDING_DELEGATION:
		return sdk.MsgTypeURL(&MsgCancelUnbondingDelegation{}), nil
	default:
		return "", sdkerrors.ErrInvalidType.Wrapf("unknown authorization type %T", authzType)
	}
}
