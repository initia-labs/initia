package keeper

import (
	"github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
)

// UnmarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes.
func (k Keeper) UnmarshalClassTrace(bz []byte) (types.ClassTrace, error) {
	var classTrace types.ClassTrace
	if err := k.cdc.Unmarshal(bz, &classTrace); err != nil {
		return types.ClassTrace{}, err
	}

	return classTrace, nil
}

// MustUnmarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalClassTrace(bz []byte) types.ClassTrace {
	var classTrace types.ClassTrace
	k.cdc.MustUnmarshal(bz, &classTrace)
	return classTrace
}

// MarshalClassTrace attempts to encode an ClassTrace object and returns the
// raw encoded bytes.
func (k Keeper) MarshalClassTrace(classTrace types.ClassTrace) ([]byte, error) {
	return k.cdc.Marshal(&classTrace)
}

// MustMarshalClassTrace attempts to encode an ClassTrace object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalClassTrace(classTrace types.ClassTrace) []byte {
	return k.cdc.MustMarshal(&classTrace)
}
