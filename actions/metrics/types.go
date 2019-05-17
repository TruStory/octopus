package main

import (
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MetricsSummary represents metrics for the platform.
type MetricsSummary struct {
	Users map[string]*UserMetrics `json:"users"`
}

// Metrics tracked.
type Metrics struct {
	// Interactions
	TotalClaims               int64 `json:"total_claims"`
	TotalArguments            int64 `json:"total_arguments"`
	TotalEndorsementsReceived int64 `json:"total_endorsements_received"`
	TotalEndorsementsGiven    int64 `json:"total_endorsements_given"`

	// StakeBased Metrics
	TotalAmountStaked     sdk.Coin `json:"total_amount_staked"`
	TotalAmountBacked     sdk.Coin `json:"total_amount_backed"`
	TotalAmountChallenged sdk.Coin `json:"total_amount_challenged"`
	StakeEarned           sdk.Coin `json:"stake_earned"`
	StakeLost             sdk.Coin `json:"stake_lost"`
	TotalAmountAtStake    sdk.Coin `json:"total_amount_at_stake"`
	InterestEarned        sdk.Coin `json:"interest_earned"`
}

// CategoryMetrics summary of metrics by category.
type CategoryMetrics struct {
	CategoryID   int64    `json:"category_id"`
	CategoryName string   `json:"category_name"`
	CredEarned   sdk.Coin `json:"cred_earned"`
	Metrics      *Metrics `json:"metrics"`
}

// UserMetrics a summary of different metrics per user
type UserMetrics struct {
	UserName string   `json:"username"`
	Balance  sdk.Coin `json:"balance"`

	// ByCategoryID
	CategoryMetrics map[int64]*CategoryMetrics `json:"category_metrics"`
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}
