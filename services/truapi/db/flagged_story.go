package db

import (
	"time"
)

// FlaggedStory represents a flagged story in the DB
type FlaggedStory struct {
	Timestamps
	ID        int64     `json:"id"`
	StoryID   int64     `json:"story_id"`
	Creator   string    `json:"creator"`
	CreatedOn time.Time `json:"created_on"`
}

// FlaggedStoriesIDs returns all flagged stories IDs
func (c *Client) FlaggedStoriesIDs(flagAdmin string, flagLimit int) ([]int64, error) {
	flaggedStoriesIDs := make([]int64, 0)

	// all story ids that have been flagged twice
	subq := c.Model((*FlaggedStory)(nil)).
		Column("story_id").
		Group("story_id").
		Having("COUNT(story_id) >= ?", flagLimit)

	// unique story ids that have been flagged twice OR flagged by the admin
	err := c.Model((*FlaggedStory)(nil)).
		ColumnExpr("DISTINCT story_id").
		Where("story_id IN (?) OR creator = ?", subq, flagAdmin).
		Select(&flaggedStoriesIDs)
	if err != nil {
		return nil, err
	}

	return flaggedStoriesIDs, nil
}

// UpsertFlaggedStory implements `Datastore`.
// Updates an existing `FlaggedStory` or creates a new one.
func (c *Client) UpsertFlaggedStory(flaggedStory *FlaggedStory) error {
	_, err := c.Model(flaggedStory).
		Where("story_id = ?", flaggedStory.StoryID).
		Where("creator = ?", flaggedStory.Creator).
		OnConflict("DO NOTHING").
		SelectOrInsert()

	return err
}
