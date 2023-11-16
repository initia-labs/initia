package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

func (k Keeper) InsertEmergencyProposalQueue(ctx sdk.Context, proposalID uint64) {
	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Set(customtypes.GetEmergencyProposalKey(proposalID), []byte{1})
}

func (k Keeper) RemoveFromEmergencyProposalQueue(ctx sdk.Context, proposalID uint64) {
	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Delete(customtypes.GetEmergencyProposalKey(proposalID))
}

func (k Keeper) IterateEmergencyProposals(ctx sdk.Context, cb func(proposal v1.Proposal) (stop bool)) {
	kvStore := ctx.KVStore(k.storeKey)
	prefix := prefix.NewStore(kvStore, customtypes.EmergencyProposalsPrefix)
	iterator := prefix.Iterator(nil, nil)
	for ; iterator.Valid(); iterator.Next() {
		proposalID := types.GetProposalIDFromBytes(iterator.Key())
		proposal, found := k.GetProposal(ctx, proposalID)
		if !found {
			panic(fmt.Sprintf("proposal %d does not exist", proposalID))
		}

		if cb(proposal) {
			break
		}
	}

	return
}

func (k Keeper) RecordLastEmergencyProposalTallyTimestamp(ctx sdk.Context) {
	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Set(customtypes.LastEmergencyProposalTallyTimestampKey, sdk.FormatTimeBytes(ctx.BlockTime()))
}

func (k Keeper) SetLastEmergencyProposalTallyTimestamp(ctx sdk.Context, t time.Time) {
	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Set(customtypes.LastEmergencyProposalTallyTimestampKey, sdk.FormatTimeBytes(t))
}

func (k Keeper) GetLastEmergencyProposalTallyTimestamp(ctx sdk.Context) time.Time {
	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(customtypes.LastEmergencyProposalTallyTimestampKey)
	if len(bz) == 0 {
		panic(fmt.Sprintf("LastEmergencyProposalTallyTimestamp not found"))
	}

	t, err := sdk.ParseTimeBytes(bz)
	if err != nil {
		panic(err)
	}

	return t
}
