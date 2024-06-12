package types_test

import (
	"testing"

	"github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"
)

func Test_MetadataAddressFromDenom(t *testing.T) {
	denom := "move/944f8dd8dc49f96c25fea9849f16436dcfa6d564eec802f3ef7f8b3ea85368ff"
	addr, err := types.MetadataAddressFromDenom(denom)
	require.NoError(t, err)
	require.Equal(t, "0x944f8dd8dc49f96c25fea9849f16436dcfa6d564eec802f3ef7f8b3ea85368ff", addr.String())

	// upper case
	denom = "move/944f8dd8dc49f96c25fea9849f16436dcfa6d564eec802f3ef7f8b3ea85368fF"
	_, err = types.MetadataAddressFromDenom(denom)
	require.Error(t, err)
}
