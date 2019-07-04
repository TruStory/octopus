package db

import (
	"time"

	"github.com/go-pg/pg"
)

// UserMetric is the db model to interact with user metrics
type UserMetric struct {
	tableName          struct{}  `sql:"user_metrics_v2"`
	Address            string    `json:"address"`
	AsOnDate           time.Time `json:"as_on_date"`
	CommunityID        string    `json:"community_id"`
	TotalAmountStaked  uint64    `json:"total_amount_staked" sql:"type:,notnull"`
	StakeEarned        uint64    `json:"stake_earned" sql:"type:,notnull"`
	StakeLost          uint64    `json:"stake_lost" sql:"type:,notnull"`
	TotalAmountAtStake uint64    `json:"total_amount_at_stake" sql:"type:,notnull"`
	AvailableStake     uint64    `json:"available_stake" sql:"type:,notnull"`
}

// AggregateUserMetricsByAddressBetweenDates gets and aggregates the user metrics for a given address on a given date
func (c *Client) AggregateUserMetricsByAddressBetweenDates(address string, from string, to string) ([]UserMetric, error) {
	userMetrics := make([]UserMetric, 0)
	err := c.Model(&userMetrics).
		Column("as_on_date", "category_id").
		ColumnExpr(`
			sum(total_amount_at_stake) as total_amount_at_stake,
			sum(stake_earned) as stake_earned,
			sum(stake_lost) as stake_lost,
			sum(total_amount_staked) as total_amount_staked,
			sum(available_stake) as available_stake
		`).
		Where("address = ?", address).
		Where("as_on_date >= ?", from).
		Where("as_on_date <= ?", to).
		Group("as_on_date").
		Group("community_id").
		Order("as_on_date").
		Order("community_id").
		Select()
	if err != nil {
		return nil, err
	}

	return userMetrics, nil
}

// UpsertDailyUserMetricInTx inserts or updates the daily metric for the user in a transaction
func UpsertDailyUserMetricInTx(tx *pg.Tx, metric UserMetric) error {
	_, err := tx.Model(&metric).
		OnConflict("ON CONSTRAINT no_duplicate_user_metric DO UPDATE").
		Set(upsertStatement()).
		Insert()

	return err
}

// AreUserMetricsEmpty returns whether the user metrics table is empty or not
func (c *Client) AreUserMetricsEmpty() (bool, error) {
	var userMetric UserMetric
	count, err := c.Model(&userMetric).Count()
	if err != nil {
		return false, err
	}

	if count == 0 {
		return true, nil
	}

	return false, nil
}

func upsertStatement() string {
	return `
		address = EXCLUDED.address,
		total_amount_at_stake = EXCLUDED.total_amount_at_stake,
		stake_earned = EXCLUDED.stake_earned,
		stake_lost = EXCLUDED.stake_lost,
		total_amount_staked = EXCLUDED.total_amount_staked,
		available_stake = EXCLUDED.available_stake
	`
}
