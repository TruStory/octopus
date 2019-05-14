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
	Amount BigInt `json:"amount"`
}

// Minus subtracts amounts of two coins with same denom. If the coins differ in denom
// then it panics.
func (coin Coin) Minus(coinB Coin) Coin {
	res := Coin{coin.Denom, coin.Amount.Sub(coinB.Amount)}
	if res.IsNegative() {
		panic("negative count amount")
	}

	return res
}

// Metrics represents metrics for the platform.
type Metrics struct {
	Users map[string]*UserMetrics `json:"users"`
}

// UserMetrics a summary of different metrics per user
type UserMetrics struct {
	TotalClaims            int64 `json:"total_claims"`
	TotalArguments         int64 `json:"total_arguments"`
	TotalGivenEndorsements int64 `json:"total_given_endorsments"`
	// Tracked by day
	// CredEarned         map[string]AccumulatedUserCred
	InterestEarned     Coin `json:"intereset_earned"`
	StakeLost          Coin `json:"stake_lost"`
	StakeEarned        Coin `json:"stake_earned"`
	TotalAmountAtStake Coin `json:"total_amount_at_stake"`
	TotalAmountStaked  Coin `json:"total_amount_staked"`
}
