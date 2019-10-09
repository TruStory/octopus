package db

import (
	"fmt"
	"time"

	"github.com/go-pg/pg"
)

// LeaderboardProcessedDate represents the status of a processed date.
type LeaderboardProcessedDate struct {
	ID          int64
	Date        time.Time
	MetricsTime time.Time
	FullDay     bool
	Timestamps
}

type LeaderboardTopUser struct {
	Address        string
	Earned         int64
	AgreesReceived int64
	AgreesGiven    int64
}

type LeaderboardUserMetric struct {
	ID             int64
	Date           time.Time
	Address        string
	CommunityID    string
	Earned         int64 `sql:"type:,notnull"`
	AgreesReceived int64 `sql:"type:,notnull"`
	AgreesGiven    int64 `sql:"type:,notnull"`
	Timestamps
}

func (c *Client) LastLeaderboardProcessedDate() (*LeaderboardProcessedDate, error) {
	lastProcessedDate := &LeaderboardProcessedDate{}
	err := c.Model(lastProcessedDate).Where("full_day is true").Order("date DESC").Limit(1).Select()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		fmt.Println("error query", err)
		return nil, err
	}
	return lastProcessedDate, nil
}

func (c *Client) UpsertLeaderboardMetric(tx *pg.Tx, metric *LeaderboardUserMetric) error {
	_, err := tx.Model(metric).
		OnConflict("ON CONSTRAINT leaderboard_metrics_no_duplicate_date DO UPDATE").
		Set(`
                earned = EXCLUDED.earned,
                agrees_received = EXCLUDED.agrees_received,
                agrees_given = EXCLUDED.agrees_given,
				updated_at = NOW()
				`).
		Insert()
	return err
}

func (c *Client) UpsertLeaderboardProcessedDate(tx *pg.Tx, processedDate *LeaderboardProcessedDate) error {
	_, err := tx.Model(processedDate).
		OnConflict("ON CONSTRAINT leaderboard_processed_no_duplicate DO UPDATE").
		Set(`
                metrics_time = EXCLUDED.metrics_time,
                full_day = EXCLUDED.full_day,
				updated_at = NOW()
				`).
		Insert()
	return err
}
func (c *Client) FeedLeaderboardInTransaction(fn func(*pg.Tx) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rolled back: %s", r)
		}
	}()
	err = c.RunInTransaction(fn)
	return err
}

func (c *Client) Leaderboard(since time.Time, sortBy string, limit int, excludedCommunities []string) ([]LeaderboardTopUser, error) {
	topUsers := make([]LeaderboardTopUser, 0)
	q := c.Model((*LeaderboardUserMetric)(nil)).
		Column("address").
		ColumnExpr("SUM(earned) earned").
		ColumnExpr("SUM(agrees_received) agrees_received").
		ColumnExpr("SUM(agrees_given) agrees_given")
	if len(excludedCommunities) > 0 {
		q = q.Where("community_id  not in(?)", pg.In(excludedCommunities))
	}
	if !since.IsZero() {
		q = q.Where("date >= ?", since)
	}
	q = q.Group("address").
		OrderExpr(fmt.Sprintf("SUM(%s) DESC", sortBy)).
		Limit(limit)
	err := q.Select(&topUsers)
	if err != nil {
		return topUsers, err
	}
	return topUsers, nil
}

// UserLeaderboardProfile fetches user profile by username
func (c *Client) UserLeaderboardProfile(address string) (*LeaderboardTopUser, error) {
	userProfile := new(LeaderboardTopUser)

	metric := &LeaderboardUserMetric{}
	err := c.Model(metric).Where("address = ?", address).First()

	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	userProfile = &LeaderboardTopUser{
		Address:        metric.Address,
		Earned:         metric.Earned,
		AgreesGiven:    metric.AgreesGiven,
		AgreesReceived: metric.AgreesReceived,
	}

	return userProfile, nil
}
