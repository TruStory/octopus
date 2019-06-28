package db

// ClaimSourceURLPreview represents a claim source url preview thumbnail in the database
type ClaimSourceURLPreview struct {
	ClaimID          uint64 `json:"claim_id"`
	SourceURLPreview string `json:"source_url_preview"`
	Timestamps
}

// ClaimSourceURLPreview returns the source url preview thumbnail
func (c *Client) ClaimSourceURLPreview(claimID uint64) (string, error) {
	claimSourceURLPreview := new(ClaimSourceURLPreview)
	err := c.Model(claimSourceURLPreview).Where("claim_id = ?", claimID).Limit(1).Select()
	if err != nil {
		return "", err
	}

	return claimSourceURLPreview.SourceURLPreview, nil
}

// AddClaimSourceURLPreview adds a new claim source url preview thumbnail to the claim_source_url_preview table
func (c *Client) AddClaimSourceURLPreview(claimSourceURLPreview *ClaimSourceURLPreview) error {
	_, err := c.Model(claimSourceURLPreview).OnConflict("(claim_id) DO UPDATE").
		Set("source_url_preview = ?", claimSourceURLPreview.SourceURLPreview).
		Insert()
	return err
}
