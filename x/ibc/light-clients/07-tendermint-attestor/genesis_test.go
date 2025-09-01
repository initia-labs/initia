package tendermintattestor_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"

	ibctmttestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
)

// expected export ordering:
// processed height and processed time per height
// then all iteration keys
func (suite *TMAttestorTestSuite) TestExportMetadata() {
	// test intializing client and exporting metadata
	path := ibctesting.NewPathWithTendermintAttestors(suite.chainA, suite.chainB, 0, 0)
	suite.coordinator.SetupClients(path)
	clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
	clientState := path.EndpointA.GetClientState()
	height := clientState.GetLatestHeight()

	initIteration := ibctmttestor.GetIterationKey(clientStore, height)
	suite.Require().NotEqual(0, len(initIteration))
	initProcessedTime, found := ibctmttestor.GetProcessedTime(clientStore, height)
	suite.Require().True(found)
	initProcessedHeight, found := ibctmttestor.GetProcessedHeight(clientStore, height)
	suite.Require().True(found)

	gm := clientState.ExportMetadata(suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID))
	suite.Require().NotNil(gm, "client with metadata returned nil exported metadata")
	suite.Require().Len(gm, 3, "exported metadata has unexpected length")

	suite.Require().Equal(ibctmttestor.ProcessedHeightKey(height), gm[0].GetKey(), "metadata has unexpected key")
	actualProcessedHeight, err := clienttypes.ParseHeight(string(gm[0].GetValue()))
	suite.Require().NoError(err)
	suite.Require().Equal(initProcessedHeight, actualProcessedHeight, "metadata has unexpected value")

	suite.Require().Equal(ibctmttestor.ProcessedTimeKey(height), gm[1].GetKey(), "metadata has unexpected key")
	suite.Require().Equal(initProcessedTime, sdk.BigEndianToUint64(gm[1].GetValue()), "metadata has unexpected value")

	suite.Require().Equal(ibctmttestor.IterationKey(height), gm[2].GetKey(), "metadata has unexpected key")
	suite.Require().Equal(initIteration, gm[2].GetValue(), "metadata has unexpected value")

	// test updating client and exporting metadata
	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	clientState = path.EndpointA.GetClientState()
	updateHeight := clientState.GetLatestHeight()

	iteration := ibctmttestor.GetIterationKey(clientStore, updateHeight)
	suite.Require().NotEqual(0, len(iteration))
	processedTime, found := ibctmttestor.GetProcessedTime(clientStore, updateHeight)
	suite.Require().True(found)
	processedHeight, found := ibctmttestor.GetProcessedHeight(clientStore, updateHeight)
	suite.Require().True(found)

	gm = clientState.ExportMetadata(suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID))
	suite.Require().NotNil(gm, "client with metadata returned nil exported metadata")
	suite.Require().Len(gm, 6, "exported metadata has unexpected length")

	// expected ordering:
	// initProcessedHeight, initProcessedTime, processedHeight, processedTime, initIteration, iteration

	// check init processed height and time
	suite.Require().Equal(ibctmttestor.ProcessedHeightKey(height), gm[0].GetKey(), "metadata has unexpected key")
	actualProcessedHeight, err = clienttypes.ParseHeight(string(gm[0].GetValue()))
	suite.Require().NoError(err)
	suite.Require().Equal(initProcessedHeight, actualProcessedHeight, "metadata has unexpected value")

	suite.Require().Equal(ibctmttestor.ProcessedTimeKey(height), gm[1].GetKey(), "metadata has unexpected key")
	suite.Require().Equal(initProcessedTime, sdk.BigEndianToUint64(gm[1].GetValue()), "metadata has unexpected value")

	// check processed height and time after update
	suite.Require().Equal(ibctmttestor.ProcessedHeightKey(updateHeight), gm[2].GetKey(), "metadata has unexpected key")
	actualProcessedHeight, err = clienttypes.ParseHeight(string(gm[2].GetValue()))
	suite.Require().NoError(err)
	suite.Require().Equal(processedHeight, actualProcessedHeight, "metadata has unexpected value")

	suite.Require().Equal(ibctmttestor.ProcessedTimeKey(updateHeight), gm[3].GetKey(), "metadata has unexpected key")
	suite.Require().Equal(processedTime, sdk.BigEndianToUint64(gm[3].GetValue()), "metadata has unexpected value")

	// check iteration keys
	suite.Require().Equal(ibctmttestor.IterationKey(height), gm[4].GetKey(), "metadata has unexpected key")
	suite.Require().Equal(initIteration, gm[4].GetValue(), "metadata has unexpected value")

	suite.Require().Equal(ibctmttestor.IterationKey(updateHeight), gm[5].GetKey(), "metadata has unexpected key")
	suite.Require().Equal(iteration, gm[5].GetValue(), "metadata has unexpected value")
}
