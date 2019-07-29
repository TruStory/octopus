package db

import (
	"errors"
	"time"
)

const (
	PasswordResetTokenValidity            = 2 * time.Hour
	PasswordResetTokenLimitWithinValidity = 3
)

// PasswordResetToken represents a token to reset password
type PasswordResetToken struct {
	Timestamps

	ID     uint64    `json:"id"`
	UserID uint64    `json:"user_id"`
	Token  string    `json:"token"`
	UsedAt time.Time `json:"used_at"`
}

// UnusedResetTokensByUser returns all the unused reset tokens by the user id, latest first
func (c *Client) UnusedResetTokensByUser(userID uint64) ([]PasswordResetToken, error) {
	var prts = make([]PasswordResetToken, 0)
	err := c.Model(&prts).
		Where("used_at IS NULL").
		Where("deleted_at IS NULL").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Select()
	if err != nil {
		return prts, err
	}

	return prts, nil
}

// IssueResetToken inserts a token into the database
func (c *Client) IssueResetToken(userID uint64) (*PasswordResetToken, error) {
	unused, err := c.UnusedResetTokensByUser(userID)
	if err != nil {
		return nil, err
	}
	// if within the validity period, too many resets are issued
	if len(unused) >= PasswordResetTokenLimitWithinValidity {
		if time.Now().Before(unused[PasswordResetTokenLimitWithinValidity-1].CreatedAt.Add(PasswordResetTokenValidity)) {
			return nil, errors.New("too many password reset tokens are issued")
		}
	}

	// all well...
	token, err := generateToken(64)
	if err != nil {
		return nil, errors.New("token could not be generated")
	}

	prt := &PasswordResetToken{
		UserID: userID,
		Token:  token,
	}
	_, err = c.Model(prt).Insert()
	if err != nil {
		return nil, err
	}

	return prt, nil
}

// UseResetToken uses a token
func (c *Client) UseResetToken(userID uint64, token string) error {
	prt := new(PasswordResetToken)
	result, err := c.Model(prt).
		Where("user_id = ?", userID).
		Where("token = ?", token).
		Where("used_at IS NULL").
		Where("deleted_at IS NULL").
		Set("used_at = ?", time.Now()).
		Update()

	if result.RowsAffected() == 0 {
		return errors.New("invalid token")
	}

	if err != nil {
		return err
	}

	return nil
}
