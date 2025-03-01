package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
)

// InitGenesis sets distribution information for genesis
func (k Keeper) InitGenesis(ctx sdk.Context, data customtypes.GenesisState) {
	var moduleHoldings sdk.DecCoins

	if err := k.FeePool.Set(ctx, data.FeePool); err != nil {
		panic(err)
	}
	if err := k.Params.Set(ctx, data.Params); err != nil {
		panic(err)
	}

	for _, dwi := range data.DelegatorWithdrawInfos {
		delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(dwi.DelegatorAddress)
		if err != nil {
			panic(err)
		}
		withdrawAddress, err := k.authKeeper.AddressCodec().StringToBytes(dwi.WithdrawAddress)
		if err != nil {
			panic(err)
		}

		if err := k.DelegatorWithdrawAddrs.Set(ctx, delegatorAddress, withdrawAddress); err != nil {
			panic(err)
		}
	}

	var previousProposer sdk.ConsAddress
	if data.PreviousProposer != "" {
		var err error
		previousProposer, err = sdk.ConsAddressFromBech32(data.PreviousProposer)
		if err != nil {
			panic(err)
		}
	}

	if err := k.PreviousProposerConsAddr.Set(ctx, previousProposer); err != nil {
		panic(err)
	}

	for _, rew := range data.OutstandingRewards {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(rew.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		if err := k.ValidatorOutstandingRewards.Set(ctx, valAddr, customtypes.ValidatorOutstandingRewards{Rewards: rew.OutstandingRewards}); err != nil {
			panic(err)
		}
		moduleHoldings = moduleHoldings.Add(rew.OutstandingRewards.Sum()...)
	}
	for _, acc := range data.ValidatorAccumulatedCommissions {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(acc.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		if err := k.ValidatorAccumulatedCommissions.Set(ctx, valAddr, acc.Accumulated); err != nil {
			panic(err)
		}
	}
	for _, his := range data.ValidatorHistoricalRewards {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(his.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		if err := k.ValidatorHistoricalRewards.Set(ctx, collections.Join(valAddr, his.Period), his.Rewards); err != nil {
			panic(err)
		}
	}
	for _, cur := range data.ValidatorCurrentRewards {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(cur.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		if err := k.ValidatorCurrentRewards.Set(ctx, valAddr, cur.Rewards); err != nil {
			panic(err)
		}
	}
	for _, del := range data.DelegatorStartingInfos {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(del.DelegatorAddress)
		if err != nil {
			panic(err)
		}
		if err := k.DelegatorStartingInfos.Set(ctx, collections.Join(valAddr, delegatorAddress), del.StartingInfo); err != nil {
			panic(err)
		}
	}
	for _, evt := range data.ValidatorSlashEvents {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(evt.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		if err := k.ValidatorSlashEvents.Set(ctx, collections.Join3(valAddr, evt.Height, evt.Period), evt.ValidatorSlashEvent); err != nil {
			panic(err)
		}
	}

	moduleHoldings = moduleHoldings.Add(data.FeePool.CommunityPool...)
	moduleHoldingsInt, _ := moduleHoldings.TruncateDecimal()

	// check if the module account exists
	moduleAcc := k.GetDistributionAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	balances := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if balances.IsZero() {
		k.authKeeper.SetModuleAccount(ctx, moduleAcc)
	}

	// It is hard to block fungible asset transfer from move side,
	// so we can't guarantee the ModuleAccount is always equal to the sum
	// of the distribution rewards and community pool. Instead, we decide to check
	// if the ModuleAccount balance is greater than or equal to the sum of
	// the distribution rewards and community pool.
	broken := !balances.IsAllGTE(moduleHoldingsInt)
	if broken {
		panic(fmt.Sprintf("distribution module balance does not match the module holdings: %s <-> %s", balances, moduleHoldingsInt))
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k Keeper) ExportGenesis(ctx sdk.Context) *customtypes.GenesisState {
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		panic(err)
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	dwi := make([]types.DelegatorWithdrawInfo, 0)
	err = k.DelegatorWithdrawAddrs.Walk(ctx, nil, func(delAddr []byte, addr []byte) (stop bool, err error) {
		delAddrStr, err := k.authKeeper.AddressCodec().BytesToString(delAddr)
		if err != nil {
			return false, err
		}
		addrStr, err := k.authKeeper.AddressCodec().BytesToString(addr)
		if err != nil {
			return false, err
		}

		dwi = append(dwi, types.DelegatorWithdrawInfo{
			DelegatorAddress: delAddrStr,
			WithdrawAddress:  addrStr,
		})

		return false, nil
	})
	if err != nil {
		panic(err)
	}

	pp, err := k.PreviousProposerConsAddr.Get(ctx)
	if err != nil {
		panic(err)
	}

	outstanding := make([]customtypes.ValidatorOutstandingRewardsRecord, 0)
	err = k.ValidatorOutstandingRewards.Walk(ctx, nil,
		func(valAddr []byte, rewards customtypes.ValidatorOutstandingRewards) (stop bool, err error) {
			valAddrStr, err := k.stakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
			if err != nil {
				return false, err
			}

			outstanding = append(outstanding, customtypes.ValidatorOutstandingRewardsRecord{
				ValidatorAddress:   valAddrStr,
				OutstandingRewards: rewards.Rewards,
			})

			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	acc := make([]customtypes.ValidatorAccumulatedCommissionRecord, 0)
	err = k.ValidatorAccumulatedCommissions.Walk(ctx, nil,
		func(valAddr []byte, commission customtypes.ValidatorAccumulatedCommission) (stop bool, err error) {
			valAddrStr, err := k.stakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
			if err != nil {
				return false, err
			}

			acc = append(acc, customtypes.ValidatorAccumulatedCommissionRecord{
				ValidatorAddress: valAddrStr,
				Accumulated:      commission,
			})
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	his := make([]customtypes.ValidatorHistoricalRewardsRecord, 0)
	err = k.ValidatorHistoricalRewards.Walk(ctx, nil,
		func(key collections.Pair[[]byte, uint64], rewards customtypes.ValidatorHistoricalRewards) (stop bool, err error) {
			valAddrStr, err := k.stakingKeeper.ValidatorAddressCodec().BytesToString(key.K1())
			if err != nil {
				return false, err
			}

			his = append(his, customtypes.ValidatorHistoricalRewardsRecord{
				ValidatorAddress: valAddrStr,
				Period:           key.K2(),
				Rewards:          rewards,
			})
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	cur := make([]customtypes.ValidatorCurrentRewardsRecord, 0)
	err = k.ValidatorCurrentRewards.Walk(ctx, nil,
		func(valAddr []byte, rewards customtypes.ValidatorCurrentRewards) (stop bool, err error) {
			valAddrStr, err := k.stakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
			if err != nil {
				return false, err
			}

			cur = append(cur, customtypes.ValidatorCurrentRewardsRecord{
				ValidatorAddress: valAddrStr,
				Rewards:          rewards,
			})

			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	dels := make([]customtypes.DelegatorStartingInfoRecord, 0)
	err = k.DelegatorStartingInfos.Walk(ctx, nil,
		func(key collections.Pair[[]byte, []byte], info customtypes.DelegatorStartingInfo) (stop bool, err error) {
			valAddrStr, err := k.stakingKeeper.ValidatorAddressCodec().BytesToString(key.K1())
			if err != nil {
				return false, err
			}
			delAddrStr, err := k.authKeeper.AddressCodec().BytesToString(key.K2())
			if err != nil {
				return false, err
			}

			dels = append(dels, customtypes.DelegatorStartingInfoRecord{
				ValidatorAddress: valAddrStr,
				DelegatorAddress: delAddrStr,
				StartingInfo:     info,
			})
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	slashes := make([]customtypes.ValidatorSlashEventRecord, 0)
	err = k.ValidatorSlashEvents.Walk(ctx, nil,
		func(key collections.Triple[[]byte, uint64, uint64], event customtypes.ValidatorSlashEvent) (stop bool, err error) {
			valAddrStr, err := k.stakingKeeper.ValidatorAddressCodec().BytesToString(key.K1())
			if err != nil {
				return false, err
			}

			slashes = append(slashes, customtypes.ValidatorSlashEventRecord{
				ValidatorAddress:    valAddrStr,
				Height:              key.K2(),
				Period:              key.K3(),
				ValidatorSlashEvent: event,
			})

			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	return customtypes.NewGenesisState(params, feePool, dwi, pp, outstanding, acc, his, cur, dels, slashes)
}
