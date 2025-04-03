package lanes

import (
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	mstakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
	mevlane "github.com/skip-mev/block-sdk/v2/lanes/mev"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
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

// NewMEVLane returns a new TOB lane.
func NewMEVLane(
	cfg blockbase.LaneConfig,
	factory mevlane.Factory,
	matchHandler blockbase.MatchHandler,
) *mevlane.MEVLane {
	mempool, err := NewMempool(
		mevlane.TxPriority(factory),
		cfg.SignerExtractor,
		cfg.MaxTxs,
		cfg.MaxBlockSpace,
		cfg.TxEncoder,
	)
	if err != nil {
		panic(err)
	}

	baseLane, err := blockbase.NewBaseLane(
		cfg,
		mevlane.LaneName,
		blockbase.WithMatchHandler(matchHandler),
		blockbase.WithMempool(mempool),
	)
	if err != nil {
		panic(err)
	}

	// Create the mev proposal handler.
	handler := mevlane.NewProposalHandler(baseLane, factory)
	baseLane.WithOptions(
		blockbase.WithMempool(mempool),
		blockbase.WithPrepareLaneHandler(handler.PrepareLaneHandler()),
		blockbase.WithProcessLaneHandler(handler.ProcessLaneHandler()),
	)

	return &mevlane.MEVLane{
		BaseLane: baseLane,
		Factory:  factory,
	}
}
