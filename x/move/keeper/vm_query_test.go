package keeper_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	govtypes "github.com/initia-labs/initia/x/gov/types"
	"github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_VMQuery_Stargate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// not existing
	_, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Stargate: &vmtypes.StargateQuery{
			Path: "/initia.gov.v1.Query/Proposals",
			Data: nil,
		},
	})
	require.ErrorContains(t, err, types.ErrNotSupportedStargateQuery.Error())

	proposal := govtypes.Proposal{
		Id:      1,
		Title:   "title",
		Summary: "summary",
		Status:  govtypesv1.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
	}

	// set Proposal
	err = input.GovKeeper.SetProposal(ctx, proposal)
	require.NoError(t, err)

	// query proposal
	resBz, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Stargate: &vmtypes.StargateQuery{
			Path: "/initia.gov.v1.Query/Proposal",
			Data: []byte(`{"proposal_id": "1"}`),
		},
	})
	require.NoError(t, err)

	// expected proposal res json bytes
	expectedResBz, err := input.EncodingConfig.Codec.MarshalJSON(&govtypes.QueryProposalResponse{
		Proposal: &proposal,
	})
	require.NoError(t, err)
	require.Equal(t, expectedResBz, resBz)
}

func Test_VMQuery_Custom(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// not existing
	_, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Custom: &vmtypes.CustomQuery{
			Name: "aaa",
			Data: nil,
		},
	})
	require.ErrorContains(t, err, types.ErrNotSupportedCustomQuery.Error())

	reqBz, err := json.Marshal(types.ToSDKAddressRequest{
		VMAddr: vmtypes.StdAddress.String(),
	})
	require.NoError(t, err)

	resBz, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Custom: &vmtypes.CustomQuery{
			Name: "to_sdk_address",
			Data: reqBz,
		},
	})
	require.NoError(t, err)

	var res types.ToSDKAddressResponse
	err = json.Unmarshal(resBz, &res)
	require.NoError(t, err)
	require.Equal(t, types.StdAddr[12:].String(), res.SDKAddr)
}
