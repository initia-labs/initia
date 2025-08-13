package keeper

import (
	"context"
	"slices"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

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

func (k customMsgServer) AddEmergencySubmitters(ctx context.Context, req *customtypes.MsgAddEmergencySubmitters) (*customtypes.MsgAddEmergencySubmittersResponse, error) {
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

	return &customtypes.MsgAddEmergencySubmittersResponse{}, nil
}

func (k customMsgServer) RemoveEmergencySubmitters(ctx context.Context, req *customtypes.MsgRemoveEmergencySubmitters) (*customtypes.MsgRemoveEmergencySubmittersResponse, error) {
	if k.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	for _, submitter := range req.EmergencySubmitters {
		if index := slices.Index(params.EmergencySubmitters, submitter); index != -1 {
			params.EmergencySubmitters = slices.Delete(params.EmergencySubmitters, index, index+1)
		} else {
			return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "submitter not found: %s", submitter)
		}
	}

	if err := k.Params.Set(ctx, params); err != nil {
		return nil, err
	}

	return &customtypes.MsgRemoveEmergencySubmittersResponse{}, nil
}

// ActivateEmergencyProposal activates an emergency proposal.
//
// It checks if the proposal is in the voting period, has met the emergency minimum deposit,
// and if the sender is an authorized emergency submitter.
//
// If all conditions are met, it activates the emergency proposal.
func (k customMsgServer) ActivateEmergencyProposal(ctx context.Context, req *customtypes.MsgActivateEmergencyProposal) (*customtypes.MsgActivateEmergencyProposalResponse, error) {
	proposal, err := k.Proposals.Get(ctx, req.ProposalId)
	if err != nil {
		return nil, err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// validate proposal is not already an emergency proposal
	if proposal.Emergency {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("proposal %d is already an emergency proposal", req.ProposalId)
	}

	// validate proposal is in voting period
	if proposal.Status != v1.StatusVotingPeriod {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("proposal %d is not in voting period", req.ProposalId)
	}

	// validate proposal has met the emergency min deposit
	if !sdk.NewCoins(proposal.TotalDeposit...).IsAllGTE(params.EmergencyMinDeposit) {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("proposal %d has not met the emergency min deposit", req.ProposalId)
	}

	// validate sender is an authorized emergency submitter
	if !slices.Contains(params.EmergencySubmitters, req.Sender) {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("sender %s is not in the emergency submitters list", req.Sender)
	}

	// activate emergency proposal
	err = k.Keeper.ActivateEmergencyProposal(ctx, proposal)
	if err != nil {
		return nil, err
	}

	return &customtypes.MsgActivateEmergencyProposalResponse{}, nil
}
