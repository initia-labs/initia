package types

import (
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

// ConvertToSDKMessage convert vm CosmosMessage to sdk.Msg
func ConvertToSDKMessage(
	ctx sdk.Context,
	fk FungibleAssetKeeper,
	ck CollectionKeeper,
	msg vmtypes.CosmosMessage,
) (sdk.Msg, error) {
	switch msg := msg.(type) {
	case *vmtypes.CosmosMessage__Staking:
		switch msg := msg.Value.(type) {
		case *vmtypes.StakingMessage__Delegate:
			validatorAddress, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
			if err != nil {
				return nil, err
			}

			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Amount.Metadata)
			if err != nil {
				return nil, err
			}

			return stakingtypes.NewMsgDelegate(
				ConvertVMAddressToSDKAddress(msg.DelegatorAddress),
				validatorAddress,
				sdk.NewCoins(sdk.NewCoin(denom, sdk.NewIntFromUint64(msg.Amount.Amount))),
			), nil
		}
	case *vmtypes.CosmosMessage__Distribution:
		switch msg := msg.Value.(type) {
		case *vmtypes.DistributionMessage__FundCommunityPool:
			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Amount.Metadata)
			if err != nil {
				return nil, err
			}

			fundCommunityPoolMsg := distrtypes.NewMsgFundCommunityPool(
				sdk.NewCoins(sdk.NewCoin(denom, sdk.NewIntFromUint64(msg.Amount.Amount))),
				ConvertVMAddressToSDKAddress(msg.SenderAddress),
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

			transferMsg := transfertypes.NewMsgTransfer(
				msg.SourcePort,
				msg.SourceChannel,
				sdk.NewCoin(denom, sdk.NewIntFromUint64(msg.Token.Amount)),
				ConvertVMAddressToSDKAddress(msg.Sender).String(),
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

			nftTransferMsg := nfttransfertypes.NewMsgTransfer(
				msg.SourcePort,
				msg.SourceChannel,
				classId,
				msg.TokenIds,
				ConvertVMAddressToSDKAddress(msg.Sender).String(),
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

			payPacketFeeMsg := ibcfeetypes.NewMsgPayPacketFee(
				ibcfeetypes.NewFee(
					sdk.NewCoins(sdk.NewCoin(recvFeeDenom, sdk.NewIntFromUint64(msg.Fee.RecvFee.Amount))),
					sdk.NewCoins(sdk.NewCoin(ackFeeDenom, sdk.NewIntFromUint64(msg.Fee.AckFee.Amount))),
					sdk.NewCoins(sdk.NewCoin(timeoutFeeDenom, sdk.NewIntFromUint64(msg.Fee.TimeoutFee.Amount))),
				),
				msg.SourcePort,
				msg.SourceChannel,
				ConvertVMAddressToSDKAddress(msg.Signer).String(),
				[]string{},
			)

			return payPacketFeeMsg, nil
		}
	case *vmtypes.CosmosMessage__OPinit:
		switch msg := msg.Value.(type) {
		case *vmtypes.OPinitMessage__InitiateTokenDeposit:
			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Amount.Metadata)
			if err != nil {
				return nil, err
			}

			depositMsg := ophosttypes.NewMsgInitiateTokenDeposit(
				ConvertVMAddressToSDKAddress(msg.SenderAddress),
				msg.BridgeId,
				ConvertVMAddressToSDKAddress(msg.ToAddress),
				sdk.NewCoin(denom, sdk.NewIntFromUint64(msg.Amount.Amount)),
				msg.Data,
			)

			return depositMsg, nil

		case *vmtypes.OPinitMessage__InitiateTokenWithdrawal: // for l2 (minitia)
			denom, err := DenomFromMetadataAddress(ctx, fk, msg.Amount.Metadata)
			if err != nil {
				return nil, err
			}

			withdrawMsg := opchildtypes.NewMsgInitiateTokenWithdrawal(
				ConvertVMAddressToSDKAddress(msg.SenderAddress),
				ConvertVMAddressToSDKAddress(msg.ToAddress),
				sdk.NewCoin(denom, sdk.NewIntFromUint64(msg.Amount.Amount)),
			)

			return withdrawMsg, nil
		}
	}

	return nil, ErrNotSupportedCosmosMessage
}
