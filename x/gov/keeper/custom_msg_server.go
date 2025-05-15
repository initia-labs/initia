package keeper

import (
	"context"
	"slices"

	"cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

type customMsgServer struct {
	*Keeper
}

// NewCustomMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewCustomMsgServerImpl(keeper *Keeper) customtypes.MsgServer {
	return &customMsgServer{Keeper: keeper}
}

var _ customtypes.MsgServer = customMsgServer{}

func (k customMsgServer) UpdateParams(ctx context.Context, req *customtypes.MsgUpdateParams) (*customtypes.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	if err := req.Params.Validate(k.authKeeper.AddressCodec()); err != nil {
		return nil, err
	}

	if err := k.Params.Set(ctx, req.Params); err != nil {
		return nil, err
	}

	return &customtypes.MsgUpdateParamsResponse{}, nil
}

func (k customMsgServer) AddEmergencyProposalSubmitters(ctx context.Context, req *customtypes.MsgAddEmergencyProposalSubmitters) (*customtypes.MsgAddEmergencyProposalSubmittersResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	for _, submitter := range req.EmergencySubmitters {
		if _, err := k.authKeeper.AddressCodec().StringToBytes(submitter); err != nil {
			return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid submitter: %s", submitter)
		}

		if !slices.Contains(params.EmergencySubmitters, submitter) {
			params.EmergencySubmitters = append(params.EmergencySubmitters, submitter)
		}
	}

	if err := k.Params.Set(ctx, params); err != nil {
		return nil, err
	}

	return &customtypes.MsgAddEmergencyProposalSubmittersResponse{}, nil
}

func (k customMsgServer) RemoveEmergencyProposalSubmitters(ctx context.Context, req *customtypes.MsgRemoveEmergencyProposalSubmitters) (*customtypes.MsgRemoveEmergencyProposalSubmittersResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	for _, submitter := range req.EmergencySubmitters {
		if index := slices.Index(params.EmergencySubmitters, submitter); index != -1 {
			params.EmergencySubmitters = append(params.EmergencySubmitters[:index], params.EmergencySubmitters[index+1:]...)
		} else {
			return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "submitter not found: %s", submitter)
		}
	}

	if err := k.Params.Set(ctx, params); err != nil {
		return nil, err
	}

	return &customtypes.MsgRemoveEmergencyProposalSubmittersResponse{}, nil
}
