package db

// ClaimImage represents a claim image in the database
type ClaimImage struct {
	ClaimID       uint64 `json:"claim_id"`
	ClaimImageURL string `json:"claim_image_url"`
	ClaimVideoURL string `json:"claim_video_url"`
	Timestamps
}

// ClaimImageURL returns the claim image url associated with as given claimID
func (c *Client) ClaimImageURL(claimID uint64) (string, error) {
	claimImage := new(ClaimImage)
	err := c.Model(claimImage).Where("claim_id = ?", claimID).Limit(1).Select()
	if err != nil {
		return "", err
	}

	return claimImage.ClaimImageURL, nil
}

// ClaimVideoURL returns the claim video url associated with as given claimID
func (c *Client) ClaimVideoURL(claimID uint64) (string, error) {
	claimImage := new(ClaimImage)
	err := c.Model(claimImage).Where("claim_id = ?", claimID).Limit(1).Select()
	if err != nil {
		return "", err
	}

	return claimImage.ClaimVideoURL, nil
}

// AddClaimImage adds a new claim image to the claim_images table
func (c *Client) AddClaimImage(claimImageURL *ClaimImage) error {
	_, err := c.Model(claimImageURL).OnConflict("(claim_id) DO UPDATE").
		Set("claim_image_url = ?", claimImageURL.ClaimImageURL).
		Insert()
	return err
}
