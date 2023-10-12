package app

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icagenesistypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/genesis/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"

	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
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
func NewDefaultGenesisState(cdc codec.JSONCodec) GenesisState {
	return ModuleBasics.DefaultGenesis(cdc)
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
	distrGenState.Params.RewardWeights = []customdistrtypes.RewardWeight{{Denom: bondDenom, Weight: sdk.OneDec()}}
	genState[distrtypes.ModuleName] = cdc.MustMarshalJSON(&distrGenState)

	var crisisGenState crisistypes.GenesisState
	cdc.MustUnmarshalJSON(genState[crisistypes.ModuleName], &crisisGenState)
	crisisGenState.ConstantFee.Denom = bondDenom
	genState[crisistypes.ModuleName] = cdc.MustMarshalJSON(&crisisGenState)

	var govGenState govtypesv1.GenesisState
	cdc.MustUnmarshalJSON(genState[govtypes.ModuleName], &govGenState)
	govGenState.Params.MinDeposit[0].Denom = bondDenom
	genState[govtypes.ModuleName] = cdc.MustMarshalJSON(&govGenState)

	var rewardGenState rewardtypes.GenesisState
	cdc.MustUnmarshalJSON(genState[rewardtypes.ModuleName], &rewardGenState)
	rewardGenState.Params.RewardDenom = bondDenom
	genState[rewardtypes.ModuleName] = cdc.MustMarshalJSON(&rewardGenState)

	var moveGenState movetypes.GenesisState
	cdc.MustUnmarshalJSON(genState[movetypes.ModuleName], &moveGenState)
	moveGenState.Params.BaseDenom = bondDenom
	genState[movetypes.ModuleName] = cdc.MustMarshalJSON(&moveGenState)

	// Builder module genesis-state bond denom configuration
	var builderGenState buildertypes.GenesisState
	cdc.MustUnmarshalJSON(genState[buildertypes.ModuleName], &builderGenState)
	builderGenState.Params.ReserveFee.Denom = bondDenom
	builderGenState.Params.MinBidIncrement.Denom = bondDenom
	genState[buildertypes.ModuleName] = cdc.MustMarshalJSON(&builderGenState)

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
			sftTransferMsgTransfer,
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
