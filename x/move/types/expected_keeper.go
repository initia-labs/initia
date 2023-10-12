package types

import (
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"
)

// AccountKeeper is expected keeper for auth module
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	SetAccount(ctx sdk.Context, acc authtypes.AccountI)
	HasAccount(ctx sdk.Context, addr sdk.AccAddress) bool

	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
	SetModuleAccount(ctx sdk.Context, macc authtypes.ModuleAccountI)

	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	NextAccountNumber(ctx sdk.Context) uint64
}

// BankViewKeeper defines a subset of methods implemented by the cosmos-sdk bank keeper
type BankViewKeeper interface {
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// BankKeeper defines a subset of methods implemented by the cosmos-sdk bank keeper
type BankKeeper interface {
	BankViewKeeper
	IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error
	BlockedAddr(addr sdk.AccAddress) bool
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
}

type RewardKeeper interface {
	GetParams(ctx sdk.Context) (params rewardtypes.Params)
}

// StakingKeeper is expected keeper for staking module
type StakingKeeper interface {
	Validator(ctx sdk.Context, address sdk.ValAddress) stakingtypes.ValidatorI
	GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator
	UnbondingTime(ctx sdk.Context) time.Duration
	Delegate(ctx sdk.Context, delAddr sdk.AccAddress, bondAmt sdk.Coins, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool) (sdk.DecCoins, error)
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (stakingtypes.Validator, bool)
	Unbond(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares sdk.DecCoins) (amount sdk.Coins, err error)
	Delegation(sdk.Context, sdk.AccAddress, sdk.ValAddress) stakingtypes.DelegationI
	BondDenoms(ctx sdk.Context) (res []string)
	SetBondDenoms(ctx sdk.Context, bondDenoms []string) error
}

// DistributionKeeper is expected keeper for distribution module
type DistributionKeeper interface {
	WithdrawDelegationRewards(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (distrtypes.Pools, error)

	// increment validator period, returning the period just ended
	IncrementValidatorPeriod(ctx sdk.Context, val stakingtypes.ValidatorI) uint64
	// calculate the total rewards accrued by a delegation
	CalculateDelegationRewards(ctx sdk.Context, val stakingtypes.ValidatorI, del stakingtypes.DelegationI, endingPeriod uint64) (rewards distrtypes.DecPools)

	GetRewardWeights(ctx sdk.Context) (rewardWeights []distrtypes.RewardWeight)
	SetRewardWeights(ctx sdk.Context, rewardWeights []distrtypes.RewardWeight)
}

type CommunityPoolKeeper interface {
	// FundCommunityPool allows an account to directly fund the community fund pool.
	FundCommunityPool(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error
}

type TransferKeeper interface {
	GetDenomTrace(ctx sdk.Context, denomTraceHash tmbytes.HexBytes) (transfertypes.DenomTrace, bool)
}

type NftTransferKeeper interface {
	GetClassTrace(ctx sdk.Context, classTraceHash tmbytes.HexBytes) (nfttransfertypes.ClassTrace, bool)
}

// MessageRouter ADR 031 request type routing
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}
