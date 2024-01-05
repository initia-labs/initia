package genutil

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

func DefaultMessageValidator(msgs []sdk.Msg) error {
	if len(msgs) != 1 {
		return fmt.Errorf("unexpected number of GenTx messages; got: %d, expected: 1", len(msgs))
	}
	if _, ok := msgs[0].(*stakingtypes.MsgCreateValidator); !ok {
		return fmt.Errorf("unexpected GenTx message type; expected: MsgCreateValidator, got: %V", msgs[0])
	}

	if m, ok := msgs[0].(sdk.HasValidateBasic); ok {
		if err := m.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid GenTx '%s': %w", msgs[0], err)
		}
	}

	return nil
}
