package main

import (
	"encoding/json"
	"math/big"
	"strconv"
)

// Coin hold some amount of one currency.
type Coin struct {
	Denom  string `json:"denom"`
	Amount uint64 `json:"amount"`
}

// Minus substracts another coin from the given coin
func (coin *Coin) Minus(another Coin) Coin {
	if coin.Denom != another.Denom {
		panic("only same denom coins can be subtracted")
	}

	coinBI := new(big.Int).SetUint64(coin.Amount)
	anotherBI := new(big.Int).SetUint64(another.Amount)
	diffBI := new(big.Int).Sub(coinBI, anotherBI)

	return Coin{
		Denom:  coin.Denom,
		Amount: diffBI.Uint64(),
	}
}

// UnmarshalJSON unmarshals json string into Coin struct
func (coin *Coin) UnmarshalJSON(data []byte) error {
	type Alias Coin
	aux := &struct {
		Amount string `json:"amount"`
		*Alias
	}{
		Alias: (*Alias)(coin),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	amount, err := strconv.ParseInt(aux.Amount, 10, 64)
	if err != nil {
		panic(err)
	}

	// amounts cannot be negative
	if amount < 0 {
		amount = 0
	}

	coin.Amount = uint64(amount)
	return nil
}

// SystemMetrics represents metrics for the platform.
type SystemMetrics struct {
	Users map[string]*UserMetricsV2 `json:"users"`
}

// MetricsV2 defines the numbers that are tracked
type MetricsV2 struct {
	// StakeBased Metrics
	TotalAmountStaked  Coin `json:"total_amount_staked"`
	StakeEarned        Coin `json:"stake_earned"`
	StakeLost          Coin `json:"stake_lost"`
	TotalAmountAtStake Coin `json:"total_amount_at_stake"`
}

// UserMetricsV2 a summary of different metrics per user
type UserMetricsV2 struct {
	AvailableStake Coin `json:"available_stake"`

	// For each community
	CommunityMetrics map[string]*CommunityMetrics `json:"community_metrics"`
}

// CommunityMetrics summary of metrics by community
type CommunityMetrics struct {
	CommunityID string     `json:"community_id"`
	Metrics     *MetricsV2 `json:"metrics"`
}
