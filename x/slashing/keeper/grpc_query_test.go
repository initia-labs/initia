package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"

	"github.com/stretchr/testify/require"
)

func TestGRPCQueryParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	paramsResp, err := input.SlashingKeeper.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), paramsResp.Params)
}

func TestGRPCSigningInfo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	infoResp, err := input.SlashingKeeper.SigningInfo(ctx, &types.QuerySigningInfoRequest{ConsAddress: ""})
	require.Error(t, err)
	require.Nil(t, infoResp)

	_, valPubKey := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 2_000_000, 1)

	info, err := input.SlashingKeeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(valPubKey.Address()))
	require.NoError(t, err)

	infoResp, err = input.SlashingKeeper.SigningInfo(ctx,
		&types.QuerySigningInfoRequest{ConsAddress: sdk.ConsAddress(valPubKey.Address()).String()})
	require.NoError(t, err)
	require.Equal(t, info, infoResp.ValSigningInfo)
}

func TestGRPCSigningInfos(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	var signingInfos []types.ValidatorSigningInfo

	_, valPubKey1 := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 2_000_000, 1)
	_, valPubKey2 := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 2_000_000, 2)

	info := types.NewValidatorSigningInfo(sdk.ConsAddress(valPubKey1.Address()), int64(5), int64(4),
		time.Unix(2, 0), false, int64(10))
	input.SlashingKeeper.SetValidatorSigningInfo(ctx, sdk.ConsAddress(valPubKey1.Address()), info)
	info = types.NewValidatorSigningInfo(sdk.ConsAddress(valPubKey2.Address()), int64(5), int64(4),
		time.Unix(2, 0), false, int64(10))
	input.SlashingKeeper.SetValidatorSigningInfo(ctx, sdk.ConsAddress(valPubKey2.Address()), info)

	input.SlashingKeeper.IterateValidatorSigningInfos(ctx, func(consAddr sdk.ConsAddress, info types.ValidatorSigningInfo) (stop bool) {
		signingInfos = append(signingInfos, info)
		return false
	})

	// verify all values are returned without pagination
	infoResp, err := input.SlashingKeeper.SigningInfos(ctx,
		&types.QuerySigningInfosRequest{Pagination: nil})
	require.NoError(t, err)
	require.Equal(t, signingInfos, infoResp.Info)

	infoResp, err = input.SlashingKeeper.SigningInfos(ctx,
		&types.QuerySigningInfosRequest{Pagination: &query.PageRequest{Limit: 1, CountTotal: true}})

	require.NoError(t, err)
	require.Len(t, infoResp.Info, 1)
	require.Equal(t, signingInfos[0], infoResp.Info[0])
	require.NotNil(t, infoResp.Pagination.NextKey)
	require.Equal(t, uint64(2), infoResp.Pagination.Total)
}
