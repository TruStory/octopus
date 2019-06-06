package db

// ClaimOfTheDayID represents a claim of the day ID in the DB
type ClaimOfTheDayID struct {
	CommunitySlug string `json:"community_slug"`
	ClaimID int64 `json:"claim_id"`
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
