package db

import "time"

type TrackEventMeta struct {
	ClaimID    *int64 `json:"claimId,omitempty"`
	CategoryID *int64 `json:"categoryId,omitempty"`
}

// TrackEvent represents analytics events track from the app.
type TrackEvent struct {
	ID               int64
	Address          string
	TwitterProfileID int64
	Event            string
	Meta             TrackEventMeta
	Timestamps
}

// UserOpenedClaimsSummary represents a metric of opened claims by user.
type UserOpenedClaimsSummary struct {
	Address      string `json:"string"`
	CategoryID   int64  `json:"category_id"`
	OpenedClaims int64  `json:"opened_claims"`
}

func (c *Client) OpenedClaimsSummary(date time.Time) ([]UserOpenedClaimsSummary, error) {
	openedClaimsSummary := make([]UserOpenedClaimsSummary, 0)
	query := `
		SELECT address, meta->'categoryId' category_id,  COUNT(DISTINCT meta->'claimId') opened_claims 
		FROM track_events 
		WHERE
			created_at < ?
			AND address is not null 
			AND meta->'categoryId' is not null 
			AND meta->'claimId' is not null 
		GROUP BY address, meta->'categoryId';
	`
	_, err := c.Query(&openedClaimsSummary, query, date)
	if err != nil {
		return nil, err
	}
	return openedClaimsSummary, nil
}
