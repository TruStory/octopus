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

// Metrics tracked.
type Metrics struct {
	// Interactions
	TotalClaims               uint64 `json:"total_claims"`
	TotalArguments            uint64 `json:"total_arguments"`
	TotalBackings             uint64 `json:"total_backings"`
	TotalChallenges           uint64 `json:"total_challenges"`
	TotalReceivedEndorsements uint64 `json:"total_received_endorsements"`
	TotalGivenEndorsements    uint64 `json:"total_given_endorsments"`

	// StakeBased Metrics
	TotalAmountBacked     Coin `json:"total_Amount_backed"`
	TotalAmountChallenged Coin `json:"total_Amount_challenged"`
	TotalAmountStaked     Coin `json:"total_amount_staked"`
	StakeEarned           Coin `json:"stake_earned"`
	StakeLost             Coin `json:"stake_lost"`
	TotalAmountAtStake    Coin `json:"total_amount_at_stake"`
	InterestEarned        Coin `json:"interest_earned"`
}

// CategoryMetrics summary of metrics by category.
type CategoryMetrics struct {
	CategoryID   uint64   `json:"category_id"`
	CategoryName string   `json:"category_name"`
	CredEarned   Coin     `json:"cred_earned"`
	Metrics      *Metrics `json:"metrics"`
}

// UserMetrics a summary of different metrics per user
type UserMetrics struct {
	UserName       string `json:"username"`
	Balance        Coin   `json:"balance"`
	RunningBalance Coin   `json:"running_balance"`

	// ByCategoryID
	CategoryMetrics map[int64]*CategoryMetrics `json:"category_metrics"`
}

// UsersByAddressesQuery fetches a user by the given address
const UsersByAddressesQuery = `
  query User($addresses: [String]) {
		users(addresses: $addresses) {
			id
			coins {
				amount
				denom
			}
		}
	}
`

// User represents the user from the chain
type User struct {
	ID    string `json:"id"`
	Coins []Coin `json:"coins"`
}

// UsersByAddressesResponse defines the JSON response
type UsersByAddressesResponse struct {
	Users []User `json:"users"`
}
