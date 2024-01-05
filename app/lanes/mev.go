package lanes

import (
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	mstakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ auctiontypes.RewardsAddressProvider = (*RewardsAddressProvider)(nil)

// NewRewardsAddressProvider returns a new RewardsAddressProvider from a staking + distribution keeper
func NewRewardsAddressProvider(sk mstakingkeeper.Keeper, dk distrkeeper.Keeper) *RewardsAddressProvider {
	return &RewardsAddressProvider{
		sk: sk,
		dk: dk,
	}
}

// RewardsAddressProvider implements the x/auction's RewardsAddressProvider interface. It is used
// to determine the address to which the rewards from the most recent block's auction are sent.
type RewardsAddressProvider struct {
	sk mstakingkeeper.Keeper
	dk distrkeeper.Keeper
}

// GetRewardsAddress returns the address of the proposer of the previous block
func (rap *RewardsAddressProvider) GetRewardsAddress(ctx sdk.Context) (sdk.AccAddress, error) {
	// get previous proposer
	prevProposer, err := rap.dk.PreviousProposerConsAddr.Get(ctx)
	if err != nil {
		return sdk.AccAddress{}, err
	}

	// get validator from state corresponding to proposer
	valAddr, err := rap.sk.ValidatorsByConsAddr.Get(ctx, prevProposer)
	if err != nil {
		return sdk.AccAddress{}, err
	}

	// return validator's operator address
	return sdk.AccAddress(valAddr), nil
}
