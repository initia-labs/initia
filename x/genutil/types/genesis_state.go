package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// ValidateGenesis validates GenTx transactions
func ValidateGenesis(genesisState *cosmostypes.GenesisState, txJSONDecoder sdk.TxDecoder) error {
	for i, genTx := range genesisState.GenTxs {
		var tx sdk.Tx
		tx, err := txJSONDecoder(genTx)
		if err != nil {
			return err
		}

		msgs := tx.GetMsgs()
		if len(msgs) != 1 {
			return errors.New(
				"must provide genesis Tx with exactly 1 CreateValidator message")
		}

		// TODO: abstract back to staking
		if _, ok := msgs[0].(*stakingtypes.MsgCreateValidator); !ok {
			return fmt.Errorf(
				"genesis transaction %v does not contain a MsgCreateValidator", i)
		}
	}
	return nil
}
