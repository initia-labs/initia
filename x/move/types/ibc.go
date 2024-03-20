package types

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
)

type ICS721Data struct {
	Description ICS721DataValue `json:"initia:description"`
}

type ICS721DataValue struct {
	Value string `json:"value"`
}

// Convert description to base64 encoded json string
// to meet the ics721 spec.
//
// - https://github.com/cosmos/ibc/tree/main/spec/app/ics-721-nft-transfer
func ConvertDescriptionToICS721Data(desc string) (string, error) {
	bz, err := json.Marshal(ICS721Data{
		Description: ICS721DataValue{
			Value: desc,
		},
	})
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bz), nil
}

func ConvertICS721DataToDescription(data string) (string, error) {
	bz, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	decoder := json.NewDecoder(bytes.NewReader(bz))
	decoder.DisallowUnknownFields()

	var ics721Data ICS721Data
	err = decoder.Decode(&ics721Data)
	if err != nil {
		return "", err
	}

	return ics721Data.Description.Value, nil
}
