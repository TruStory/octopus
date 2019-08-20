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
