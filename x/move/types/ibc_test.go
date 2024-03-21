package types_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"
)

func Test_ConvertDescriptionToICS721Data(t *testing.T) {
	desc := "this collection is for ics721 testing"
	data, err := types.ConvertDescriptionToICS721Data(desc)
	require.NoError(t, err)

	bz, err := base64.StdEncoding.DecodeString(data)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(`{"initia:description":{"value":"%s"}}`, desc), string(bz))

	// raw data used as description
	_data, err := types.ConvertDescriptionToICS721Data(data)
	require.NoError(t, err)
	require.Equal(t, data, _data)

	// empty description
	data, err = types.ConvertDescriptionToICS721Data("")
	require.NoError(t, err)

	bz, err = base64.StdEncoding.DecodeString(data)
	require.NoError(t, err)
	require.Equal(t, `{}`, string(bz))
}

func Test_ConvertICS721DataToDescription(t *testing.T) {
	desc := "this collection is for ics721 testing"
	data, err := types.ConvertDescriptionToICS721Data(desc)
	require.NoError(t, err)

	_desc, err := types.ConvertICS721DataToDescription(data)
	require.NoError(t, err)
	require.Equal(t, desc, _desc)

	// invalid base64
	_, err = types.ConvertICS721DataToDescription("invalid base64")
	require.Error(t, err)

	// description key not found
	emptyJson := base64.StdEncoding.EncodeToString([]byte(`{}`))
	desc, err = types.ConvertICS721DataToDescription(emptyJson)
	require.NoError(t, err)
	require.Equal(t, emptyJson, desc)
}
