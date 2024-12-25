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
	moveante "github.com/initia-labs/initia/x/move/ante"
	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/skip-mev/block-sdk/v2/block"
	auctionante "github.com/skip-mev/block-sdk/v2/x/auction/ante"
	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"
)

// HandlerOptions extends the SDK's AnteHandler options by including IBC channel keeper and custom handlers.
type HandlerOptions struct {
	ante.HandlerOptions
	Codec         codec.BinaryCodec
	MoveKeeper    movetypes.AnteKeeper
	IBCkeeper     *ibckeeper.Keeper
	AuctionKeeper auctionkeeper.Keeper
	TxEncoder     sdk.TxEncoder
	MevLane       auctionante.MEVLane
	FreeLane      block.Lane
}

// NewAnteHandler creates a custom AnteHandler pipeline for transaction processing.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	// Validate mandatory dependencies.
	if err := validateHandlerOptions(options); err != nil {
		return nil, err
	}

	// Default to provided or custom fee checker.
	txFeeChecker := getTxFeeChecker(options)

	// Define a custom free lane fee checker.
	freeLaneFeeChecker := func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		if !options.FreeLane.Match(ctx, tx) {
			return txFeeChecker(ctx, tx)
		}
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return nil, 0, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
		}
		return feeTx.GetFee(), 1 /* FIFO */, nil
	}

	// Create the AnteDecorators sequence.
	anteDecorators := buildAnteDecorators(options, freeLaneFeeChecker)

	// Chain the AnteDecorators to construct the AnteHandler.
	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

// validateHandlerOptions ensures all mandatory dependencies are provided.
func validateHandlerOptions(options HandlerOptions) error {
	if options.AccountKeeper == nil {
		return errors.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return errors.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return errors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}
	return nil
}

// getTxFeeChecker returns the appropriate TxFeeChecker function.
func getTxFeeChecker(options HandlerOptions) func(sdk.Context, sdk.Tx) (sdk.Coins, int64, error) {
	if options.TxFeeChecker != nil {
		return options.TxFeeChecker
	}
	return moveante.NewMempoolFeeChecker(options.MoveKeeper).CheckTxFeeWithMinGasPrices
}

// buildAnteDecorators constructs the list of AnteDecorators in the correct order.
func buildAnteDecorators(options HandlerOptions, freeLaneFeeChecker func(sdk.Context, sdk.Tx) (sdk.Coins, int64, error)) []sdk.AnteDecorator {
	return []sdk.AnteDecorator{
		accnum.NewAccountNumberDecorator(options.AccountKeeper),
		ante.NewSetUpContextDecorator(), // Must be the first decorator.
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		moveante.NewGasPricesDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, freeLaneFeeChecker),
		ante.NewSetPubKeyDecorator(options.AccountKeeper), // Must be called before signature verification.
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigverify.DefaultSigVerificationGasConsumer),
		sigverify.NewSigVerificationDecorator(options.AccountKeeper, sigGasConsumer),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCkeeper),
		auctionante.NewAuctionDecorator(options.AuctionKeeper, options.TxEncoder, options.MevLane),
	}
}
