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

const (
	// LaneName defines the name of the mev lane.
	LaneName = "mev"
)

// MEVLane defines a MEV (Maximal Extracted Value) auction lane. The MEV auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The MEV auction lane stores bid transactions that are sorted by their bid price.
// The highest valid bid transaction is selected for inclusion in the next block.
// The bundled transactions of the selected bid transaction are also included in the
// next block.
type (
	MEVLane struct { //nolint
		*blockbase.BaseLane

		// Factory defines the API/functionality which is responsible for determining
		// if a transaction is a bid transaction and how to extract relevant
		// information from the transaction (bid, timeout, bidder, etc.).
		mevlane.Factory
	}
)

// NewMEVLane returns a new TOB lane.
func NewMEVLane(
	cfg blockbase.LaneConfig,
	factory mevlane.Factory,
	matchHandler blockbase.MatchHandler,
) *MEVLane {
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
		LaneName,
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

	return &MEVLane{
		BaseLane: baseLane,
		Factory:  factory,
	}
}
