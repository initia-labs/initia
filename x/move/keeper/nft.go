package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

var _ nfttransfertypes.NftKeeper = NftKeeper{}
var _ types.CollectionKeeper = NftKeeper{}

// NftKeeper implements move wrapper for types.NftKeeper interface
type NftKeeper struct {
	*Keeper
}

// NewNftKeeper return new NftKeeper instance
func NewNftKeeper(k *Keeper) NftKeeper {
	return NftKeeper{k}
}

func (k NftKeeper) CollectionInfo(ctx context.Context, collectionAddr vmtypes.AccountAddress) (
	creator vmtypes.AccountAddress,
	name, uri, desc string,
	err error,
) {
	bz, err := k.GetResourceBytes(ctx, collectionAddr, vmtypes.StructTag{
		Address: vmtypes.StdAddress,
		Module:  types.MoveModuleNameCollection,
		Name:    types.ResourceNameCollection,
	})
	if err != nil {
		return
	}

	return types.ReadCollectionInfo(bz)
}

// Transfer transfers an NFT from one account to another.
func (k NftKeeper) Transfer(ctx context.Context, sender, receiver, tokenAddr vmtypes.AccountAddress) error {
	return k.ExecuteEntryFunction(
		ctx,
		sender,
		vmtypes.StdAddress,
		types.MoveModuleNameObject,
		types.FunctionNameObjectTransfer,
		[]vmtypes.TypeTag{types.TypeTagFromStructTag(vmtypes.StructTag{
			Address: vmtypes.StdAddress,
			Module:  types.MoveModuleNameNft,
			Name:    types.ResourceNameNft,
		})},
		[][]byte{tokenAddr[:], receiver[:]},
	)
}

// Mint mints a new NFT.
func (k NftKeeper) Mint(
	ctx context.Context,
	collectionName, tokenID, tokenURI, tokenDesc string,
	recipientAddr vmtypes.AccountAddress,
) error {
	collectionNameBz, err := vmtypes.SerializeString(collectionName)
	if err != nil {
		return err
	}

	tokenIDBz, err := vmtypes.SerializeString(tokenID)
	if err != nil {
		return err
	}

	tokenURIBz, err := vmtypes.SerializeString(tokenURI)
	if err != nil {
		return err
	}

	tokenDescBz, err := vmtypes.SerializeString(tokenDesc)
	if err != nil {
		return err
	}

	return k.ExecuteEntryFunction(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		types.MoveModuleNameSimpleNft,
		types.FunctionNameSimpleNftMint,
		[]vmtypes.TypeTag{},
		[][]byte{collectionNameBz, tokenDescBz, tokenIDBz, tokenURIBz, {1}, append([]byte{1}, recipientAddr[:]...)},
	)
}

func (k NftKeeper) Burn(ctx context.Context, ownerAddr, tokenAddr vmtypes.AccountAddress) error {
	return k.ExecuteEntryFunction(
		ctx,
		ownerAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameSimpleNft,
		types.FunctionNameSimpleNftBurn,
		[]vmtypes.TypeTag{types.TypeTagFromStructTag(vmtypes.StructTag{
			Address: vmtypes.StdAddress,
			Module:  types.MoveModuleNameSimpleNft,
			Name:    types.ResourceNameSimpleNft,
		})},
		[][]byte{tokenAddr[:]},
	)
}

func (k NftKeeper) isCollectionInitialized(ctx context.Context, collection vmtypes.AccountAddress) (bool, error) {
	return k.HasResource(ctx, collection, vmtypes.StructTag{
		Address: vmtypes.StdAddress,
		Module:  types.MoveModuleNameCollection,
		Name:    types.ResourceNameCollection,
	})
}

func (k NftKeeper) CreateOrUpdateClass(ctx context.Context, classId, classUri, classData string) error {
	collectionAddr, err := types.CollectionAddressFromClassId(classId)
	if err != nil {
		return err
	}

	if ok, err := k.isCollectionInitialized(ctx, collectionAddr); err != nil {
		return err
	} else if !ok {
		// use classId as collection name
		err := k.initializeCollection(ctx, classId, classUri, classData)
		if err != nil {
			return err
		}
	} // update not supported; ignore

	return nil
}

