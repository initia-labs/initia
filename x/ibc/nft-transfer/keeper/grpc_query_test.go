package keeper_test

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

func (suite *KeeperTestSuite) TestQueryClassData() {
	classTrace := types.ParseClassTrace("nft-transfer/channel-1/0x1::nft_store::Collection")
	traceHash := classTrace.Hash()
	classData := "collection-data"
	hashParam := "ibc/" + traceHash.String()

	testCases := []struct {
		name      string
		request   *types.QueryClassDataRequest
		setup     func(context.Context, *keeper.Keeper)
		expectErr bool
		errCode   codes.Code
		expData   string
	}{
		{
			name:    "class data found",
			request: &types.QueryClassDataRequest{Hash: hashParam},
			setup: func(ctx context.Context, k *keeper.Keeper) {
				err := k.ClassData.Set(ctx, traceHash, classData)
				suite.Require().NoError(err)
			},
			expData: classData,
		},
		{
			name:      "class data not found",
			request:   &types.QueryClassDataRequest{Hash: hashParam},
			expectErr: true,
			errCode:   codes.NotFound,
		},
		{
			name:      "invalid hash",
			request:   &types.QueryClassDataRequest{Hash: "invalid-hash"},
			expectErr: true,
			errCode:   codes.InvalidArgument,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, k := suite.SetupKeeperTest()

			if tc.setup != nil {
				tc.setup(ctx, k)
			}

			res, err := suite.queryClient.ClassData(context.Background(), tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Equal(tc.errCode, status.Code(err))
				return
			}

			suite.Require().NoError(err)
			suite.Require().Equal(tc.expData, res.GetData())
		})
	}
}
