package db

// Highlight represents a highlight in a paragraph
type Highlight struct {
	Timestamps

	ID                int64  `json:"id"`
	HighlightableType string `json:"highlightable_type"`
	HighlightableID   int64  `json:"highlightable_id"`
	Text              string `json:"text"`
	ImageURL          string `json:"image_url"`
}

// AddImageURLToHighlight adds the url for the cached version to a highlight
func (c *Client) AddImageURLToHighlight(id int64, url string) error {
	var highlight Highlight
	_, err := c.Model(&highlight).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Set("image_url = ?", url).
		Update()

	if err != nil {
		return err
	}

	return nil
}
