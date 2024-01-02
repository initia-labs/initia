package types

import (
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Pools defines denom and sdk.Coins wrapper to represents
// rewards pools for multi-token staking
type Pools []Pool

// Sum returns sum of pool tokens
func (p Pools) Sum() (coins sdk.Coins) {
	for _, p := range p {
		coins = coins.Add(p.Coins...)
	}

	return
}

// Add adds two sets of Pools
func (pools Pools) Add(poolsB ...Pool) Pools {
	return pools.safeAdd(poolsB)
}

// Add will perform addition of two Pools sets.
func (pools Pools) safeAdd(poolsB Pools) Pools {
	sum := ([]Pool)(nil)
	indexA, indexB := 0, 0
	lenA, lenB := len(pools), len(poolsB)

	for {
		if indexA == lenA {
			if indexB == lenB {
				// return nil pools if both sets are empty
				return sum
			}

			// return set B (excluding zero pools) if set A is empty
			return append(sum, removeZeroPools(poolsB[indexB:])...)
		} else if indexB == lenB {
			// return set A (excluding zero pools) if set B is empty
			return append(sum, removeZeroPools(pools[indexA:])...)
		}

		poolA, poolB := pools[indexA], poolsB[indexB]

		switch strings.Compare(poolA.Denom, poolB.Denom) {
		case -1: // pool A denom < pool B denom
			if !poolA.IsEmpty() {
				sum = append(sum, poolA)
			}

			indexA++

		case 0: // pool A denom == pool B denom
			res := poolA.Add(poolB)
			if !res.IsEmpty() {
				sum = append(sum, res)
			}

			indexA++
			indexB++

		case 1: // pool A denom > pool B denom
			if !poolB.IsEmpty() {
				sum = append(sum, poolB)
			}

			indexB++
		}
	}
}

// Sub subtracts a set of Pools from another (adds the inverse).
func (pools Pools) Sub(poolsB Pools) Pools {
	diff, hasNeg := pools.SafeSub(poolsB)
	if hasNeg {
		panic("negative pool coins")
	}

	return diff
}

// SafeSub performs the same arithmetic as Sub but returns a boolean if any
// negative Pool coins amount was returned.
func (pools Pools) SafeSub(poolsB Pools) (Pools, bool) {
	diff := pools.safeAdd(poolsB.negative())
	return diff, diff.IsAnyNegative()
}

// IsAnyNegative returns true if there is at least one coin whose amount
// is negative; returns false otherwise. It returns false if the Pools set
// is empty too.
func (pools Pools) IsAnyNegative() bool {
	for _, pool := range pools {
		if pool.Coins.IsAnyNegative() {
			return true
		}
	}

	return false
}

// negative returns a set of coins with all amount negative.
func (pools Pools) negative() Pools {
	res := make([]Pool, 0, len(pools))
	for _, pool := range pools {
		coins := make([]sdk.Coin, 0, len(pool.Coins))
		for _, coin := range pool.Coins {
			coins = append(coins, sdk.Coin{
				Denom:  coin.Denom,
				Amount: coin.Amount.Neg(),
			})
		}

		res = append(res, Pool{
			Denom: pool.Denom,
			Coins: coins,
		})
	}
	return res
}

// CoinsOf returns the Coins of a denom from Pools
func (pools Pools) CoinsOf(denom string) sdk.Coins {
	switch len(pools) {
	case 0:
		return sdk.Coins{}

	case 1:
		coin := pools[0]
		if coin.Denom == denom {
			return coin.Coins
		}
		return sdk.Coins{}

	default:
		midIdx := len(pools) / 2 // 2:1, 3:1, 4:2
		coin := pools[midIdx]

		switch {
		case denom < coin.Denom:
			return pools[:midIdx].CoinsOf(denom)
		case denom == coin.Denom:
			return coin.Coins
		default:
			return pools[midIdx+1:].CoinsOf(denom)
		}
	}
}

// IsZero returns whether all pools are empty
func (pools Pools) IsEmpty() bool {
	for _, pool := range pools {
		if !pool.IsEmpty() {
			return false
		}
	}
	return true
}

// IsEqual returns true if the two sets of Pools have the same value.
func (pools Pools) IsEqual(poolsB Pools) bool {
	if len(pools) != len(poolsB) {
		return false
	}

	pools = pools.Sort()
	poolsB = poolsB.Sort()

	for i := 0; i < len(pools); i++ {
		if !pools[i].IsEqual(poolsB[i]) {
			return false
		}
	}

	return true
}

// String implements the Stringer interface for Pools. It returns a
// human-readable representation of Pools.
func (pools Pools) String() string {
	if len(pools) == 0 {
		return ""
	}

	out := ""
	for _, pool := range pools {
		out += fmt.Sprintf("%v,", pool.String())
	}

	return out[:len(out)-1]
}

//-----------------------------------------------------------------------------
// Sort interface

// Len implements sort.Interface for Pools
func (p Pools) Len() int { return len(p) }

// Less implements sort.Interface for Pools
func (p Pools) Less(i, j int) bool { return p[i].Denom < p[j].Denom }

// Swap implements sort.Interface for Pools
func (p Pools) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

var _ sort.Interface = Pools{}

// Sort is a helper function to sort the set of p in-place
func (p Pools) Sort() Pools {
	sort.Sort(p)
	return p
}

//-----------------------------------------------------------------------------
// Pool functions

// NewPool return new pool instance
func NewPool(denom string, coins sdk.Coins) Pool {
	return Pool{denom, coins}
}

// IsEmpty returns wether the pool coins are empty or not
func (pool Pool) IsEmpty() bool {
	return pool.Coins.IsZero()
}

// Add adds amounts of two pool coins with same denom.
func (pool Pool) Add(poolB Pool) Pool {
	if pool.Denom != poolB.Denom {
		panic(fmt.Sprintf("pool denom different: %v %v\n", pool.Denom, poolB.Denom))
	}
	return Pool{pool.Denom, pool.Coins.Add(poolB.Coins...)}
}

// Sub subtracts amounts of two pool coins with same denom.
func (pool Pool) Sub(poolB Pool) Pool {
	if pool.Denom != poolB.Denom {
		panic(fmt.Sprintf("pool denom different: %v %v\n", pool.Denom, poolB.Denom))
	}
	res := Pool{pool.Denom, pool.Coins.Sub(poolB.Coins...)}
	if res.Coins.IsAnyNegative() {
		panic("negative pool coins")
	}
	return res
}

func removeZeroPools(pools Pools) Pools {
	result := make([]Pool, 0, len(pools))

	for _, pool := range pools {
		if !pool.IsEmpty() {
			result = append(result, pool)
		}
	}

	return result
}

// IsEqual returns true if the two sets of Pools have the same value.
func (pool Pool) IsEqual(other Pool) bool {
	if pool.Denom != other.Denom {
		panic(fmt.Sprintf("invalid pool denominations; %s, %s", pool.Denom, other.Denom))
	}

	return pool.Coins.Equal(other.Coins)
}
