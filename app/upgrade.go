package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmprecompile "github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "0.6.0"

// RegisterUpgradeHandlers returns upgrade handlers
func (app *InitiaApp) RegisterUpgradeHandlers(cfg module.Configurator) {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

			fmt.Println("upgrade to 0.6.0")

			// 1. publish new code module first
			codeModuleBz, err := vmprecompile.ReadStdlib("code.mv")
			if err != nil {
				return nil, err
			}
			err = app.MoveKeeper.SetModule(ctx, vmtypes.StdAddress, movetypes.MoveModuleNameCode, codeModuleBz[0])
			if err != nil {
				return nil, err
			}

			fmt.Println("2. set code module")

			// 2. update vm data with new seperator and add checksums of each module

			type KV struct {
				key   []byte
				value []byte
			}
			kvs := make([]KV, 0)

			//  Previous:
			// 	ModuleSeparator     = byte(0)
			// 	ResourceSeparator   = byte(1)
			// 	TableEntrySeparator = byte(2)
			//	TableInfoSeparator  = byte(3)

			//  Current:
			// 	ModuleSeparator     = byte(0)
			//	ChecksumSeparator   = byte(1)
			// 	ResourceSeparator   = byte(2)
			// 	TableEntrySeparator = byte(3)
			//	TableInfoSeparator  = byte(4)

			fmt.Println("3. loading all kvs")

			err = app.MoveKeeper.VMStore.Walk(ctx, nil, func(key, value []byte) (stop bool, err error) {
				cursor := movetypes.AddressBytesLength
				if len(key) <= cursor {
					return true, fmt.Errorf("invalid key length: %d", len(key))
				}

				separator := key[cursor]

				if separator == movetypes.ModuleSeparator {
					identifier, err := vmtypes.BcsDeserializeIdentifier(key[cursor+1:])
					if err != nil {
						return true, err
					}

					fmt.Println("module", identifier)

					checksum := movetypes.ModuleBzToChecksum(value)
					fmt.Println("checksum", hex.EncodeToString(checksum[:]))

					value = checksum[:]
				} else if separator >= movetypes.TableInfoSeparator {
					return true, errors.New("unknown prefix")
				} else {
					err = app.MoveKeeper.VMStore.Remove(ctx, key)
					if err != nil {
						return true, err
					}
				}
				key[cursor] = key[cursor] + 1
				kvs = append(kvs, KV{
					key:   bytes.Clone(key),
					value: bytes.Clone(value),
				})
				return false, nil
			})
			if err != nil {
				return nil, err
			}

			fmt.Println("4. storing all kvs")

			for _, kv := range kvs {
				err = app.MoveKeeper.VMStore.Set(ctx, kv.key, kv.value)
				if err != nil {
					return nil, err
				}
			}

			// 3. update new modules

			fmt.Println("5. upgrading modules")

			codesBz, err := vmprecompile.ReadStdlib("object_code_deployment.mv", "coin.mv", "cosmos.mv", "dex.mv", "json.mv", "bech32.mv", "hash.mv", "collection.mv")
			if err != nil {
				return nil, err
			}
			modules := make([]vmtypes.Module, len(codesBz))
			for i, codeBz := range codesBz {
				modules[i] = vmtypes.NewModule(codeBz)
			}

			err = app.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "failed to publish module bundle")
			}

			return vm, nil
		},
	)
}
