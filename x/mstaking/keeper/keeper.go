package keeper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/initia-labs/initia/x/mstaking/types"

	"cosmossdk.io/collections"
	collectioncodec "cosmossdk.io/collections/codec"
	addresscodec "cosmossdk.io/core/address"
	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// Implements ValidatorSet interface
var _ types.ValidatorSet = Keeper{}

// Implements DelegationSet interface
var _ types.DelegationSet = Keeper{}

// keeper of the staking store
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService corestoretypes.KVStoreService

	authKeeper          types.AccountKeeper
	bankKeeper          types.BankKeeper
	moveKeeper          types.MoveKeeper
	fungibleAssetKeeper types.FungibleAssetKeeper
	balancerKeeper      types.BalancerKeeper
	VotingPowerKeeper   types.VotingPowerKeeper
	hooks               types.StakingHooks
	slashingHooks       types.SlashingHooks

	authority string

	validatorAddressCodec addresscodec.Codec
	consensusAddressCodec addresscodec.Codec

	Schema collections.Schema

	LastValidatorConsPowers    collections.Map[[]byte, int64]
	WhitelistedValidators      collections.Map[[]byte, bool]
	Validators                 collections.Map[[]byte, types.Validator]
	ValidatorsByConsAddr       collections.Map[[]byte, []byte]
	ValidatorsByConsPowerIndex collections.Map[collections.Pair[int64, []byte], bool]

	Delegations                    collections.Map[collections.Pair[[]byte, []byte], types.Delegation]             // delAddr, valAddr
	DelegationsByValIndex          collections.Map[collections.Pair[[]byte, []byte], bool]                         // valAddr, delAddr
	UnbondingDelegations           collections.Map[collections.Pair[[]byte, []byte], types.UnbondingDelegation]    // delAddr, valAddr
	UnbondingDelegationsByValIndex collections.Map[collections.Pair[[]byte, []byte], bool]                         // valAddr, delAddr
	Redelegations                  collections.Map[collections.Triple[[]byte, []byte, []byte], types.Redelegation] // delAddr, valSrcAddr, valDstAddr
	RedelegationsByValSrcIndex     collections.Map[collections.Triple[[]byte, []byte, []byte], bool]               // valSrcAddr, delAddr, valDstAddr
	RedelegationsByValDstIndex     collections.Map[collections.Triple[[]byte, []byte, []byte], bool]               // valDstAddr, delAddr, valSrcAddr

	NextUnbondingId collections.Sequence
	UnbondingsIndex collections.Map[uint64, collections.Triple[[]byte, []byte, []byte]]
	UnbondingsType  collections.Map[uint64, uint32]

	UnbondingQueue    collections.Map[time.Time, types.DVPairs]
	RedelegationQueue collections.Map[time.Time, types.DVVTriplets]
	ValidatorQueue    collections.Map[time.Time, types.ValAddresses]

	HistoricalInfos collections.Map[int64, cosmostypes.HistoricalInfo]

	RegisteredMigrations collections.Map[[]byte, types.DelegationMigration]

	Params collections.Item[types.Params]
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	mk types.MoveKeeper,
	fak types.FungibleAssetKeeper,
	balk types.BalancerKeeper,
	vk types.VotingPowerKeeper,
	authority string,
	validatorAddressCodec addresscodec.Codec,
	consensusAddressCodec addresscodec.Codec,
) *Keeper {
	// ensure bonded and not bonded module accounts are set
	if addr := ak.GetModuleAddress(types.BondedPoolName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.BondedPoolName))
	}

	if addr := ak.GetModuleAddress(types.NotBondedPoolName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.NotBondedPoolName))
	}

	// ensure that authority is a valid AccAddress
	if _, err := ak.AddressCodec().StringToBytes(authority); err != nil {
		panic("authority is not a valid acc address")
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := &Keeper{
		cdc:                 cdc,
		storeService:        storeService,
		authKeeper:          ak,
		bankKeeper:          bk,
		moveKeeper:          mk,
		fungibleAssetKeeper: fak,
		balancerKeeper:      balk,
		VotingPowerKeeper:   vk,
		hooks:               nil,
		slashingHooks:       nil,

		authority: authority,

		validatorAddressCodec: validatorAddressCodec,
		consensusAddressCodec: consensusAddressCodec,

		LastValidatorConsPowers:    collections.NewMap(sb, types.LastValidatorConsPowersPrefix, "last_validator_cons_powers", collections.BytesKey, collections.Int64Value),
		WhitelistedValidators:      collections.NewMap(sb, types.WhitelistedValidatorsPrefix, "whitelisted_validators", collections.BytesKey, collections.BoolValue),
		Validators:                 collections.NewMap(sb, types.ValidatorsPrefix, "validators", collections.BytesKey, codec.CollValue[types.Validator](cdc)),
		ValidatorsByConsAddr:       collections.NewMap(sb, types.ValidatorsByConsAddrPrefix, "validators_by_cons_addr", collections.BytesKey, collections.BytesValue),
		ValidatorsByConsPowerIndex: collections.NewMap(sb, types.ValidatorsByConsPowerIndexPrefix, "validators_by_cons_power_index_prefix", collections.PairKeyCodec(collections.Int64Key, collections.BytesKey), collections.BoolValue),

		Delegations:                    collections.NewMap(sb, types.DelegationsPrefix, "delegations", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), codec.CollValue[types.Delegation](cdc)),
		DelegationsByValIndex:          collections.NewMap(sb, types.DelegationsByValIndexPrefix, "delegations_by_val_index", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), collections.BoolValue),
		UnbondingDelegations:           collections.NewMap(sb, types.UnbondingDelegationsPrefix, "unbonding_delegations", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), codec.CollValue[types.UnbondingDelegation](cdc)),
		UnbondingDelegationsByValIndex: collections.NewMap(sb, types.UnbondingDelegationsByValIndexPrefix, "unbonding_delegations_by_val_index", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), collections.BoolValue),
		Redelegations:                  collections.NewMap(sb, types.RedelegationsPrefix, "redelegations", collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.BytesKey), codec.CollValue[types.Redelegation](cdc)),
		RedelegationsByValSrcIndex:     collections.NewMap(sb, types.RedelegationsByValSrcIndexPrefix, "redelegations_by_val_src_index", collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.BytesKey), collections.BoolValue),
		RedelegationsByValDstIndex:     collections.NewMap(sb, types.RedelegationsByValDstIndexPrefix, "redelegations_by_val_dst_index", collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.BytesKey), collections.BoolValue),

		NextUnbondingId: collections.NewSequence(sb, types.NextUnbondingIdKey, "next_unbonding_id"),
		UnbondingsIndex: collections.NewMap(sb, types.UnbondingsIndexPrefix, "unbondings_index", collections.Uint64Key, collectioncodec.KeyToValueCodec(collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.BytesKey))),
		UnbondingsType:  collections.NewMap(sb, types.UnbondingsTypePrefix, "unbondings_type", collections.Uint64Key, collections.Uint32Value),

		UnbondingQueue:    collections.NewMap(sb, types.UnbondingQueuePrefix, "unbonding_queue", sdk.TimeKey, codec.CollValue[types.DVPairs](cdc)),
		RedelegationQueue: collections.NewMap(sb, types.RedelegationQueuePrefix, "redelegation_queue", sdk.TimeKey, codec.CollValue[types.DVVTriplets](cdc)),
		ValidatorQueue:    collections.NewMap(sb, types.ValidatorQueuePrefix, "validator_queue", sdk.TimeKey, codec.CollValue[types.ValAddresses](cdc)),

		HistoricalInfos: collections.NewMap(sb, types.HistoricalInfosPrefix, "historical_infos", collections.Int64Key, codec.CollValue[cosmostypes.HistoricalInfo](cdc)),

		RegisteredMigrations: collections.NewMap(sb, types.RegisteredMigrationsPrefix, "registered_migrations", collections.BytesKey, codec.CollValue[types.DelegationMigration](cdc)),

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// Hooks gets the hooks for staking *Keeper {
func (k *Keeper) Hooks() types.StakingHooks {
	if k.hooks == nil {
		// return a no-op implementation if no hooks are set
		return types.MultiStakingHooks{}
	}

	return k.hooks
}

