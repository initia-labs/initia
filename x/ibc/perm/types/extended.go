package types

import (
	"slices"

	"cosmossdk.io/errors"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (rl *PermissionedRelayersList) HasRelayer(addr string) bool {
	return slices.Contains(rl.Relayers, addr)
}
func (rl *PermissionedRelayersList) AddRelayer(addr string) {
	rl.Relayers = append(rl.Relayers, addr)
}

func (rl *PermissionedRelayersList) GetAccAddr(cdc address.Codec) ([]sdk.AccAddress, error) {

	var relayerStrs []sdk.AccAddress
	for _, relayer := range rl.Relayers {

		relayer, err := cdc.StringToBytes(relayer)
		if err != nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		relayerStrs = append(relayerStrs, relayer)
	}

	return relayerStrs, nil
}

func ToRelayerList(cdc address.Codec, relayers []sdk.AccAddress) (PermissionedRelayersList, error) {
	var relayerList PermissionedRelayersList
	for _, relayer := range relayers {
		relayerStr, error := cdc.BytesToString(relayer)
		if error != nil {
			return relayerList, errors.Wrapf(sdkerrors.ErrInvalidAddress, "relayer address could not be converted to string : %v", error)
		}
		relayerList.AddRelayer(relayerStr)
	}
	return relayerList, nil
}

func ToRelayerAccAddr(cdc address.Codec, relayerStrs []string) ([]sdk.AccAddress, error) {
	var relayerAccAddrs []sdk.AccAddress
	for _, relayerStr := range relayerStrs {
		relayer, err := cdc.StringToBytes(relayerStr)
		if err != nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		relayerAccAddrs = append(relayerAccAddrs, relayer)
	}
	return relayerAccAddrs, nil
}
