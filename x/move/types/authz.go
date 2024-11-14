package types

import (
	context "context"

	"github.com/IGLOU-EU/go-wildcard"

	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/initia-labs/movevm/api"
)

// it has similar issue with gasCostPerIterarion in staking/type/authz.go
// TODO: Revisit this once we have proper gas fee framework.
// Tracking issues https://github.com/cosmos/cosmos-sdk/issues/9054, https://github.com/cosmos/cosmos-sdk/discussions/9072
const gasCostPerMatchTest = uint64(100)
const gasCostPerIteration = uint64(10)
const gasCostPerReadMoveModuleInfo = uint64(1000)

// Normalized Msg type URLs
var (
	_ authz.Authorization = &PublishAuthorization{}
	_ authz.Authorization = &ExecuteAuthorization{}
)

// NewPublishAuthorization creates a new PublishAuthorization object.
func NewPublishAuthorization(moduleNames []string) (*PublishAuthorization, error) {
	return &PublishAuthorization{moduleNames}, nil
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a PublishAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgPublish{})
}

func (a PublishAuthorization) ValidateBasic() error {
	if len(a.ModuleNames) == 0 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "module names cannot be empty")
	}
	for _, v := range a.ModuleNames {
		if len(v) == 0 {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid module names")
		}
	}
	return nil
}

func isMatching(moduleName string, patterns []string) bool {
	for _, pattern := range patterns {
		if wildcard.Match(pattern, moduleName) {
			return true
		}
	}
	return false
}

func (a PublishAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	switch msg := msg.(type) {
	case *MsgPublish:
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		for _, codeBytes := range msg.CodeBytes {
			sdkCtx.GasMeter().ConsumeGas(gasCostPerIteration, "publish authorization iteration")

			if len(codeBytes) == 0 {
				return authz.AcceptResponse{}, sdkerrors.ErrInvalidRequest.Wrap("empty module bytes")
			}

			sdkCtx.GasMeter().ConsumeGas(gasCostPerReadMoveModuleInfo, "read move module info")
			_, moduleName, err := api.ReadModuleInfo(codeBytes)
			if err != nil {
				return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrap("code bytes are not valid")
			}

			sdkCtx.GasMeter().ConsumeGas(gasCostPerMatchTest, "publish authorization match test")
			if !isMatching(moduleName, a.ModuleNames) {
				return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrap("unauthorized")
			}
		}
		return authz.AcceptResponse{Accept: true}, nil
	default:
		return authz.AcceptResponse{}, sdkerrors.ErrInvalidRequest.Wrap("unknown msg type")
	}
}

// NewExecuteAuthorization creates a new ExecuteAuthorization object.
func NewExecuteAuthorization(ac address.Codec, moduleIdentifiers []ExecuteAuthorizationItem) (*ExecuteAuthorization, error) {
	if len(moduleIdentifiers) == 0 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("moduleIdentifiers cannot be empty")
	}

	a := ExecuteAuthorization{}
	for _, moduleIdentifier := range moduleIdentifiers {
		addr, err := AccAddressFromString(ac, moduleIdentifier.ModuleAddress)
		if err != nil {
			return nil, sdkerrors.ErrInvalidAddress.Wrap("invalid module address")
		}
		a.Items = append(a.Items, ExecuteAuthorizationItem{
			ModuleAddress: addr.String(),
			ModuleName:    moduleIdentifier.ModuleName,
			FunctionNames: moduleIdentifier.FunctionNames,
		})
	}
	return &a, nil
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a ExecuteAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgExecute{})
}

func (a ExecuteAuthorization) ValidateBasic() error {
	moduleMap := make(map[string][]string)
	for _, v := range a.Items {
		if len(v.ModuleName) == 0 {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid module name: %s", v.ModuleName)
		}
		if len(v.FunctionNames) == 0 {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid module names")
		}
		if module, ok := moduleMap[v.ModuleAddress]; ok {
			for _, m := range module {
				if m == v.ModuleName {
					return errors.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate module name: %s", v.ModuleName)
				}
			}
			moduleMap[v.ModuleAddress] = append(module, v.ModuleName)
		} else {
			moduleMap[v.ModuleAddress] = []string{v.ModuleName}
		}
	}
	return nil
}

func (a ExecuteAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	switch msg := msg.(type) {
	case *MsgExecute:
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// TODO - cannot retrieve address codec here
		ac := codec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
		msgModuleAddr, err := AccAddressFromString(ac, msg.ModuleAddress)
		if err != nil {
			return authz.AcceptResponse{}, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid module address: %s", msg.ModuleAddress)
		}

		for _, v := range a.Items {
			moduleAddr, err := AccAddressFromString(ac, v.ModuleAddress)
			if err != nil {
				return authz.AcceptResponse{}, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid module address: %s", v.ModuleAddress)
			}

			sdkCtx.GasMeter().ConsumeGas(gasCostPerIteration, "execute authorization iteration")

			if !msgModuleAddr.Equals(moduleAddr) {
				continue
			}
			if msg.ModuleName != v.ModuleName {
				continue
			}
			sdkCtx.GasMeter().ConsumeGas(gasCostPerMatchTest, "execute authorization match test")
			if isMatching(msg.FunctionName, v.FunctionNames) {
				return authz.AcceptResponse{Accept: true}, nil
			}
		}
		// all items are checked, but no match
		return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrap("unauthorized")
	default:
		return authz.AcceptResponse{}, sdkerrors.ErrInvalidRequest.Wrap("unknown msg type")
	}
}