func (k NftKeeper) initializeCollection(ctx context.Context, collectionName, classUri, classData string) error {
	collectionNameBz, err := vmtypes.SerializeString(collectionName)
	if err != nil {
		return err
	}

	collectionUriBz, err := vmtypes.SerializeString(classUri)
	if err != nil {
		return err
	}

	collectionDescBz, err := vmtypes.SerializeString(classData)
	if err != nil {
		return err
	}

	return k.ExecuteEntryFunction(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		types.MoveModuleNameSimpleNft,
		types.FunctionNameSimpleNftInitialize,
		[]vmtypes.TypeTag{},
		[][]byte{collectionDescBz, {0}, collectionNameBz, collectionUriBz, {0}, {0}, {0}, {0}, {0}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	)
}

func (k NftKeeper) Transfers(ctx context.Context, sender, receiver sdk.AccAddress, classId string, tokenIds []string) error {
	// register account (nft is not using bank module, so need to register manually)
	accExists := k.authKeeper.HasAccount(ctx, receiver)
	if !accExists {
		defer telemetry.IncrCounter(1, "new", "account")
		k.authKeeper.SetAccount(ctx, k.authKeeper.NewAccountWithAddress(ctx, receiver))
	}

	senderAddr, err := vmtypes.NewAccountAddressFromBytes(sender)
	if err != nil {
		return err
	}

	receiverAddr, err := vmtypes.NewAccountAddressFromBytes(receiver)
	if err != nil {
		return err
	}

	collectionAddr, err := types.CollectionAddressFromClassId(classId)
	if err != nil {
		return err
	}

	collectionCreator, collectionName, _, _, err := k.CollectionInfo(ctx, collectionAddr)
	if err != nil {
		return err
	}

	for _, tokenId := range tokenIds {
		tokenAddr, err := types.TokenAddressFromTokenId(collectionCreator, collectionName, tokenId)
		if err != nil {
			return err
		}

		if err := k.Transfer(ctx, senderAddr, receiverAddr, tokenAddr); err != nil {
			return err
		}
	}

	return nil
}

func (k NftKeeper) Burns(ctx context.Context, owner sdk.AccAddress, classId string, tokenIds []string) error {
	ownerAddr, err := vmtypes.NewAccountAddressFromBytes(owner)
	if err != nil {
		return err
	}

	collection, err := types.CollectionAddressFromClassId(classId)
	if err != nil {
		return err
	}

	collectionCreator, collectionName, _, _, err := k.CollectionInfo(ctx, collection)
	if err != nil {
		return err
	}

	for _, tokenId := range tokenIds {
		tokenAddr, err := types.TokenAddressFromTokenId(collectionCreator, collectionName, tokenId)
		if err != nil {
			return err
		}

		if err := k.Burn(ctx, ownerAddr, tokenAddr); err != nil {
			return err
		}
	}

	return nil
}

func (k NftKeeper) Mints(
	ctx context.Context, receiver sdk.AccAddress,
	classId string, tokenIds, tokenUris, tokenData []string,
) error {
	// register account (nft is not using bank module, so need to register manually)
	accExists := k.authKeeper.HasAccount(ctx, receiver)
	if !accExists {
		defer telemetry.IncrCounter(1, "new", "account")
		k.authKeeper.SetAccount(ctx, k.authKeeper.NewAccountWithAddress(ctx, receiver))
	}

	receiverAddr, err := vmtypes.NewAccountAddressFromBytes(receiver)
	if err != nil {
		return err
	}

	collection, err := types.CollectionAddressFromClassId(classId)
	if err != nil {
		return err
	}

	_, collectionName, _, _, err := k.CollectionInfo(ctx, collection)
	if err != nil {
		return err
	}

	for i := range tokenIds {
		if err := k.Mint(ctx, collectionName, tokenIds[i], tokenUris[i], tokenData[i], receiverAddr); err != nil {
			return err
		}
	}

	return nil
}

func (k NftKeeper) GetClassInfo(ctx context.Context, classId string) (className string, classUri string, classDescs string, err error) {
	collectionAddr, err := types.CollectionAddressFromClassId(classId)
	if err != nil {
		return "", "", "", err
	}

	_, className, classUri, classDescs, err = k.CollectionInfo(ctx, collectionAddr)
	return
}

func (k NftKeeper) GetTokenInfos(ctx context.Context, classId string, tokenIds []string) (tokenUris []string, tokenDescs []string, err error) {
	collectionAddr, err := types.CollectionAddressFromClassId(classId)
	if err != nil {
		return nil, nil, err
	}

	collectionCreator, collectionName, _, _, err := k.CollectionInfo(ctx, collectionAddr)
	if err != nil {
		return nil, nil, err
	}

	tokenUris = make([]string, len(tokenIds))
	tokenDescs = make([]string, len(tokenIds))
	for i, tokenId := range tokenIds {
		tokenAddr, err := types.TokenAddressFromTokenId(collectionCreator, collectionName, tokenId)
		if err != nil {
			return nil, nil, err
		}

		bz, err := k.GetResourceBytes(ctx, tokenAddr, vmtypes.StructTag{
			Address: vmtypes.StdAddress,
			Module:  types.MoveModuleNameNft,
			Name:    types.ResourceNameNft,
		})
		if err != nil {
			return nil, nil, err
		}

		_, tokenUri, tokenDesc := types.ReadNftInfo(bz)
		tokenUris[i] = tokenUri
		tokenDescs[i] = tokenDesc
	}
	return
}
