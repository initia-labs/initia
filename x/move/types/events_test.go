package types_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func testSDKContext(t *testing.T) sdk.Context {
	t.Helper()

	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	require.NoError(t, cms.LoadLatestVersion())

	return sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())
}

func TestEmitContractEventsNestedMap(t *testing.T) {
	ctx := testSDKContext(t)

	eventData := `{"shape":{"Circle":{"radius":"1"}}}`
	types.EmitContractEvents(ctx, []vmtypes.JsonEvent{{
		TypeTag:   "0x1::Counter::TestEvent",
		EventData: eventData,
	}})

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	e := events[0]
	require.Equal(t, types.EventTypeMove, e.Type)
	require.Equal(t, types.AttributeKeyTypeTag, e.Attributes[0].Key)
	require.Equal(t, "0x1::Counter::TestEvent", e.Attributes[0].Value)
	require.Equal(t, types.AttributeKeyData, e.Attributes[1].Key)
	require.Equal(t, eventData, e.Attributes[1].Value)
	require.Len(t, e.Attributes, 2)
}

func TestEmitContractEventsArray(t *testing.T) {
	ctx := testSDKContext(t)

	eventData := `{"vals":[1,2]}`
	types.EmitContractEvents(ctx, []vmtypes.JsonEvent{{
		TypeTag:   "0x1::Counter::TestEvent",
		EventData: eventData,
	}})

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	e := events[0]
	require.Equal(t, types.EventTypeMove, e.Type)
	require.Len(t, e.Attributes, 2)
}

func TestEmitContractEventsInvalidJSON(t *testing.T) {
	ctx := testSDKContext(t)

	eventData := "not-json"
	types.EmitContractEvents(ctx, []vmtypes.JsonEvent{{
		TypeTag:   "0x1::Counter::TestEvent",
		EventData: eventData,
	}})

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	e := events[0]
	require.Equal(t, types.EventTypeMove, e.Type)
	require.Len(t, e.Attributes, 2)
	require.Equal(t, types.AttributeKeyTypeTag, e.Attributes[0].Key)
	require.Equal(t, types.AttributeKeyData, e.Attributes[1].Key)
	require.Equal(t, eventData, e.Attributes[1].Value)
}
