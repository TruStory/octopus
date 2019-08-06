package db

import "time"

type TrackEventMeta struct {
	ClaimID     *int64  `json:"claimId,omitempty"`
	CommunityID *string `json:"communityId,omitempty"`
	ArgumentID  *int64  `json:"argumentId,omitempty"`
}

// TrackEvent represents analytics events track from the app.
type TrackEvent struct {
	ID          int64
	Address     string
	Event       string
	Meta        TrackEventMeta
	SessionID   string
	IsAnonymous bool
	Timestamps
}

// UserOpenedClaimsSummary represents a metric of opened claims by user.
type UserOpenedClaimsSummary struct {
	Address            string `json:"string"`
	CommunityID        string `json:"community_id"`
	OpenedClaims       int64  `json:"opened_claims"`
	UniqueOpenedClaims int64  `json:"unique_opened_claims"`
}

// UserOpenedArgumentsSummary represents a metric of opened claims by user.
type UserOpenedArgumentsSummary struct {
	Address               string `json:"string"`
	CommunityID           string `json:"community_id"`
	OpenedArguments       int64  `json:"opened_arguments"`
	UniqueOpenedArguments int64  `json:"unique_opened_arguments"`
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
			AND event = 'claim_opened'
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

func (c *Client) OpenedArgumentsSummary(date time.Time) ([]UserOpenedArgumentsSummary, error) {
	openedArgumentsSummary := make([]UserOpenedArgumentsSummary, 0)
	query := `
		SELECT address, meta->>'communityId' community_id,  
			COUNT(meta->'argumentId') opened_arguments ,
			COUNT(DISTINCT meta->'argumentId') unique_opened_arguments
		FROM track_events 
		WHERE
			created_at < ?
			AND event = 'argument_opened'
			AND address is not null 
			AND meta->'communityId' is not null 
			AND meta->'argumentId' is not null 
		GROUP BY address, meta->>'communityId';
	`
	_, err := c.Query(&openedArgumentsSummary, query, date)
	if err != nil {
		return nil, err
	}
	return openedArgumentsSummary, nil
}
