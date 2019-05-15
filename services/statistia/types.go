package main

import (
	"fmt"
	"math/big"
)

// Int wraps integer with 256 bit range bound
// Checks overflow, underflow and division by zero
// Exists in range from -(2^255-1) to 2^255-1
type Int struct {
	i *big.Int
}

type BigInt struct {
	big.Int
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *BigInt) UnmarshalJSON(p string) error {
	if string(p) == "null" {
		return nil
	}
	var z big.Int
	_, ok := z.SetString(p, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", p)
	}
	b.Int = z
	return nil
}

// Coin hold some amount of one currency.
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// Minus subtracts amounts of two coins with same denom. If the coins differ in denom
// then it panics.
// func (coin Coin) Minus(coinB Coin) Coin {
// 	res := Coin{coin.Denom, coin.Amount.Sub(coinB.Amount)}
// 	if res.IsNegative() {
// 		panic("negative count amount")
// 	}

// 	return res
// }

// MetricsSummary represents metrics for the platform.
type MetricsSummary struct {
	Users map[string]*UserMetrics `json:"users"`
}

// AccumulatedUserCred tracks accumulated cred by day.
type AccumulatedUserCred map[string]Coin

// Metrics tracked.
type Metrics struct {
	// Interactions
	TotalClaims               int64 `json:"total_claims"`
	TotalArguments            int64 `json:"total_arguments"`
	TotalReceivedEndorsements int64 `json:"total_received_endorsements"`
	TotalGivenEndorsements    int64 `json:"total_given_endorsments"`

	// StakeBased Metrics
	TotalAmountStaked  Coin `json:"total_amount_staked"`
	StakeEarned        Coin `json:"stake_earned"`
	StakeLost          Coin `json:"stake_lost"`
	TotalAmountAtStake Coin `json:"total_amount_at_stake"`
	InterestEarned     Coin `json:"interest_earned"`
}

// CategoryMetrics summary of metrics by category.
type CategoryMetrics struct {
	CategoryID   int64    `json:"category_id"`
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
