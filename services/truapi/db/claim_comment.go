package db

// ClaimComment represents a claim comment in the DB
type ClaimComment struct {
	Timestamps
	ID       int64  `json:"id"`
	ParentID int64  `json:"parent_id"`
	ClaimID  int64  `json:"claim_id"`
	Body     string `json:"body"`
	Creator  string `json:"creator"`
}

// ClaimCommentsByClaimID finds claim_comments by claim id
func (c *Client) ClaimCommentsByClaimID(claimID int64) ([]ClaimComment, error) {
	comments := make([]ClaimComment, 0)
	err := c.Model(&comments).Where("claim_id = ?", claimID).Select()
	if err != nil {
		return nil, err
	}
	transformedComments := make([]ClaimComment, 0)
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

// AddClaimComment adds a new comment to the claim_comments table
func (c *Client) AddClaimComment(comment *ClaimComment) error {
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

// ClaimCommentsParticipantsByClaimID gets the list of users participating on a claim comment thread.
func (c *Client) ClaimCommentsParticipantsByClaimID(claimID int64) ([]string, error) {
	comments := make([]ClaimComment, 0)
	addresses := make([]string, 0)
	err := c.Model(&comments).ColumnExpr("DISTINCT creator").Where("argument_id = ?", claimID).Select()
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		addresses = append(addresses, c.Creator)
	}
	return addresses, nil
}

// ClaimCommentByID returns the comment for specific pk.
func (c *Client) ClaimCommentByID(id int64) (*ClaimComment, error) {
	comment := new(ClaimComment)
	err := c.Model(comment).Where("id = ?", id).Select()
	if err != nil {
		return comment, err
	}
	return comment, nil
}
