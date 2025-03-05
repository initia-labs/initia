package keeper_test

import (
	"github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
)

func (suite *KeeperTestSuite) TestMarshalClassTrace() {
	_, keeper := suite.SetupKeeperTest()

	classTrace := types.ParseClassTrace("nft-transfer/channel-1/gaeguri")
	bz, err := keeper.MarshalClassTrace(classTrace)
	suite.Require().NoError(err)

	ct, err := keeper.UnmarshalClassTrace(bz)
	suite.Require().NoError(err)

	suite.Require().Equal(classTrace, ct)
}

func (suite *KeeperTestSuite) TestMustMarshalClassTrace() {
	_, keeper := suite.SetupKeeperTest()

	classTrace := types.ParseClassTrace("nft-transfer/channel-1/gaeguri")
	bz := keeper.MustMarshalClassTrace(classTrace)

	ct := keeper.MustUnmarshalClassTrace(bz)
	suite.Require().Equal(classTrace, ct)
}
