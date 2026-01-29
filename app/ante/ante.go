package ante

import (
	"cosmossdk.io/errors"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/initia-labs/initia/app/ante/accnum"
	"github.com/initia-labs/initia/app/ante/sigverify"
	dynamicfeeante "github.com/initia-labs/initia/x/dynamic-fee/ante"
	moveante "github.com/initia-labs/initia/x/move/ante"

	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
)

// HandlerOptions extends the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	Codec     codec.BinaryCodec
	TxEncoder sdk.TxEncoder

	// expected keepers
	DynamicFeeKeeper         dynamicfeetypes.AnteKeeper
	IBCkeeper                *ibckeeper.Keeper
	AccountAbstractionKeeper sigverify.AccountAbstractionKeeper
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}

	if options.AccountAbstractionKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "account abstraction keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = sigverify.DefaultSigVerificationGasConsumer
	}

	txFeeChecker := options.TxFeeChecker
	if txFeeChecker == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "tx fee checker is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		accnum.NewAccountNumberDecorator(options.AccountKeeper),
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		moveante.NewGasPricesDecorator(),
		dynamicfeeante.NewBlockGasDecorator(options.DynamicFeeKeeper),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, txFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		sigverify.NewSigVerificationDecoratorWithAccountAbstraction(options.AccountKeeper, options.SignModeHandler, options.AccountAbstractionKeeper),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCkeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
