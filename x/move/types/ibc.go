package types

import (
	"encoding/base64"
	"encoding/json"
)

type ICS721Data struct {
	Name        *string          `json:"initia:name,omitempty"`
	Description *ICS721DataValue `json:"initia:description,omitempty"`
}

type ICS721DataValue struct {
	Value string `json:"value"`
}

// Convert description to base64 encoded json string
// to meet the ics721 spec.
//
// - https://github.com/cosmos/ibc/tree/main/spec/app/ics-721-nft-transfer
func ConvertDescriptionToICS721Data(desc string) (string, error) {
	data := ICS721Data{}

	if desc != "" {
		// if the desc is base64 format, then pass it without wrapping.
		if _, err := base64.StdEncoding.DecodeString(desc); err == nil {
			return desc, nil
		}

		data.Description = &ICS721DataValue{
			Value: desc,
		}
	}

	bz, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bz), nil
}

// Convert description to base64 encoded json string
// to meet the ics721 spec.
//
// - https://github.com/cosmos/ibc/tree/main/spec/app/ics-721-nft-transfer
func ConvertDescriptionToICS721DataWithName(desc, name string) (string, error) {
	data := ICS721Data{
		Name: &name,
	}

	if desc != "" {
		// if the desc is base64 format, then pass it without wrapping.
		if _, err := base64.StdEncoding.DecodeString(desc); err == nil {
			return desc, nil
		}

		data.Description = &ICS721DataValue{
			Value: desc,
		}
	}

	bz, err := json.Marshal(data)
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

	// use normal json unmarshal to allow key not found.
	var ics721Data ICS721Data
	err = json.Unmarshal(bz, &ics721Data)
	if err != nil {
		return "", err
	}

	desc := ""
	if ics721Data.Description != nil {
		desc = ics721Data.Description.Value
	}

	return desc, nil
}
