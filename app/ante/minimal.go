package ante

import (
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"

	"github.com/initia-labs/initia/app/ante/sigverify"
	dynamicfeeante "github.com/initia-labs/initia/x/dynamic-fee/ante"
	moveante "github.com/initia-labs/initia/x/move/ante"
)

// NewMinimalAnteHandler returns a reduced AnteHandler chain for CheckTx mode.
// It validates signatures, format, gas limits, and fees (for priority) but
// does not deduct fees or increment sequences, with those are handled by the
// full handler during PrepareProposal/FinalizeBlock.
func NewMinimalAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "account keeper is required for minimal ante handler")
	}

	if options.AccountAbstractionKeeper == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "account abstraction keeper is required for minimal ante handler")
	}

	if options.SignModeHandler == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for minimal ante handler")
	}

	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = sigverify.DefaultSigVerificationGasConsumer
	}

	txFeeChecker := options.TxFeeChecker
	if txFeeChecker == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "tx fee checker is required for minimal ante handler")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		moveante.NewGasPricesDecorator(),
		dynamicfeeante.NewBlockGasDecorator(options.DynamicFeeKeeper),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		NewCheckFeeDecorator(txFeeChecker), // validate fee + set priority, no deduction
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		sigverify.NewSigVerificationDecoratorWithAccountAbstraction(options.AccountKeeper, options.SignModeHandler, options.AccountAbstractionKeeper),
		// no IncrementSequenceDecorator here since mempool tracks nonces
		ibcante.NewRedundantRelayDecorator(options.IBCkeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

// CheckFeeDecorator validates that the tx meets minimum fee requirements and
// sets the priority on the context. Unlike DeductFeeDecorator, it does not
// deduct fees from the sender's account, that would happen in the full handler
// during PrepareProposal/FinalizeBlock.
type CheckFeeDecorator struct {
	feeChecker ante.TxFeeChecker
}

// NewCheckFeeDecorator returns a CheckFeeDecorator using the given fee checker.
func NewCheckFeeDecorator(feeChecker ante.TxFeeChecker) CheckFeeDecorator {
	return CheckFeeDecorator{feeChecker: feeChecker}
}

func (cfd CheckFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if _, ok := tx.(sdk.FeeTx); !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate {
		_, priority, err := cfd.feeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
		ctx = ctx.WithPriority(priority)
	}

	return next(ctx, tx, simulate)
}

// NewDualAnteHandler returns an AnteHandler that routes to the minimal handler
// during CheckTx/ReCheckTx and to the full handler otherwise.
func NewDualAnteHandler(minimal, full sdk.AnteHandler) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		if ctx.IsCheckTx() || ctx.IsReCheckTx() {
			return minimal(ctx, tx, simulate)
		}
		return full(ctx, tx, simulate)
	}
}
