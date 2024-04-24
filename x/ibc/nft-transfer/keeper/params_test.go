package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

func (suite *KeeperTestSuite) TestSendEnabled() {
	testCases := []struct {
		name        string
		sendEnabled bool
	}{
		{"enable send", true},
		{"disable send", false},
	}

	for _, tc := range testCases {
		ctx, keeper := suite.SetupKeeperTest()
		params := types.Params{
			SendEnabled: tc.sendEnabled,
		}
		err := keeper.Params.Set(ctx, params)
		suite.Require().NoError(err, tc.name)

		sendEnabled, err := keeper.GetSendEnabled(sdk.UnwrapSDKContext(ctx))
		suite.Require().NoError(err, tc.name)
		suite.Require().Equal(tc.sendEnabled, sendEnabled, tc.name)
	}
}

func (suite *KeeperTestSuite) TestReceiveEnabled() {
	testCases := []struct {
		name           string
		receiveEnabled bool
	}{
		{"enable receive", true},
		{"disable receive", false},
	}

	for _, tc := range testCases {
		ctx, keeper := suite.SetupKeeperTest()
		params := types.Params{
			ReceiveEnabled: tc.receiveEnabled,
		}
		err := keeper.Params.Set(ctx, params)
		suite.Require().NoError(err, tc.name)

		receiveEnabled, err := keeper.GetReceiveEnabled(sdk.UnwrapSDKContext(ctx))
		suite.Require().NoError(err, tc.name)
		suite.Require().Equal(tc.receiveEnabled, receiveEnabled, tc.name)
	}
}
