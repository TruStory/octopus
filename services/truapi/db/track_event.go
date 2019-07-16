package db

import "time"

type TrackEventMeta struct {
	ClaimID     *int64  `json:"claimId,omitempty"`
	CommunityID *string `json:"communityId,omitempty"`
}

// TrackEvent represents analytics events track from the app.
type TrackEvent struct {
	ID               int64
	Address          string
	TwitterProfileID int64
	Event            string
	Meta             TrackEventMeta
	SessionID        string
	IsAnonymous      bool
	Timestamps
}

// UserOpenedClaimsSummary represents a metric of opened claims by user.
type UserOpenedClaimsSummary struct {
	Address            string `json:"string"`
	CommunityID        string `json:"community_id"`
	OpenedClaims       int64  `json:"opened_claims"`
	UniqueOpenedClaims int64  `json:"unique_opened_claims"`
}

func (c *Client) OpenedClaimsSummary(date time.Time) ([]UserOpenedClaimsSummary, error) {
	openedClaimsSummary := make([]UserOpenedClaimsSummary, 0)
	query := `
		SELECT address, meta->>'communityId' community_id,  
			COUNT(meta->'claimId') opened_claims ,
			COUNT(DISTINCT meta->'claimId') unique_opened_claims
		FROM track_events 
		WHERE
			created_at < ?
			AND address is not null 
			AND meta->'communityId' is not null 
			AND meta->'claimId' is not null 
		GROUP BY address, meta->>'communityId';
	`
	_, err := c.Query(&openedClaimsSummary, query, date)
	if err != nil {
		return nil, err
	}
	return openedClaimsSummary, nil
}
