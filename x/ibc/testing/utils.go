package ibctesting

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ApplyValSetChanges takes in tmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new tmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func ApplyValSetChanges(tb testing.TB, valSet *tmtypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) *tmtypes.ValidatorSet {
	tb.Helper()
	updates, err := tmtypes.PB2TM.ValidatorUpdates(valUpdates)
	require.NoError(tb, err)

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	require.NoError(tb, err)

	return newVals
}

// GenerateString generates a random string of the given length in bytes
func GenerateString(length uint) string {
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = charset[rand.Intn(len(charset))] //nolint weak random number generator is acceptable here
	}
	return string(bytes)
}

func GetMarshaledValue(data []byte) ([][]byte, error) {
	var msgData sdk.TxMsgData
	err := proto.Unmarshal(data, &msgData)
	if err != nil {
		return nil, err
	}
	res := make([][]byte, len(msgData.MsgResponses))
	for i, msgResponse := range msgData.MsgResponses {
		res[i] = msgResponse.Value
	}
	return res, nil
}
