package app

import (
	"encoding/json"

	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icagenesistypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/genesis/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	l2connect "github.com/initia-labs/OPinit/x/opchild/l2connect"
	"github.com/initia-labs/initia/app/genesis_markets"
	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	customgovtypes "github.com/initia-labs/initia/x/gov/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"

	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"
)

// GenesisState - The genesis state of the blockchain is represented here as a map of raw json
// messages key'd by a identifier string.
// The identifier is used to determine which module genesis information belongs
// to so it may be appropriately routed during init chain.
// Within this application default genesis information is retrieved from
// the ModuleBasicManager which populates json from each BasicModule
// object provided to it during init.
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(cdc codec.Codec, bondDenom string) GenesisState {
	return GenesisState(BasicManager().DefaultGenesis(cdc)).
		ConfigureBondDenom(cdc, bondDenom).
		ConfigureICA(cdc).
		AddMarketData(cdc, cdc.InterfaceRegistry().SigningContext().AddressCodec())
}

// ConfigureBondDenom generates the default state for the application.
func (genState GenesisState) ConfigureBondDenom(cdc codec.JSONCodec, bondDenom string) GenesisState {
	// customize bond denom
	var stakingGenState stakingtypes.GenesisState
	cdc.MustUnmarshalJSON(genState[stakingtypes.ModuleName], &stakingGenState)
	stakingGenState.Params.BondDenoms = []string{bondDenom}
	genState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(&stakingGenState)

	var distrGenState customdistrtypes.GenesisState
	cdc.MustUnmarshalJSON(genState[distrtypes.ModuleName], &distrGenState)
	distrGenState.Params.RewardWeights = []customdistrtypes.RewardWeight{{Denom: bondDenom, Weight: math.LegacyOneDec()}}
	genState[distrtypes.ModuleName] = cdc.MustMarshalJSON(&distrGenState)

	var crisisGenState crisistypes.GenesisState
	cdc.MustUnmarshalJSON(genState[crisistypes.ModuleName], &crisisGenState)
	crisisGenState.ConstantFee.Denom = bondDenom
	genState[crisistypes.ModuleName] = cdc.MustMarshalJSON(&crisisGenState)

	var govGenState customgovtypes.GenesisState
	cdc.MustUnmarshalJSON(genState[govtypes.ModuleName], &govGenState)
	govGenState.Params.MinDeposit[0].Denom = bondDenom
	govGenState.Params.ExpeditedMinDeposit[0].Denom = bondDenom
	govGenState.Params.EmergencyMinDeposit[0].Denom = bondDenom
	genState[govtypes.ModuleName] = cdc.MustMarshalJSON(&govGenState)

	var rewardGenState rewardtypes.GenesisState
	cdc.MustUnmarshalJSON(genState[rewardtypes.ModuleName], &rewardGenState)
	rewardGenState.Params.RewardDenom = bondDenom
	genState[rewardtypes.ModuleName] = cdc.MustMarshalJSON(&rewardGenState)

	var moveGenState movetypes.GenesisState
	cdc.MustUnmarshalJSON(genState[movetypes.ModuleName], &moveGenState)
	moveGenState.Params.BaseDenom = bondDenom
	genState[movetypes.ModuleName] = cdc.MustMarshalJSON(&moveGenState)

	return genState
}

func (genState GenesisState) AddMarketData(cdc codec.JSONCodec, ac address.Codec) GenesisState {
	var oracleGenState oracletypes.GenesisState
	cdc.MustUnmarshalJSON(genState[oracletypes.ModuleName], &oracleGenState)

	var marketGenState marketmaptypes.GenesisState
	cdc.MustUnmarshalJSON(genState[marketmaptypes.ModuleName], &marketGenState)

	// Load initial markets
	markets, err := genesis_markets.ReadMarketsFromFile(genesis_markets.GenesisMarkets)
	if err != nil {
		panic(err)
	}
	marketGenState.MarketMap = genesis_markets.ToMarketMap(markets)

	var id uint64

	// Initialize all markets plus ReservedCPTimestamp
	currencyPairGenesis := make([]oracletypes.CurrencyPairGenesis, len(markets)+1)
	cp, err := connecttypes.CurrencyPairFromString(l2connect.ReservedCPTimestamp)
	if err != nil {
		panic(err)
	}
	currencyPairGenesis[id] = oracletypes.CurrencyPairGenesis{
		CurrencyPair:      cp,
		CurrencyPairPrice: nil,
		Nonce:             0,
		Id:                id,
	}
	id++
	for i, market := range markets {
		currencyPairGenesis[i+1] = oracletypes.CurrencyPairGenesis{
			CurrencyPair:      market.Ticker.CurrencyPair,
			CurrencyPairPrice: nil,
			Nonce:             0,
			Id:                id,
		}
		id++
	}

	oracleGenState.CurrencyPairGenesis = currencyPairGenesis
	oracleGenState.NextId = id

	// write the updates to genState
	genState[marketmaptypes.ModuleName] = cdc.MustMarshalJSON(&marketGenState)
	genState[oracletypes.ModuleName] = cdc.MustMarshalJSON(&oracleGenState)
	return genState
}

func (genState GenesisState) ConfigureICA(cdc codec.JSONCodec) GenesisState {
	// create ICS27 Controller submodule params
	controllerParams := icacontrollertypes.Params{
		ControllerEnabled: true,
	}

	// create ICS27 Host submodule params
	hostParams := icahosttypes.Params{
		HostEnabled: true,
		AllowMessages: []string{
			authzMsgExec,
			authzMsgGrant,
			authzMsgRevoke,
			bankMsgSend,
			bankMsgMultiSend,
			distrMsgSetWithdrawAddr,
			distrMsgWithdrawValidatorCommission,
			distrMsgFundCommunityPool,
			distrMsgWithdrawDelegatorReward,
			feegrantMsgGrantAllowance,
			feegrantMsgRevokeAllowance,
			govMsgVoteWeighted,
			govMsgSubmitProposal,
			govMsgDeposit,
			govMsgVote,
			groupCreateGroup,
			groupCreateGroupPolicy,
			groupExec,
			groupLeaveGroup,
			groupSubmitProposal,
			groupUpdateGroupAdmin,
			groupUpdateGroupMember,
			groupUpdateGroupPolicyAdmin,
			groupUpdateGroupPolicyDecisionPolicy,
			groupVote,
			groupWithdrawProposal,
			stakingMsgEditValidator,
			stakingMsgDelegate,
			stakingMsgUndelegate,
			stakingMsgBeginRedelegate,
			stakingMsgCreateValidator,
			transferMsgTransfer,
			nftTransferMsgTransfer,
			moveMsgPublishModuleBundle,
			moveMsgExecuteEntryFunction,
			moveMsgExecuteScript,
		},
	}

	var icaGenState icagenesistypes.GenesisState
	cdc.MustUnmarshalJSON(genState[icatypes.ModuleName], &icaGenState)
	icaGenState.ControllerGenesisState.Params = controllerParams
	icaGenState.HostGenesisState.Params = hostParams
	genState[icatypes.ModuleName] = cdc.MustMarshalJSON(&icaGenState)

	return genState
}
