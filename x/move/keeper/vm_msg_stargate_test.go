package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_HandleVMStargateMsg(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	addr0, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	addr1, err := vmtypes.NewAccountAddressFromBytes(addrs[1])
	require.NoError(t, err)

	jsonData := []byte(fmt.Sprintf(`{
		"@type": "%s",
		"proposal_id": "1",
		"voter": "%s", 
		"option": 2,
		"metadata": ""
	}`, sdk.MsgTypeURL(&govtypes.MsgVote{}), addrs[1]))

	// unauthorized test
	_, err = input.MoveKeeper.HandleVMStargateMsg(ctx, &vmtypes.CosmosMessage{
		Sender: addr0,
		Data:   jsonData,
	})
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)

	// valid test
	res, err := input.MoveKeeper.HandleVMStargateMsg(ctx, &vmtypes.CosmosMessage{
		Sender: addr1,
		Data:   jsonData,
	})
	require.NoError(t, err)
	require.Equal(t, &govtypes.MsgVote{
		ProposalId: 1,
		Voter:      addrs[1].String(),
		Option:     2,
		Metadata:   "",
	}, res)
}
