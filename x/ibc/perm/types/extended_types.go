package types

import (
	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (rl *PermissionedRelayerList) Contains(addr string) bool {
	for _, r := range rl.Relayers {
		if r == addr {
			return true
		}
	}
	return false
}
func (rl *PermissionedRelayerList) Add(addr string) {
	rl.Relayers = append(rl.Relayers, addr)
}

func (rl *PermissionedRelayerList) ToAccAddress(cdc address.Codec) ([]sdk.AccAddress, error) {

	var relayers []sdk.AccAddress
	for _, relayer := range rl.Relayers {

		relayer, err := cdc.StringToBytes(relayer)
		if err != nil {
			return nil, err
		}
		relayers = append(relayers, relayer)
	}

	return relayers, nil
}

func ToRelayerList(relayers []sdk.AccAddress) PermissionedRelayerList {
	var relayerList PermissionedRelayerList
	for _, relayer := range relayers {
		relayerList.Add(relayer.String())
	}
	return relayerList
}
