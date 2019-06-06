package db

// ClaimOfTheDayID represents a claim of the day ID in the DB
type ClaimOfTheDayID struct {
	CommunitySlug string `json:"community_slug"`
	ClaimID       int64  `json:"claim_id"`
}

// ClaimOfTheDayIDByCommunitySlug returns currently featured claim in each community
func (c *Client) ClaimOfTheDayIDByCommunitySlug(communitySlug string) (int64, error) {
	claimOfTheDayID := new(ClaimOfTheDayID)
	err := c.Model(claimOfTheDayID).Where("community_slug = ?", communitySlug).Limit(1).Select()
	if err != nil {
		return -1, err
	}

	return claimOfTheDayID.ClaimID, nil
}

// AddClaimOfTheDayID adds a new claim of the day id to the claim_of_the_day_ids table
func (c *Client) AddClaimOfTheDayID(claimOfTheDay *ClaimOfTheDayID) error {
	_, err := c.Model(claimOfTheDay).OnConflict("(community_slug) DO UPDATE").
		Set("claim_id = ?", claimOfTheDay.ClaimID).
		Insert()
	return err
}

// DeleteClaimOfTheDayID deletes a claim of the day id from the claim_of_the_day_ids table
func (c *Client) DeleteClaimOfTheDayID(communitySlug string) error {
	t := &ClaimOfTheDayID{}
	_, err := c.Model(t).Where("community_slug = ?", communitySlug).Delete()
	return err
}
