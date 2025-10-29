package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/distribution/keeper"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func TestQueryParams(t *testing.T) {
	ctx, testKp := createDefaultTestInput(t)
	queryServer := keeper.NewQueryServer(testKp.DistKeeper)

	cases := []struct {
		name   string
		req    *types.QueryParamsRequest
		resp   *types.QueryParamsResponse
		errMsg string
	}{
		{
			name: "success",
			req:  &types.QueryParamsRequest{},
			resp: &types.QueryParamsResponse{
				Params: types.DefaultParams(),
			},
			errMsg: "",
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			out, err := queryServer.Params(ctx, (*types.QueryParamsRequest)(tc.req))
			if tc.errMsg == "" {
				require.NoError(t, err)
				require.Equal(t, tc.resp, out)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, out)
			}
		})
	}
}

func TestQueryValidatorDistributionInfo(t *testing.T) {
	ctx, testKp := createDefaultTestInput(t)
	queryServer := keeper.NewQueryServer(testKp.DistKeeper)
	// operator, err := codectestutil.CodecOptions{}.NewInterfaceRegistry().SigningContext().ValidatorAddressCodec().BytesToString(operatorAddr)

	vali, operator := createValidatorAndOperatorWithBalance(ctx, testKp, 100, 100, 1)
	cases := []struct {
		name   string
		req    *types.QueryValidatorDistributionInfoRequest
		resp   *types.QueryValidatorDistributionInfoResponse
		errMsg string
	}{
		{
			name: "invalid validator address",
			req: &types.QueryValidatorDistributionInfoRequest{
				ValidatorAddress: "invalid address",
			},
			resp:   &types.QueryValidatorDistributionInfoResponse{},
			errMsg: "decoding bech32 failed",
		},
		{
			name: "validator",
			req: &types.QueryValidatorDistributionInfoRequest{
				ValidatorAddress: vali.String(),
			},
			resp: &types.QueryValidatorDistributionInfoResponse{
				OperatorAddress: operator.String(),
			},
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			out, err := queryServer.ValidatorDistributionInfo(ctx, tc.req)
			if tc.errMsg == "" {
				require.NoError(t, err)
				require.Equal(t, tc.resp, out)
			} else {
				require.Error(t, err)
			}
		})
	}
}
