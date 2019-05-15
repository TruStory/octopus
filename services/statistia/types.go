package main

import (
	"math/big"
)

// Coin hold some amount of one currency.
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// Minus substracts another coin from the given coin
func (given *Coin) Minus(another Coin) Coin {
	if given.Denom != another.Denom {
		panic("only same denom coins can be subtracted")
	}

	givenBI, _ := new(big.Int).SetString(given.Amount, 10)
	anotherBI, _ := new(big.Int).SetString(another.Amount, 10)
	diffBI := new(big.Int).Sub(givenBI, anotherBI)

	return Coin{
		Denom:  given.Denom,
		Amount: diffBI.String(),
	}
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
