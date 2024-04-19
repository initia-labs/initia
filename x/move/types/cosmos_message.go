package types

import (
	context "context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// ConvertToSDKMessage convert vm CosmosMessage to sdk.Msg
func ConvertToSDKMessage(
	ctx context.Context,
	fk FungibleAssetKeeper,
	ck CollectionKeeper,
	msg vmtypes.CosmosMessage,
	ac address.Codec,
	vc address.Codec,
) (sdk.Msg, error) {
	switch msg := msg.(type) {
	case *vmtypes.CosmosMessage__Move:
		switch msg := msg.Value.(type) {
		case *vmtypes.MoveMessage__Execute:
			sender, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.Sender))
			if err != nil {
				return nil, err
			}

			return NewMsgExecute(
				sender,
				msg.ModuleAddress.String(),
				msg.ModuleName,
				msg.FunctionName,
				msg.TypeArgs,
				msg.Args,
			), nil
		case *vmtypes.MoveMessage__Script:
			sender, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.Sender))
			if err != nil {
				return nil, err
			}

			return NewMsgScript(
				sender,
				msg.CodeBytes,
				msg.TypeArgs,
				msg.Args,
			), nil
		}

	case *vmtypes.CosmosMessage__Staking:
		switch msg := msg.Value.(type) {
		case *vmtypes.StakingMessage__Delegate:
			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Amount.Metadata)
			if err != nil {
				return nil, err
			}

			delAddr, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.DelegatorAddress))
			if err != nil {
				return nil, err
			}

			return stakingtypes.NewMsgDelegate(
				delAddr,
				msg.ValidatorAddress,
				sdk.NewCoins(sdk.NewCoin(denom, math.NewIntFromUint64(msg.Amount.Amount))),
			), nil
		}
	case *vmtypes.CosmosMessage__Distribution:
		switch msg := msg.Value.(type) {
		case *vmtypes.DistributionMessage__FundCommunityPool:
			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Amount.Metadata)
			if err != nil {
				return nil, err
			}

			senderAddr, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.SenderAddress))
			if err != nil {
				return nil, err
			}

			fundCommunityPoolMsg := distrtypes.NewMsgFundCommunityPool(
				sdk.NewCoins(sdk.NewCoin(denom, math.NewIntFromUint64(msg.Amount.Amount))),
				senderAddr,
			)

			return fundCommunityPoolMsg, nil
		}
	case *vmtypes.CosmosMessage__Ibc:
		switch msg := msg.Value.(type) {
		case *vmtypes.IBCMessage__Transfer:
			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Token.Metadata)
			if err != nil {
				return nil, err
			}

			senderAddr, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.Sender))
			if err != nil {
				return nil, err
			}

			transferMsg := transfertypes.NewMsgTransfer(
				msg.SourcePort,
				msg.SourceChannel,
				sdk.NewCoin(denom, math.NewIntFromUint64(msg.Token.Amount)),
				senderAddr,
				msg.Receiver,
				clienttypes.NewHeight(msg.TimeoutHeight.RevisionNumber, msg.TimeoutHeight.RevisionHeight),
				msg.TimeoutTimestamp,
				msg.Memo,
			)

			return transferMsg, nil
		case *vmtypes.IBCMessage__NftTransfer:
			classId, err := ClassIdFromCollectionAddress(ctx, ck, msg.Collection)
			if err != nil {
				return nil, err
			}

			senderAddr, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.Sender))
			if err != nil {
				return nil, err
			}

			nftTransferMsg := nfttransfertypes.NewMsgTransfer(
				msg.SourcePort,
				msg.SourceChannel,
				classId,
				msg.TokenIds,
				senderAddr,
				msg.Receiver,
				clienttypes.NewHeight(msg.TimeoutHeight.RevisionNumber, msg.TimeoutHeight.RevisionHeight),
				msg.TimeoutTimestamp,
				msg.Memo,
			)

			return nftTransferMsg, nil
		case *vmtypes.IBCMessage__PayFee:
			recvFeeDenom, err := DenomFromMetadataAddress(ctx, fk, msg.Fee.RecvFee.Metadata)
			if err != nil {
				return nil, err
			}

			ackFeeDenom, err := DenomFromMetadataAddress(ctx, fk, msg.Fee.AckFee.Metadata)
			if err != nil {
				return nil, err
			}

			timeoutFeeDenom, err := DenomFromMetadataAddress(ctx, fk, msg.Fee.TimeoutFee.Metadata)
			if err != nil {
				return nil, err
			}

			senderAddr, err := ac.BytesToString(ConvertVMAddressToSDKAddress(msg.Signer))
			if err != nil {
				return nil, err
			}

			payPacketFeeMsg := ibcfeetypes.NewMsgPayPacketFee(
				ibcfeetypes.NewFee(
					sdk.NewCoins(sdk.NewCoin(recvFeeDenom, math.NewIntFromUint64(msg.Fee.RecvFee.Amount))),
					sdk.NewCoins(sdk.NewCoin(ackFeeDenom, math.NewIntFromUint64(msg.Fee.AckFee.Amount))),
					sdk.NewCoins(sdk.NewCoin(timeoutFeeDenom, math.NewIntFromUint64(msg.Fee.TimeoutFee.Amount))),
				),
				msg.SourcePort,
				msg.SourceChannel,
				senderAddr,
				[]string{},
			)

			return payPacketFeeMsg, nil
		}
	}

	return nil, ErrNotSupportedCosmosMessage
}
