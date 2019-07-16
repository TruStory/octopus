package db

// ClaimImageURL represents a claim source url preview thumbnail in the database
type ClaimImageURL struct {
	ClaimID uint64 `json:"claim_id"`
	URL     string `json:"claim_image_url"`
	Timestamps
}

// ClaimImageURL returns the source url preview thumbnail
func (c *Client) ClaimImageURL(claimID uint64) (string, error) {
	claimImageURL := new(ClaimImageURL)
	err := c.Model(claimImageURL).Where("claim_id = ?", claimID).Limit(1).Select()
	if err != nil {
		return "", err
	}

	return claimImageURL.URL, nil
}

// AddClaimImageURL adds a new claim source url preview thumbnail to the claim_source_url_preview table
func (c *Client) AddClaimImageURL(claimImageURL *ClaimImageURL) error {
	_, err := c.Model(claimImageURL).OnConflict("(claim_id) DO UPDATE").
		Set("claim_image_url = ?", claimImageURL.URL).
		Insert()
	return err
}
