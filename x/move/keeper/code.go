package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

type CodeKeeper struct {
	*Keeper
}

// NewCodeKeeper create new CodeKeeper instance
func NewCodeKeeper(k *Keeper) CodeKeeper {
	return CodeKeeper{k}
}

// Load the code params from the move store
func (k CodeKeeper) GetParams(ctx context.Context) (bool, []vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   vmtypes.Identifier(types.MoveModuleNameCode),
		Name:     vmtypes.Identifier(types.ResourceNameModuleStore),
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return false, nil, err
	}

	return types.ReadCodeModuleStore(bz)
}

// Load the allow_arbitrary flag from the move store
func (k CodeKeeper) GetAllowArbitrary(ctx context.Context) (bool, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   vmtypes.Identifier(types.MoveModuleNameCode),
		Name:     vmtypes.Identifier(types.ResourceNameModuleStore),
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return false, err
	}

	return vmtypes.NewDeserializer(bz[:1]).DeserializeBool()
}

// Store the allow_arbitrary flag to move store.
func (k CodeKeeper) SetAllowArbitrary(ctx context.Context, allow bool) error {
	ser := vmtypes.NewSerializer()
	if err := ser.SerializeBool(allow); err != nil {
		return err
	}

	return k.ExecuteEntryFunction(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		types.MoveModuleNameCode,
		types.FunctionNameCodeSetAllowArbitrary,
		[]vmtypes.TypeTag{},
		[][]byte{
			ser.GetBytes(),
		},
	)
}

// Load the allowed_publishers from the move store
func (k CodeKeeper) GetAllowedPublishers(ctx context.Context) ([]vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   vmtypes.Identifier(types.MoveModuleNameCode),
		Name:     vmtypes.Identifier(types.ResourceNameModuleStore),
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return nil, err
	}

	_, allowedPublishers, err := types.ReadCodeModuleStore(bz)
	return allowedPublishers, err
}

// Store the allowed_publishers to move store.
func (k CodeKeeper) SetAllowedPublishers(ctx context.Context, allowedPublishers []vmtypes.AccountAddress) error {
	bz, err := vmtypes.SerializeAddressVector(allowedPublishers)
	if err != nil {
		return err
	}

	return k.ExecuteEntryFunction(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		types.MoveModuleNameCode,
		types.FunctionNameCodeSetAllowedPublishers,
		[]vmtypes.TypeTag{},
		[][]byte{
			bz,
		},
	)
}

// GetUpgradePolicy reads upgrade policy from the code module.
func (k CodeKeeper) GetUpgradePolicy(ctx context.Context, addr vmtypes.AccountAddress, name string) (types.UpgradePolicy, error) {
	st := vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   vmtypes.Identifier(types.MoveModuleNameCode),
		Name:     vmtypes.Identifier(types.ResourceNameMetadataStore),
		TypeArgs: []vmtypes.TypeTag{},
	}

	bz, err := k.GetResourceBytes(ctx, addr, st)
	if err != nil {
		return types.UpgradePolicy_ARBITRARY, err
	}

	tableHandle, err := types.ReadMetadataTableHandleFromMetadataStore(bz)
	if err != nil {
		return types.UpgradePolicy_ARBITRARY, err
	}

	vmAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return types.UpgradePolicy_ARBITRARY, err
	}

	tableKey, err := vmtypes.SerializeString(vmtypes.NewModuleId(vmAddr, name).String())
	if err != nil {
		return types.UpgradePolicy_ARBITRARY, err
	}

	tableEntry, err := k.GetTableEntryBytes(ctx, tableHandle, tableKey)
	if err != nil {
		return types.UpgradePolicy_ARBITRARY, err
	}

	return types.ReadUpgradePolicyFromModuleMetadata(tableEntry.ValueBytes)
}
