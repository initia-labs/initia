package keepers

import (
	movehooktypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"
)

func (appKeepers *AppKeepers) makeQueryWhitelist() movetypes.VMQueryWhiteList {
	queryWhitelist := movetypes.DefaultVMQueryWhiteList(appKeepers.ac)

	// stargate queries

	queryWhitelist.Stargate["/initia.mstaking.v1.Query/UnbondingDelegation"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryUnbondingDelegationRequest{},
		Response: &stakingtypes.QueryUnbondingDelegationResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/Pool"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryPoolRequest{},
		Response: &stakingtypes.QueryPoolResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/DelegatorDelegations"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryDelegatorDelegationsRequest{},
		Response: &stakingtypes.QueryDelegatorDelegationsResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/DelegatorTotalDelegationBalance"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryDelegatorTotalDelegationBalanceRequest{},
		Response: &stakingtypes.QueryDelegatorTotalDelegationBalanceResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/Delegation"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryDelegationRequest{},
		Response: &stakingtypes.QueryDelegationResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/Redelegations"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryRedelegationsRequest{},
		Response: &stakingtypes.QueryRedelegationsResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetAllCurrencyPairs"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetAllCurrencyPairsRequest{},
		Response: &oracletypes.GetAllCurrencyPairsResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetPrice"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetPriceRequest{},
		Response: &oracletypes.GetPriceResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetPrices"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetPricesRequest{},
		Response: &oracletypes.GetPricesResponse{},
	}

	// custom queries

	queryWhitelist.Custom[movehooktypes.VMCustomQueryTransferFunds] = appKeepers.IBCHooksKeeper.QueryTransferFunds
	return queryWhitelist
}
