package abcipp

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type tierMatcher struct {
	Name    string
	Matcher TierMatcher
}

// compareEntries orders txEntries by tier, priority, order, sender, and nonce.
func compareEntries(a, b any) int {
	left := a.(*txEntry)
	right := b.(*txEntry)

	// Lower tier index wins.
	if left.tier != right.tier {
		if left.tier < right.tier {
			return -1
		}
		return 1
	}

	// Higher priority wins within the same tier.
	if left.priority != right.priority {
		if left.priority > right.priority {
			return -1
		}
		return 1
	}

	// Preserve FIFO for same-tier/same-priority entries.
	if left.order != right.order {
		if left.order < right.order {
			return -1
		}
		return 1
	}

	// Keep deterministic ordering even when all ranking fields match.
	if left.key.sender != right.key.sender {
		return strings.Compare(left.key.sender, right.key.sender)
	}

	switch {
	case left.key.nonce < right.key.nonce:
		return -1
	case left.key.nonce > right.key.nonce:
		return 1
	default:
		return 0
	}
}

// selectTier returns the index of the first matching tier matcher for the tx.
func (p *PriorityMempool) selectTier(ctx sdk.Context, tx sdk.Tx) int {
	for idx, tier := range p.tiers {
		if tier.Matcher == nil || tier.Matcher(ctx, tx) {
			return idx
		}
	}
	return len(p.tiers) - 1
}

// tierName returns the configured name for a tier index, or empty if invalid.
func (p *PriorityMempool) tierName(idx int) string {
	if idx < 0 || idx >= len(p.tiers) {
		return ""
	}
	return p.tiers[idx].Name
}

// buildTierMatchers canonicalizes configured tiers and appends a default tier.
func buildTierMatchers(cfg PriorityMempoolConfig) []tierMatcher {
	matchers := make([]tierMatcher, 0, len(cfg.Tiers)+1)
	for idx, tier := range cfg.Tiers {
		if tier.Matcher == nil {
			continue
		}

		name := strings.TrimSpace(tier.Name)
		if name == "" {
			name = fmt.Sprintf("tier-%d", idx)
		}

		matchers = append(matchers, tierMatcher{
			Name:    name,
			Matcher: tier.Matcher,
		})
	}

	matchers = append(matchers, tierMatcher{
		Name:    "default",
		Matcher: func(ctx sdk.Context, tx sdk.Tx) bool { return true },
	})

	return matchers
}

// initTierDistribution creates a zeroed counter map for each named tier.
func initTierDistribution(tiers []tierMatcher) map[string]uint64 {
	dist := make(map[string]uint64, len(tiers))
	for _, tier := range tiers {
		if tier.Name == "" {
			continue
		}
		dist[tier.Name] = 0
	}
	return dist
}

// isBetterThan determines whether a new entry should outrank an existing one.
func (p *PriorityMempool) isBetterThan(entry *txEntry, tier int, priority int64) bool {
	if tier != entry.tier {
		return tier < entry.tier
	}
	return priority > entry.priority
}
