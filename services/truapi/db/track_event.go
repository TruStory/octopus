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

// ClaimViewsStats represents stats of opened claims and arguments related to the claim.
type ClaimViewsStats struct {
	ClaimID                  int64
	UserViews                int64
	UniqueUserViews          int64
	AnonViews                int64
	UniqueAnonViews          int64
	UserArgumentsViews       int64
	UniqueUserArgumentsViews int64
	AnonArgumentsViews       int64
	UniqueAnonArgumentsViews int64
}

// ClaimRepliesStats represets stats about replies.
type ClaimRepliesStats struct {
	ClaimID int64
	Replies int64
}

func (c *Client) OpenedClaimsSummary(date time.Time) ([]UserOpenedClaimsSummary, error) {
	openedClaimsSummary := make([]UserOpenedClaimsSummary, 0)
	query := `
	SELECT m.address, m.community_id, sum(m.opened_claims) opened_claims, sum(m.unique_opened_claims) unique_opened_claims
	FROM (
	SELECT date(created_at) created_at, address, meta->>'communityId' community_id,  
		COUNT(meta->'claimId') opened_claims ,
		COUNT(DISTINCT meta->'claimId') unique_opened_claims
	FROM track_events 
	WHERE
		created_at < ?
		AND event = 'claim_opened'
		AND address is not null 
		AND address != '' 
		AND meta->'communityId' is not null 
		AND meta->'claimId' is not null 
	GROUP BY DATE(created_at), address, meta->>'communityId') AS m
	GROUP  by m.address, m.community_id
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
	SELECT
	m.address,
	m.community_id,
	sum(m.opened_arguments) opened_arguments,
	sum(m.unique_opened_arguments) unique_opened_arguments
    FROM (
        	SELECT
                date(created_at) created_at,
                address,
                meta ->> 'communityId' community_id,
                COUNT(meta -> 'argumentId') opened_arguments,
                COUNT(DISTINCT meta -> 'argumentId') unique_opened_arguments
        	FROM
			    track_events
        	WHERE
    	        created_at < ?
      	        AND event = 'argument_opened'
				AND address IS NOT NULL
				AND address != '' 
      	        AND meta -> 'communityId' IS NOT NULL
              	AND meta -> 'argumentId' IS NOT NULL
        	GROUP BY
  	        	DATE(created_at),
  	        	address,
   	        	meta ->> 'communityId') AS m
    GROUP BY
       	m.address,
       	m.community_id
	`
	_, err := c.Query(&openedArgumentsSummary, query, date)
	if err != nil {
		return nil, err
	}
	return openedArgumentsSummary, nil
}

func (c *Client) ClaimViewsStats(date time.Time) ([]ClaimViewsStats, error) {
	claimViewsStats := make([]ClaimViewsStats, 0)
	query := `
			SELECT
				claim_id,
				sum(m.user_views) user_views,
				sum(m.unique_user_views) unique_user_views,
				sum(m.anon_views) anon_views,
				sum(m.unique_anon_views) unique_anon_views,
				sum(m.user_arguments_views) user_arguments_views,
				sum(m.unique_user_arguments_views) unique_user_arguments_views,
				sum(m.anon_arguments_views) anon_arguments_views,
				sum(m.unique_anon_arguments_views) unique_anon_arguments_views
			FROM (
				SELECT
					DATE(created_at),
					meta ->> 'claimId' claim_id,
					count(address) FILTER (WHERE event = 'claim_opened') user_views,
					count(DISTINCT address) FILTER (WHERE event = 'claim_opened') unique_user_views,
					count(session_id) FILTER (WHERE event = 'claim_opened') anon_views,
					count(DISTINCT session_id) FILTER (WHERE event = 'claim_opened') unique_anon_views,
					count(address) FILTER (WHERE event = 'argument_opened') user_arguments_views,
					count(DISTINCT address) FILTER (WHERE event = 'argument_opened') unique_user_arguments_views,
					count(session_id) FILTER (WHERE event = 'argument_opened') anon_arguments_views,
					count(DISTINCT session_id) FILTER (WHERE event = 'argument_opened') unique_anon_arguments_views 
					FROM track_events WHERE event in('claim_opened', 'argument_opened')
					AND
					created_at < ?
				GROUP BY
					DATE(created_at),
					meta ->> 'claimId') AS m
			GROUP BY
				m.claim_id;
				`

	_, err := c.Query(&claimViewsStats, query, date)
	if err != nil {
		return nil, err
	}
	return claimViewsStats, nil
}

func (c *Client) ClaimRepliesStats(date time.Time) ([]ClaimRepliesStats, error) {
	claimRepliesStats := make([]ClaimRepliesStats, 0)
	query := `
				SELECT
					claim_id,
					count(claim_id) replies
				FROM
					comments
				WHERE
					created_at < ?
				GROUP BY
					claim_id
				`

	_, err := c.Query(&claimRepliesStats, query, date)
	if err != nil {
		return nil, err
	}
	return claimRepliesStats, nil
}
