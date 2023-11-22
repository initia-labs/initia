package lanes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	"github.com/skip-mev/block-sdk/block/base"
)

// FreeLaneMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are ibc related. In particular,
// any transaction that is a MsgTimeout, MsgAcknowledgement.
func FreeLaneMatchHandler() base.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *channeltypes.MsgTimeout:
				return true
			case *channeltypes.MsgAcknowledgement:
				return true
			}
		}

		return false
	}
}
