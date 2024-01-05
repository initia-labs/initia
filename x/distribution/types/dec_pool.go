package types

import (
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DecPools defines denom and sdk.DecCoins wrapper to represents
// rewards pools for multi-token staking
type DecPools []DecPool

// NewDecPoolsFromPools create DecPools from Pools
func NewDecPoolsFromPools(pools Pools) DecPools {
	decPools := DecPools{}
	for _, p := range pools {
		decPools = append(decPools, NewDecPool(p.Denom, sdk.NewDecCoinsFromCoins(p.Coins...)))
	}

	return decPools
}

// Sum returns sum of pool tokens
func (p DecPools) Sum() (coins sdk.DecCoins) {
	for _, p := range p {
		coins = coins.Add(p.DecCoins...)
	}

	return
}

// Add adds two sets of DecPools
func (pools DecPools) Add(poolsB ...DecPool) DecPools {
	return pools.safeAdd(poolsB)
}

// Add will perform addition of two DecPools sets.
func (pools DecPools) safeAdd(poolsB DecPools) DecPools {
	sum := ([]DecPool)(nil)
	indexA, indexB := 0, 0
	lenA, lenB := len(pools), len(poolsB)

	for {
		if indexA == lenA {
			if indexB == lenB {
				// return nil pools if both sets are empty
				return sum
			}

			// return set B (excluding zero pools) if set A is empty
			return append(sum, removeZeroDecPools(poolsB[indexB:])...)
		} else if indexB == lenB {
			// return set A (excluding zero pools) if set B is empty
			return append(sum, removeZeroDecPools(pools[indexA:])...)
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

// Sub subtracts a set of DecPools from another (adds the inverse).
func (pools DecPools) Sub(poolsB DecPools) DecPools {
	diff, hasNeg := pools.SafeSub(poolsB)
	if hasNeg {
		panic("negative pool coins")
	}

	return diff
}

// SafeSub performs the same arithmetic as Sub but returns a boolean if any
// negative DecPool coins amount was returned.
func (pools DecPools) SafeSub(poolsB DecPools) (DecPools, bool) {
	diff := pools.safeAdd(poolsB.negative())
	return diff, diff.IsAnyNegative()
}

// IsAnyNegative returns true if there is at least one coin whose amount
// is negative; returns false otherwise. It returns false if the DecPools set
// is empty too.
func (pools DecPools) IsAnyNegative() bool {
	for _, pool := range pools {
		if pool.DecCoins.IsAnyNegative() {
			return true
		}
	}

	return false
}

// negative returns a set of coins with all amount negative.
func (pools DecPools) negative() DecPools {
	res := make([]DecPool, 0, len(pools))
	for _, pool := range pools {
		decCoins := make([]sdk.DecCoin, 0, len(pool.DecCoins))
		for _, decCoin := range pool.DecCoins {
			decCoins = append(decCoins, sdk.DecCoin{
				Denom:  decCoin.Denom,
				Amount: decCoin.Amount.Neg(),
			})
		}

		res = append(res, DecPool{
			Denom:    pool.Denom,
			DecCoins: decCoins,
		})
	}
	return res
}

// CoinsOf returns the Coins of a denom from DecPools
func (pools DecPools) CoinsOf(denom string) sdk.DecCoins {
	switch len(pools) {
	case 0:
		return sdk.DecCoins{}

	case 1:
		pool := pools[0]
		if pool.Denom == denom {
			return pool.DecCoins
		}
		return sdk.DecCoins{}

	default:
		midIdx := len(pools) / 2 // 2:1, 3:1, 4:2
		decPool := pools[midIdx]

		switch {
		case denom < decPool.Denom:
			return pools[:midIdx].CoinsOf(denom)
		case denom == decPool.Denom:
			return decPool.DecCoins
		default:
			return pools[midIdx+1:].CoinsOf(denom)
		}
	}
}

// IsZero returns whether all pools are empty
func (pools DecPools) IsEmpty() bool {
	for _, pool := range pools {
		if !pool.IsEmpty() {
			return false
		}
	}
	return true
}

// IsEqual returns true if the two sets of DecPools have the same value.
func (pools DecPools) IsEqual(poolsB DecPools) bool {
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

// TruncateDecimal returns the pools with truncated decimals and returns the
// change. Note, it will not return any zero-amount pools in either the truncated or
// change pools.
func (pools DecPools) TruncateDecimal() (truncatedDecPools Pools, changeDecPools DecPools) {
	for _, pool := range pools {
		truncated, change := pool.TruncateDecimal()
		if !truncated.IsEmpty() {
			truncatedDecPools = truncatedDecPools.Add(truncated)
		}
		if !change.IsEmpty() {
			changeDecPools = changeDecPools.Add(change)
		}
	}

	return truncatedDecPools, changeDecPools
}

// Intersect will return a new set of pools which contains the minimum pool Coins
// for common denoms found in both `pools` and `poolsB`. For denoms not common
// to both `pools` and `poolsB` the minimum is considered to be 0, thus they
// are not added to the final set. In other words, trim any denom amount from
// pool which exceeds that of poolB, such that (pool.Intersect(poolB)).IsLTE(poolB).
func (pools DecPools) Intersect(poolsB DecPools) DecPools {
	res := make([]DecPool, len(pools))
	for i, pool := range pools {
		minDecPool := DecPool{
			Denom:    pool.Denom,
			DecCoins: pool.DecCoins.Intersect(poolsB.CoinsOf(pool.Denom)),
		}
		res[i] = minDecPool
	}
	return removeZeroDecPools(res)
}

// String implements the Stringer interface for DecPools. It returns a
// human-readable representation of DecPools.
func (pools DecPools) String() string {
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

// Len implements sort.Interface for DecPools
func (p DecPools) Len() int { return len(p) }

// Less implements sort.Interface for DecPools
func (p DecPools) Less(i, j int) bool { return p[i].Denom < p[j].Denom }

// Swap implements sort.Interface for DecPools
func (p DecPools) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

var _ sort.Interface = DecPools{}

// Sort is a helper function to sort the set of p in-place
func (p DecPools) Sort() DecPools {
	sort.Sort(p)
	return p
}

//-----------------------------------------------------------------------------
// DecPool functions

// NewDecPool return new pool instance
func NewDecPool(denom string, coins sdk.DecCoins) DecPool {
	return DecPool{denom, coins}
}

// IsEmpty returns wether the pool coins are empty or not
func (pool DecPool) IsEmpty() bool {
	return pool.DecCoins.IsZero()
}

// Add adds amounts of two pool coins with same denom.
func (pool DecPool) Add(poolB DecPool) DecPool {
	if pool.Denom != poolB.Denom {
		panic(fmt.Sprintf("pool denom different: %v %v\n", pool.Denom, poolB.Denom))
	}
	return DecPool{pool.Denom, pool.DecCoins.Add(poolB.DecCoins...)}
}

// Sub subtracts amounts of two pool coins with same denom.
func (pool DecPool) Sub(poolB DecPool) DecPool {
	if pool.Denom != poolB.Denom {
		panic(fmt.Sprintf("pool denom different: %v %v\n", pool.Denom, poolB.Denom))
	}
	res := DecPool{pool.Denom, pool.DecCoins.Sub(poolB.DecCoins)}
	if res.DecCoins.IsAnyNegative() {
		panic("negative pool coins")
	}
	return res
}

// TruncateDecimal returns a DecPool with a DecPool for truncated decimal and a DecPool for the
// change. Note, the change may be zero.
func (pool DecPool) TruncateDecimal() (Pool, DecPool) {
	truncated, change := pool.DecCoins.TruncateDecimal()
	return NewPool(pool.Denom, sdk.NewCoins(truncated...)), NewDecPool(pool.Denom, change)
}

func removeZeroDecPools(pools DecPools) DecPools {
	result := make([]DecPool, 0, len(pools))

	for _, pool := range pools {
		if !pool.IsEmpty() {
			result = append(result, pool)
		}
	}

	return result
}

// IsEqual returns true if the two sets of DecPools have the same value.
func (pool DecPool) IsEqual(other DecPool) bool {
	if pool.Denom != other.Denom {
		panic(fmt.Sprintf("invalid pool denominations; %s, %s", pool.Denom, other.Denom))
	}

	return pool.DecCoins.Equal(other.DecCoins)
}
