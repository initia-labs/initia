package types_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	"github.com/stretchr/testify/require"
)

func Test_ConvertDescriptionToICS721Data(t *testing.T) {
	collectionName := "ics721 testing name "
	collectionDesc := "this collection is for ics721 testing"
	classData, err := types.ConvertClassDataToICS721(collectionName, collectionDesc)
	require.NoError(t, err)

	bz, err := base64.StdEncoding.DecodeString(classData)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(`{"name":"%s","description":"%s"}`, collectionName, collectionDesc), string(bz))

	// raw data used as description
	_collectionName, _collectionDesc, err := types.ConvertClassDataFromICS721(classData)
	require.NoError(t, err)
	require.Equal(t, collectionDesc, _collectionDesc)
	require.Equal(t, collectionName, _collectionName)

	// empty description
	classData, err = types.ConvertClassDataToICS721("", "")
	require.NoError(t, err)

	bz, err = base64.StdEncoding.DecodeString(classData)
	require.NoError(t, err)
	require.Equal(t, `{}`, string(bz))
}

func Test_ConvertICS721DataToDescription(t *testing.T) {
	collectionName := "ics721 testing name "
	collectionDesc := "this collection is for ics721 testing"
	data, err := types.ConvertClassDataToICS721(collectionName, collectionDesc)
	require.NoError(t, err)

	_name, _desc, err := types.ConvertClassDataFromICS721(data)
	require.NoError(t, err)
	require.Equal(t, collectionDesc, _desc)
	require.Equal(t, collectionName, _name)

	// invalid base64
	_, _, err = types.ConvertClassDataFromICS721("invalid base64")
	require.Error(t, err)

	// description key not found
	emptyJson := base64.StdEncoding.EncodeToString([]byte(`{}`))
	name, desc, err := types.ConvertClassDataFromICS721(emptyJson)
	require.NoError(t, err)
	require.Equal(t, "", desc)
	require.Equal(t, "", name)
}
