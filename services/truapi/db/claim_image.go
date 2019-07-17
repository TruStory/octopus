package db

// ClaimImage represents a claim image in the database
type ClaimImage struct {
	ClaimID       uint64 `json:"claim_id"`
	ClaimImageURL string `json:"claim_image_url"`
	Timestamps
}

// ClaimImageURL returns the claim image url associated with as given claimID
func (c *Client) ClaimImageURL(claimID uint64) (string, error) {
	claimImageURL := new(ClaimImage)
	err := c.Model(claimImageURL).Where("claim_id = ?", claimID).Limit(1).Select()
	if err != nil {
		return "", err
	}

	return claimImageURL.ClaimImageURL, nil
}

// AddClaimImage adds a new claim image to the claim_images table
func (c *Client) AddClaimImage(claimImageURL *ClaimImage) error {
	_, err := c.Model(claimImageURL).OnConflict("(claim_id) DO UPDATE").
		Set("claim_image_url = ?", claimImageURL.ClaimImageURL).
		Insert()
	return err
}
