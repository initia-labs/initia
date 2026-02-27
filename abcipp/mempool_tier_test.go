package abcipp

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestBuildTierMatchersAndDistribution(t *testing.T) {
	cfg := PriorityMempoolConfig{
		Tiers: []Tier{
			{Name: "  vip  ", Matcher: func(sdk.Context, sdk.Tx) bool { return true }},
			{Name: "", Matcher: func(sdk.Context, sdk.Tx) bool { return false }},
			{Name: "ignored-nil", Matcher: nil},
		},
	}

	matchers := buildTierMatchers(cfg)
	// Nil matcher tier is skipped, default tier is always appended.
	require.Len(t, matchers, 3)
	require.Equal(t, "vip", matchers[0].Name)
	require.Equal(t, "tier-1", matchers[1].Name)
	require.Equal(t, "default", matchers[2].Name)

	dist := initTierDistribution(matchers)
	require.Equal(t, map[string]uint64{
		"vip":     0,
		"tier-1":  0,
		"default": 0,
	}, dist)
}

func TestSelectTierAndTierName(t *testing.T) {
	mp := newTestPriorityMempool(t, []Tier{
		testTierMatcher("vip"),
		testTierMatcher("standard"),
	})

	ctx := testSDKContext()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())

	txVIP := newTestTx(sender, 0, 1000, "vip")
	txStd := newTestTx(sender, 1, 1000, "standard")
	txUnknown := newTestTx(sender, 2, 1000, "unknown")

	// First matching tier should win.
	require.Equal(t, 0, mp.selectTier(ctx, txVIP))
	require.Equal(t, 1, mp.selectTier(ctx, txStd))
	// Unknown should fall back to appended default tier.
	require.Equal(t, 2, mp.selectTier(ctx, txUnknown))

	require.Equal(t, "vip", mp.tierName(0))
	require.Equal(t, "standard", mp.tierName(1))
	require.Equal(t, "default", mp.tierName(2))
	require.Equal(t, "", mp.tierName(-1))
	require.Equal(t, "", mp.tierName(99))
}

func TestCompareEntriesOrdering(t *testing.T) {
	base := &txEntry{
		tier:     1,
		priority: 100,
		order:    10,
		key:      txKey{sender: "sender-a", nonce: 3},
	}

	// Lower tier outranks higher tier.
	higherTier := &txEntry{tier: 2, priority: 999, order: 0, key: txKey{sender: "zzz", nonce: 0}}
	require.Less(t, compareEntries(base, higherTier), 0)

	// Within same tier, higher priority outranks lower priority.
	lowerPriority := &txEntry{tier: 1, priority: 50, order: 0, key: txKey{sender: "zzz", nonce: 0}}
	require.Less(t, compareEntries(base, lowerPriority), 0)

	// Same tier/priority: smaller order (earlier FIFO) outranks.
	laterOrder := &txEntry{tier: 1, priority: 100, order: 20, key: txKey{sender: "zzz", nonce: 0}}
	require.Less(t, compareEntries(base, laterOrder), 0)

	// Same ranking fields: sender lexicographic order, then nonce.
	senderAfter := &txEntry{tier: 1, priority: 100, order: 10, key: txKey{sender: "sender-b", nonce: 0}}
	require.Less(t, compareEntries(base, senderAfter), 0)
	sameSenderHigherNonce := &txEntry{tier: 1, priority: 100, order: 10, key: txKey{sender: "sender-a", nonce: 9}}
	require.Less(t, compareEntries(base, sameSenderHigherNonce), 0)
}

func TestIsBetterThan(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	existing := &txEntry{tier: 1, priority: 100}

	// Better tier should always win.
	require.True(t, mp.isBetterThan(existing, 0, 1))
	require.False(t, mp.isBetterThan(existing, 2, 1000))

	// Same tier compares only priority.
	require.True(t, mp.isBetterThan(existing, 1, 101))
	require.False(t, mp.isBetterThan(existing, 1, 100))
	require.False(t, mp.isBetterThan(existing, 1, 99))
}
