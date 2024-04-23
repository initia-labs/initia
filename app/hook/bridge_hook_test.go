package hook_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/initia/app/hook"
)

var _ hook.ChannelKeeper = MockChannelKeeper{}
var _ hook.PermKeeper = MockPermKeeper{}

type MockChannelKeeper struct {
	sequenceIDs map[string]uint64
}

func (k MockChannelKeeper) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	seq, ok := k.sequenceIDs[portID+"/"+channelID]
	return seq, ok
}

type MockPermKeeper struct {
	perms map[string]sdk.AccAddress
}

func (k MockPermKeeper) HasPermission(ctx context.Context, portID, channelID string, relayer sdk.AccAddress) (bool, error) {
	return k.perms[portID+"/"+channelID].Equals(relayer), nil
}

func (k MockPermKeeper) SetPermissionedRelayer(ctx context.Context, portID, channelID string, relayer sdk.AccAddress) error {
	k.perms[portID+"/"+channelID] = relayer
	return nil
}

func setup() (context.Context, hook.BridgeHook) {
	h := hook.NewBridgeHook(MockChannelKeeper{
		sequenceIDs: map[string]uint64{
			"transfer/channel-0": 1,
			"transfer/channel-1": 2,
			"transfer/channel-2": 1,
		},
	}, MockPermKeeper{
		perms: make(map[string]sdk.AccAddress),
	}, address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()))

	ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	return ctx, h
}

func acc_addr() sdk.AccAddress {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
}

func Test_BridgeHook_BridgeCreated(t *testing.T) {
	ctx, h := setup()

	metadata, err := json.Marshal(hook.PermsMetadata{
		PermChannels: []hook.PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-0",
			},
		},
	})
	require.NoError(t, err)

	metadata2, err := json.Marshal(hook.PermsMetadata{
		PermChannels: []hook.PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-1",
			},
		},
	})
	require.NoError(t, err)

	addr := acc_addr()
	err = h.BridgeCreated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: addr.String(),
		Metadata:   metadata,
	})
	require.NoError(t, err)

	// cannot take non-1 sequence channel
	err = h.BridgeCreated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: addr.String(),
		Metadata:   metadata2,
	})
	require.Error(t, err)

	// check permission is applied
	ok, err := h.IBCPermKeeper.HasPermission(ctx, "transfer", "channel-0", addr)
	require.NoError(t, err)
	require.True(t, ok)

	// check permission is applied
	ok, err = h.IBCPermKeeper.HasPermission(ctx, "transfer", "channel-0", acc_addr())
	require.NoError(t, err)
	require.False(t, ok)
}

func Test_BridgeHook_ChallengerUpdated(t *testing.T) {
	ctx, h := setup()

	metadata, err := json.Marshal(hook.PermsMetadata{
		PermChannels: []hook.PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-0",
			},
		},
	})
	require.NoError(t, err)

	addr := acc_addr()
	err = h.BridgeCreated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: addr.String(),
		Metadata:   metadata,
	})
	require.NoError(t, err)

	newAddr := acc_addr()
	err = h.BridgeChallengerUpdated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: newAddr.String(),
		Metadata:   metadata,
	})
	require.NoError(t, err)

	// check permission is applied
	ok, err := h.IBCPermKeeper.HasPermission(ctx, "transfer", "channel-0", newAddr)
	require.NoError(t, err)
	require.True(t, ok)
}

func Test_BridgeHook_MetadataUpdated(t *testing.T) {
	ctx, h := setup()

	metadata, err := json.Marshal(hook.PermsMetadata{
		PermChannels: []hook.PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-0",
			},
		},
	})
	require.NoError(t, err)

	addr := acc_addr()
	err = h.BridgeCreated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: addr.String(),
		Metadata:   metadata,
	})
	require.NoError(t, err)

	// new metadata
	metadata, err = json.Marshal(hook.PermsMetadata{
		PermChannels: []hook.PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-0",
			},
			{
				PortID:    "transfer",
				ChannelID: "channel-1",
			},
		},
	})
	require.NoError(t, err)

	// cannot take non-1 sequence channel
	err = h.BridgeMetadataUpdated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: addr.String(),
		Metadata:   metadata,
	})
	require.Error(t, err)

	// new metadata
	metadata, err = json.Marshal(hook.PermsMetadata{
		PermChannels: []hook.PortChannelID{
			{
				PortID:    "transfer",
				ChannelID: "channel-0",
			},
			{
				PortID:    "transfer",
				ChannelID: "channel-2",
			},
		},
	})
	require.NoError(t, err)

	err = h.BridgeMetadataUpdated(ctx, 1, ophosttypes.BridgeConfig{
		Challenger: addr.String(),
		Metadata:   metadata,
	})
	require.NoError(t, err)

	// check permission is applied
	ok, err := h.IBCPermKeeper.HasPermission(ctx, "transfer", "channel-2", addr)
	require.NoError(t, err)
	require.True(t, ok)
}
