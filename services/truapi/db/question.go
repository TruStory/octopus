package db

import "time"

// Question represents a question in the DB
type Question struct {
	Timestamps
	ID      int64  `json:"id"`
	ClaimID int64  `json:"claim_id"`
	Body    string `json:"body"`
	Creator string `json:"creator"`
}

// QuestionsByClaimID finds questions by claim id
func (c *Client) QuestionsByClaimID(claimID uint64) ([]Question, error) {
	questions := make([]Question, 0)
	err := c.Model(&questions).Where("claim_id = ?", claimID).Where("deleted_at is NULL").Select()
	if err != nil {
		return nil, err
	}

	return questions, nil
}

// AddQuestion adds a new question to the questions table
func (c *Client) AddQuestion(question *Question) error {
	err := c.Add(question)
	if err != nil {
		return err
	}

	return nil
}

// QuestionByID finds a question by id
func (c *Client) QuestionByID(ID int64) (*Question, error) {
	question := new(Question)
	err := c.Model(question).Where("id = ?", ID).First()
	if err != nil {
		return nil, err
	}

	return question, nil
}

// DeleteQuestion deletes a question by id
func (c *Client) DeleteQuestion(ID int64) error {
	question := new(Question)

	_, err := c.Model(question).
		Where("id = ?", ID).
		Set("deleted_at = ?", time.Now()).
		Update()

	if err != nil {
		return err
	}

	return nil
}
