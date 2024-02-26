package hook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_jsonStringHasKey(t *testing.T) {
	require.True(t, jsonStringHasKey(`{
		"hello": 1,
		"bye": {
			"abc": "def"
		}
	}`, "hello"))
	require.True(t, jsonStringHasKey(`{
		"hello": 1,
		"bye": {
			"abc": "def"
		}
	}`, "bye"))
	require.False(t, jsonStringHasKey(`{
		"hello": 1,
		"bye": {
			"abc": "def"
		}
	}`, "HI"))
}

func Test_hasPermChannels(t *testing.T) {
	ok, metadata := hasPermChannels([]byte("{\"perm_channels\":[{\"port_id\":\"transfer\", \"channel_id\":\"channel-0\"}, {\"port_id\":\"icqhost\", \"channel_id\":\"channel-1\"}]}"))
	require.True(t, ok)
	require.Equal(t, PermsMetadata{
		PermChannels: []PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-0",
			},
			{
				PortID:    "icqhost",
				ChannelID: "channel-1",
			},
		},
	}, metadata)

	ok, _ = hasPermChannels([]byte("{\"perm_channel\":[{\"port_id\":\"transfer\", \"channel_id\":\"channel-0\"}, {\"port_id\":\"icqhost\", \"channel_id\":\"channel-1\"}]}"))
	require.False(t, ok)

	ok, _ = hasPermChannels([]byte("{\"perm_channels\":[{\"port\":\"transfer\", \"channel\":\"channel-0\"}, {\"port_id\":\"icqhost\", \"channel_id\":\"channel-1\"}]}"))
	require.False(t, ok)
}
