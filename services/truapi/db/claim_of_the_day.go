package db

// ClaimOfTheDayID represents a claim of the day ID in the DB
type ClaimOfTheDayID struct {
	CommunityID string `json:"community_id"`
	ClaimID     int64  `json:"claim_id"`
}

// ClaimOfTheDayIDByCommunityID returns currently featured claim in each community
func (c *Client) ClaimOfTheDayIDByCommunityID(communityID string) (int64, error) {
	// personal home feed and all claims (discover) feed should return same Claim of the Day
	if communityID == "home" {
		communityID = "all"
	}
	claimOfTheDayID := new(ClaimOfTheDayID)
	err := c.Model(claimOfTheDayID).Where("community_id = ?", communityID).Limit(1).Select()
	if err != nil {
		return -1, err
	}

	return claimOfTheDayID.ClaimID, nil
}

// AddClaimOfTheDayID adds a new claim of the day id to the claim_of_the_day_ids table
func (c *Client) AddClaimOfTheDayID(claimOfTheDay *ClaimOfTheDayID) error {
	_, err := c.Model(claimOfTheDay).OnConflict("(community_id) DO UPDATE").
		Set("claim_id = ?", claimOfTheDay.ClaimID).
		Insert()
	return err
}

// DeleteClaimOfTheDayID deletes a claim of the day id from the claim_of_the_day_ids table
func (c *Client) DeleteClaimOfTheDayID(communityID string) error {
	t := &ClaimOfTheDayID{}
	_, err := c.Model(t).Where("community_id = ?", communityID).Delete()
	return err
}
