package types

import (
	"encoding/base64"
	"encoding/json"
)

// ICS721Data represents the data structure for original name and description.
type ICS721Data struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ConvertClassDataToICS721 converts collection name and collection description to base64 encoded json string
// to meet the ics721 spec.
//
// - https://github.com/cosmos/ibc/tree/main/spec/app/ics-721-nft-transfer
func ConvertClassDataToICS721(name string, desc string) (string, error) {
	return convertDataToICS721(name, desc)
}

// ConvertTokenDataToICS721 converts token description to base64 encoded json string
// to meet the ics721 spec.
func ConvertTokenDataToICS721(desc string) (string, error) {
	return convertDataToICS721("", desc)
}

func convertDataToICS721(name string, desc string) (string, error) {
	data := ICS721Data{
		Name:        name,
		Description: desc,
	}

	bz, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bz), nil
}

// ConvertClassDataFromICS721 converts base64 encoded json string to collection name and collection description.
func ConvertClassDataFromICS721(data string) (name string, desc string, err error) {
	return convertDataFromICS721(data)
}

// ConvertTokenDataFromICS721 converts base64 encoded json string to token description.
func ConvertTokenDataFromICS721(data string) (desc string, err error) {
	_, desc, err = convertDataFromICS721(data)
	return
}

func convertDataFromICS721(data string) (name string, desc string, err error) {
	bz, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", "", err
	}

	// use normal json unmarshal to allow key not found.
	var ics721Data ICS721Data
	err = json.Unmarshal(bz, &ics721Data)
	if err != nil {
		return "", "", err
	}
	return ics721Data.Name, ics721Data.Description, nil
}
