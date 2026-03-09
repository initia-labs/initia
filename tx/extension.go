package tx

import (
	"slices"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	txtypes "github.com/initia-labs/initia/tx/types"
)

const ExtensionOptionQueuedTxTypeURL = "/initia.tx.v1.ExtensionOptionQueuedTx"
const FlagAllowQueued = "allow-queued"

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdktx.TxExtensionOptionI)(nil),
		&txtypes.ExtensionOptionQueuedTx{},
	)
}

func ExtensionOptionChecker(any *codectypes.Any) bool {
	return any != nil && any.TypeUrl == ExtensionOptionQueuedTxTypeURL
}

func HasQueuedTxExtension(tx sdk.Tx) bool {
	hasExtOptsTx, ok := tx.(ante.HasExtensionOptionsTx)
	if !ok {
		return false
	}

	return slices.ContainsFunc(hasExtOptsTx.GetExtensionOptions(), ExtensionOptionChecker)
}
