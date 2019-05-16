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

	amount, err := strconv.ParseUint(aux.Amount, 10, 64)
	if err != nil {
		panic(err)
	}

	coin.Amount = amount
	return nil
}

// MetricsSummary represents metrics for the platform.
type MetricsSummary struct {
	Users map[string]*UserMetrics `json:"users"`
}

// AccumulatedUserCred tracks accumulated cred by day.
type AccumulatedUserCred map[string]Coin

// Metrics tracked.
type Metrics struct {
	// Interactions
	TotalClaims               uint64 `json:"total_claims"`
	TotalArguments            uint64 `json:"total_arguments"`
	TotalReceivedEndorsements uint64 `json:"total_received_endorsements"`
	TotalGivenEndorsements    uint64 `json:"total_given_endorsments"`

	// StakeBased Metrics
	TotalAmountStaked  Coin `json:"total_amount_staked"`
	StakeEarned        Coin `json:"stake_earned"`
	StakeLost          Coin `json:"stake_lost"`
	TotalAmountAtStake Coin `json:"total_amount_at_stake"`
	InterestEarned     Coin `json:"interest_earned"`
}

// CategoryMetrics summary of metrics by category.
type CategoryMetrics struct {
	CategoryID   uint64   `json:"category_id"`
	CategoryName string   `json:"category_name"`
	CredBalance  Coin     `json:"cred_balance"`
	Metrics      *Metrics `json:"metrics"`
}

// UserMetrics a summary of different metrics per user
type UserMetrics struct {
	UserName string `json:"username"`
	Balance  Coin   `json:"balance"`

	// ByCategoryID
	CategoryMetrics map[int64]*CategoryMetrics `json:"category_metrics"`

	// Tracked by day
	CredEarned map[string]AccumulatedUserCred
}