// Set the staking hooks
func (k *Keeper) SetHooks(sh types.StakingHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set staking hooks twice")
	}

	k.hooks = sh

	return k
}

// SlashingHooks gets the slashing hooks for staking *Keeper {
func (k *Keeper) SlashingHooks() types.SlashingHooks {
	if k.slashingHooks == nil {
		// return a no-op implementation if no hooks are set
		return types.MultiSlashingHooks{}
	}

	return k.slashingHooks
}

// Set the slashing hooks
func (k *Keeper) SetSlashingHooks(sh types.SlashingHooks) *Keeper {
	if k.slashingHooks != nil {
		panic("cannot set slashing hooks twice")
	}

	k.slashingHooks = sh

	return k
}

// GetAuthority returns the x/staking module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ValidatorAddressCodec returns the app validator address codec.
func (k Keeper) ValidatorAddressCodec() addresscodec.Codec {
	return k.validatorAddressCodec
}

// ConsensusAddressCodec returns the app consensus address codec.
func (k Keeper) ConsensusAddressCodec() addresscodec.Codec {
	return k.consensusAddressCodec
}

// RegisterMigration registers a migration of a delegation from one lp denom to another.
// Swap contract requires the following functions:
// - initialize(account: &signer, coin_in: Object<Metadata>, coin_out: Object<Metadata>)
// - provide_liquidity(account: &signer, coin: Object<Metadata>, amount: u64)
// - denom_out(denom_in: String) -> String
// - swap(account: &signer, coin_in: Object<Metadata>, coin_out: Object<Metadata>, amount: u64)
func (k Keeper) RegisterMigration(ctx context.Context, lpDenomIn string, lpDenomOut string, denomIn string, denomOut string, swapContractStr string) error {
	lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenomIn)
	if err != nil {
		return err
	}

	hasPool, err := k.balancerKeeper.HasPool(ctx, lpMetadataIn)
	if err != nil {
		return err
	} else if !hasPool {
		return fmt.Errorf("lp metadata is not found in balancer")
	}

	lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenomOut)
	if err != nil {
		return err
	}

	hasPool, err = k.balancerKeeper.HasPool(ctx, lpMetadataOut)
	if err != nil {
		return err
	} else if !hasPool {
		return fmt.Errorf("lp metadata is not found in balancer")
	}

	swapContract := strings.Split(swapContractStr, "::")
	if len(swapContract) != 2 {
		return fmt.Errorf("invalid swap contract address: %s, expected format: <module_addr>::<module_name>", swapContractStr)
	}

	swapContractModuleAddress, err := movetypes.AccAddressFromString(k.authKeeper.AddressCodec(), swapContract[0])
	if err != nil {
		return err
	}

	output, _, err := k.moveKeeper.ExecuteViewFunctionJSON(
		ctx,
		vmtypes.AccountAddress(swapContractModuleAddress),
		swapContract[1],
		"denom_out",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", denomIn)},
	)
	if err != nil {
		return err
	}

	wrappedDenomOut := fmt.Sprintf("\"%s\"", denomOut)
	if output.Ret != wrappedDenomOut {
		return fmt.Errorf("invalid denom out: got %s, expected %s", output.Ret, wrappedDenomOut)
	}

	// even if the migration is already registered, it will be overwritten
	err = k.RegisteredMigrations.Set(ctx, lpMetadataIn[:], types.DelegationMigration{
		LpMetadataOut:             lpMetadataOut[:],
		SwapContractModuleAddress: swapContractModuleAddress[:],
		SwapContractModuleName:    swapContract[1],
	})
	if err != nil {
		return err
	}
	return nil
}
