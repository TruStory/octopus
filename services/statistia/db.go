package main

import (
	"time"

	"github.com/go-pg/pg"
)

// UserMetric is the db model to interact with user metrics
type UserMetric struct {
	Address                   string    `json:"address"`
	AsOnDate                  time.Time `json:"as_on_date"`
	CategoryID                int64     `json:"category_id"`
	TotalClaims               uint64    `json:"total_claims"  sql:"type:,notnull"`
	TotalArguments            uint64    `json:"total_arguments"  sql:"type:,notnull"`
	TotalClaimsBacked         uint64    `json:"total_claims_backed"  sql:"type:,notnull"`
	TotalClaimsChallenged     uint64    `json:"total_claims_challenged"  sql:"type:,notnull"`
	TotalAmountBacked         uint64    `json:"total_amount_backed"  sql:"type:,notnull"`
	TotalAmountChallenged     uint64    `json:"total_amount_challenged"  sql:"type:,notnull"`
	TotalEndorsementsGiven    uint64    `json:"total_endorsements_given"  sql:"type:,notnull"`
	TotalEndorsementsReceived uint64    `json:"total_endorsements_received"  sql:"type:,notnull"`
	StakeEarned               uint64    `json:"stake_earned"  sql:"type:,notnull"`
	StakeLost                 uint64    `json:"stake_lost"  sql:"type:,notnull"`
	StakeBalance              uint64    `json:"stake_balance"  sql:"type:,notnull"`
	InterestEarned            uint64    `json:"interest_earned"  sql:"type:,notnull"`
	TotalAmountAtStake        uint64    `json:"total_amount_at_stake"  sql:"type:,notnull"`
	TotalAmountStaked         uint64    `json:"total_amount_staked"  sql:"type:,notnull"`
	CredEarned                uint64    `json:"cred_earned"  sql:"type:,notnull"`
}

// UpsertDailyUserMetricInTx inserts or updates the daily metric for the user in a transaction
func UpsertDailyUserMetricInTx(tx *pg.Tx, metric UserMetric) error {
	_, err := tx.Model(&metric).
		OnConflict("ON CONSTRAINT no_duplicate_metric DO UPDATE").
		Set(upsertStatement()).
		Insert()

	return err
}

func upsertStatement() string {
	return `
		address = EXCLUDED.address,
		total_claims = EXCLUDED.total_claims,
		total_arguments = EXCLUDED.total_arguments,
		total_claims_backed = EXCLUDED.total_claims_backed,
		total_claims_challenged = EXCLUDED.total_claims_challenged,
		total_amount_backed = EXCLUDED.total_amount_backed,
		total_amount_challenged = EXCLUDED.total_amount_challenged,
		total_endorsements_given = EXCLUDED.total_endorsements_given,
		total_endorsements_received = EXCLUDED.total_endorsements_received,
		stake_earned = EXCLUDED.stake_earned,
		stake_lost = EXCLUDED.stake_lost,
		stake_balance = EXCLUDED.stake_balance,
		interest_earned = EXCLUDED.interest_earned,
		total_amount_at_stake = EXCLUDED.total_amount_at_stake,
		total_amount_staked = EXCLUDED.total_amount_staked,
		cred_earned = EXCLUDED.cred_earned
	`
}
