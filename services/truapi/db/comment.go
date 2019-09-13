package db

import "time"

// Comment represents a comment in the DB
type Comment struct {
	Timestamps
	ID          int64  `json:"id"`
	ParentID    int64  `json:"parent_id"`
	ClaimID     int64  `json:"claim_id"`
	ArgumentID  int64  `json:"argument_id"`
	ElementID   int64  `json:"element_id"`
	Body        string `json:"body"`
	Creator     string `json:"creator"`
	CommunityID string `json:"community_id"`
}

// ClaimLevelComments returns claim level comments, excluding argument level comments
func (c *Client) ClaimLevelComments(claimID uint64) ([]Comment, error) {
	comments := make([]Comment, 0)
	err := c.Model(&comments).Where("claim_id = ?", claimID).
		Where("argument_id is NULL").
		Where("element_id is NULL").
		Order("id ASC").Select()
	if err != nil {
		return nil, err
	}
	transformedComments, subErr := c.replaceAddressesWithProfileURLsInComments(comments)
	if subErr != nil {
		return nil, err
	}
	return transformedComments, nil
}

// ArgumentLevelComments returns argument level comments
func (c *Client) ArgumentLevelComments(argumentID uint64, elementID uint64) ([]Comment, error) {
	comments := make([]Comment, 0)
	err := c.Model(&comments).
		Where("argument_id = ?", argumentID).
		Where("element_id = ?", elementID).
		Order("id ASC").Select()
	if err != nil {
		return nil, err
	}
	transformedComments, subErr := c.replaceAddressesWithProfileURLsInComments(comments)
	if subErr != nil {
		return nil, err
	}
	return transformedComments, nil
}

// CommentsByClaimID finds all comments pertaining to a specific claim, both claim level and argument level comments
// useful when determining all participants on a claim
func (c *Client) CommentsByClaimID(claimID uint64) ([]Comment, error) {
	comments := make([]Comment, 0)
	err := c.Model(&comments).Where("claim_id = ?", claimID).Order("id ASC").Select()
	if err != nil {
		return nil, err
	}
	transformedComments, subErr := c.replaceAddressesWithProfileURLsInComments(comments)
	if subErr != nil {
		return nil, err
	}
	return transformedComments, nil
}

// AddComment adds a new comment to the comments table
func (c *Client) AddComment(comment *Comment) error {
	transformedBody, err := c.replaceUsernamesWithAddress(comment.Body)
	if err != nil {
		return err
	}
	comment.Body = transformedBody
	err = c.Add(comment)
	if err != nil {
		return err
	}

	return nil
}

// ClaimLevelCommentsParticipants gets the list of users participating on a claim thread.
func (c *Client) ClaimLevelCommentsParticipants(claimID int64) ([]string, error) {
	comments := make([]Comment, 0)
	addresses := make([]string, 0)
	err := c.Model(&comments).ColumnExpr("DISTINCT creator").Where("claim_id = ?", claimID).Select()
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		addresses = append(addresses, c.Creator)
	}
	return addresses, nil
}

// ArgumentLevelCommentsParticipants gets the list of users participating on a argument comment thread.
func (c *Client) ArgumentLevelCommentsParticipants(argumentID int64, elementID int64) ([]string, error) {
	comments := make([]Comment, 0)
	addresses := make([]string, 0)
	err := c.Model(&comments).ColumnExpr("DISTINCT creator").
		Where("argument_id = ?", argumentID).
		Where("element_id = ?", elementID).Select()
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		addresses = append(addresses, c.Creator)
	}
	return addresses, nil
}

// CommentByID returns the comment for specific pk.
func (c *Client) CommentByID(id int64) (*Comment, error) {
	comment := new(Comment)
	err := c.Model(comment).Where("id = ?", id).Select()
	if err != nil {
		return comment, err
	}
	return comment, nil
}

func (c *Client) replaceAddressesWithProfileURLsInComments(comments []Comment) ([]Comment, error) {
	transformedComments := make([]Comment, 0)
	for _, comment := range comments {
		transformedComment := comment
		transformedBody, err := c.replaceAddressesWithProfileURLs(comment.Body)
		if err != nil {
			return transformedComments, err
		}
		transformedComment.Body = transformedBody
		transformedComments = append(transformedComments, transformedComment)
	}
	return transformedComments, nil
}

// UserRepliesStats represets stats about user comments.
type UserRepliesStats struct {
	Address     string
	CommunityID string
	Replies     int64
}

func (c *Client) UserRepliesStats(date time.Time) ([]UserRepliesStats, error) {
	userRepliesStats := make([]UserRepliesStats, 0)
	query := `
				SELECT
					creator address,
					community_id,
					count(id) replies
				FROM
					comments
				WHERE
					created_at < ?
				GROUP BY
					creator,
					community_id
				`

	_, err := c.Query(&userRepliesStats, query, date)
	if err != nil {
		return nil, err
	}
	return userRepliesStats, nil
}
