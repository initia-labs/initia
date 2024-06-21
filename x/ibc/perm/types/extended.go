package types

import (
	"slices"

	"cosmossdk.io/errors"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (rl *PermissionedRelayersList) HasRelayer(addrStr string) bool {
	return slices.Contains(rl.Relayers, addrStr)
}
func (rl *PermissionedRelayersList) AddRelayer(addrStr string) {
	rl.Relayers = append(rl.Relayers, addrStr)
}

func (rl *PermissionedRelayersList) GetAccAddr(ac address.Codec) ([]sdk.AccAddress, error) {

	var relayerStrs []sdk.AccAddress
	for _, relayer := range rl.Relayers {

		relayer, err := ac.StringToBytes(relayer)
		if err != nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		relayerStrs = append(relayerStrs, relayer)
	}

	return relayerStrs, nil
}

func ToRelayerList(ac address.Codec, relayers []sdk.AccAddress) (PermissionedRelayersList, error) {
	var relayerList PermissionedRelayersList
	for _, relayer := range relayers {
		relayerStr, error := ac.BytesToString(relayer)
		if error != nil {
			return relayerList, errors.Wrapf(sdkerrors.ErrInvalidAddress, "relayer address could not be converted to string : %v", error)
		}
		relayerList.AddRelayer(relayerStr)
	}
	return relayerList, nil
}

func ToRelayerAccAddr(ac address.Codec, relayerStrs []string) ([]sdk.AccAddress, error) {
	var relayerAccAddrs []sdk.AccAddress
	for _, relayerStr := range relayerStrs {
		relayer, err := ac.StringToBytes(relayerStr)
		if err != nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		relayerAccAddrs = append(relayerAccAddrs, relayer)
	}
	return relayerAccAddrs, nil
}
