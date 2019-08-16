package db

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-pg/pg"
)

// PasswordResetToken represents a token to reset password
type PasswordResetToken struct {
	Timestamps

	ID     int64     `json:"id"`
	UserID int64     `json:"user_id"`
	Token  string    `json:"token"`
	UsedAt time.Time `json:"used_at"`
}

// UnusedResetTokenByUserAndToken returns the unused reset token by the user id and the token
func (c *Client) UnusedResetTokenByUserAndToken(userID int64, token string) (*PasswordResetToken, error) {
	var prt = new(PasswordResetToken)
	err := c.Model(prt).
		Where("used_at IS NULL").
		Where("deleted_at IS NULL").
		Where("user_id = ?", userID).
		Where("token = ?", token).
		First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return prt, nil
}

// UnusedResetTokensByUser returns all the unused reset tokens by the user id, latest first
func (c *Client) UnusedResetTokensByUser(userID int64) ([]PasswordResetToken, error) {
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
func (c *Client) IssueResetToken(userID int64) (*PasswordResetToken, error) {
	token, err := generateCryptoSafeRandomBytes(64)
	if err != nil {
		return nil, errors.New("token could not be generated")
	}

	prt := &PasswordResetToken{
		UserID: userID,
		Token:  hex.EncodeToString(token),
	}
	_, err = c.Model(prt).Insert()
	if err != nil {
		return nil, err
	}

	// revoke all previously issued tokens
	_, err = c.Model(prt).
		Where("token <> ?", prt.Token).
		Where("used_at IS NULL").
		Set("deleted_at = NOW()").
		Update()
	if err != nil {
		return nil, err
	}

	return prt, nil
}

// UseResetToken uses a token
func (c *Client) UseResetToken(prt *PasswordResetToken) error {
	result, err := c.Model(prt).
		Where("user_id = ?", prt.UserID).
		Where("token = ?", prt.Token).
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
