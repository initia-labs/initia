package hook

import (
	"encoding/json"
	"strings"
)

const permsMetadataKey = "perm_channels"

type PermsMetadata struct {
	PermChannels []PortChannelID `json:"perm_channels"`
}

type PortChannelID struct {
	PortID    string `json:"port_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

func hasPermChannels(metadata []byte) (hasPerms bool, data PermsMetadata) {
	if !jsonStringHasKey(string(metadata), permsMetadataKey) {
		return false, data
	}

	decoder := json.NewDecoder(strings.NewReader(string(metadata)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return false, data
	}
	return true, data
}

// jsonStringHasKey parses the metadata string as a json object and checks if it contains the key.
func jsonStringHasKey(metadata, key string) bool {
	if len(metadata) == 0 {
		return false
	}

	jsonObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(metadata), &jsonObject)
	if err != nil {
		return false
	}

	_, ok := jsonObject[key]
	return ok
}
