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

	"github.com/skip-mev/block-sdk/v2/block"
	auctionante "github.com/skip-mev/block-sdk/v2/x/auction/ante"
	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"

	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
)

// HandlerOptions extends the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	Codec            codec.BinaryCodec
	DynamicFeeKeeper dynamicfeetypes.AnteKeeper
	IBCkeeper        *ibckeeper.Keeper
	AuctionKeeper    auctionkeeper.Keeper
	TxEncoder        sdk.TxEncoder
	MevLane          auctionante.MEVLane
	FreeLane         block.Lane
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

	if options.SignModeHandler == nil {
		return nil, errors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = sigverify.DefaultSigVerificationGasConsumer
	}

	txFeeChecker := options.TxFeeChecker
	if txFeeChecker == nil {
		txFeeChecker = dynamicfeeante.NewMempoolFeeChecker(options.DynamicFeeKeeper).CheckTxFeeWithMinGasPrices
	}

	freeLaneFeeChecker := func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		// skip fee checker if the tx is free lane tx.
		if !options.FreeLane.Match(ctx, tx) {
			return txFeeChecker(ctx, tx)
		}

		// return fee without fee check
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return nil, 0, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
		}

		return feeTx.GetFee(), 1 /* FIFO */, nil
	}

	anteDecorators := []sdk.AnteDecorator{
		accnum.NewAccountNumberDecorator(options.AccountKeeper),
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		dynamicfeeante.NewGasPricesDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, freeLaneFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		sigverify.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCkeeper),
		auctionante.NewAuctionDecorator(options.AuctionKeeper, options.TxEncoder, options.MevLane),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
