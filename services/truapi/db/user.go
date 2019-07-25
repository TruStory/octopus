package db

import (
	"time"
)

// User is the user on the TruStory platform
type User struct {
	Timestamps

	ID           uint64    `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Address      string    `json:"address"`
	Password     string    `json:"password"`
	InvitedBy    string    `json:"invited_by"`
	RequestToken string    `json:"request_token"`
	ApprovedAt   time.Time `json:"approved_at"`
	RejectedAt   time.Time `json:"rejected_at"`
}

// ApproveUserByID approves a user to signup (set their password + username)
func (c *Client) ApproveUserByID(id uint64) error {
	user := new(User)
	_, err := c.Model(user).
		Where("id = ?", id).
		Where("signedup_at IS NULL"). // the flag can be updated only until the user hasn't signed up
		Set("approved_at = NOW()").
		Set("rejected_at = NULL").
		Update()

	if err != nil {
		return err
	}

	return nil
}

// RejectUserByID rejects a user from signing up (set their password + username)
func (c *Client) RejectUserByID(id uint64) error {
	user := new(User)
	_, err := c.Model(user).
		Where("id = ?", id).
		Where("signedup_at IS NULL"). // the flag can be updated only until the user hasn't signed up
		Set("rejected_at = ?", time.Now()).
		Set("approved_at = NULL").
		Update()

	if err != nil {
		return err
	}

	return nil
}

// AddUser upserts the user into the database
func (c *Client) AddUser(user *User) error {
	_, err := c.Model(user).
		Where("email = ?", user.Email).
		Where("username = ?", user.Username).
		OnConflict("DO NOTHING").
		SelectOrInsert()

	return err
}
