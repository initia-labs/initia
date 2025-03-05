package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"

	stakingtypes "github.com/initia-labs/initia/v1/x/mstaking/types"
)

// InitGenesis initialize default parameters
// and the keeper's address to pubkey map
func (k Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {
	err := k.sk.IterateValidators(ctx,
		func(validator stakingtypes.ValidatorI) (bool, error) {
			consPk, err := validator.ConsPubKey()
			if err != nil {
				return true, err
			}
			if err = k.AddPubkey(ctx, consPk); err != nil {
				return true, err
			}
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	for _, info := range data.SigningInfos {
		address, err := k.sk.ConsensusAddressCodec().StringToBytes(info.Address)
		if err != nil {
			panic(err)
		}

		err = k.SetValidatorSigningInfo(ctx, address, info.ValidatorSigningInfo)
		if err != nil {
			panic(err)
		}
	}

	for _, array := range data.MissedBlocks {
		address, err := k.sk.ConsensusAddressCodec().StringToBytes(array.Address)
		if err != nil {
			panic(err)
		}

		for _, missed := range array.MissedBlocks {
			if err := k.SetMissedBlockBitmapValue(ctx, address, missed.Index, missed.Missed); err != nil {
				panic(err)
			}
		}
	}

	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func (k Keeper) ExportGenesis(ctx sdk.Context) (data *types.GenesisState) {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	signingInfos := make([]types.SigningInfo, 0)
	missedBlocks := make([]types.ValidatorMissedBlocks, 0)
	err = k.IterateValidatorSigningInfos(ctx, func(address sdk.ConsAddress, info types.ValidatorSigningInfo) (stop bool) {
		bechAddr := address.String()
		signingInfos = append(signingInfos, types.SigningInfo{
			Address:              bechAddr,
			ValidatorSigningInfo: info,
		})

		localMissedBlocks, err := k.GetValidatorMissedBlocks(ctx, address)
		if err != nil {
			panic(err)
		}

		missedBlocks = append(missedBlocks, types.ValidatorMissedBlocks{
			Address:      bechAddr,
			MissedBlocks: localMissedBlocks,
		})

		return false
	})
	if err != nil {
		panic(err)
	}

	return types.NewGenesisState(params, signingInfos, missedBlocks)
}
