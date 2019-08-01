package db

import (
	"github.com/go-pg/pg"
)

// ConnectedAccountMeta represents the meta data for the connected accounts
type ConnectedAccountMeta struct {
	Email     string `json:"email,omitempty"`
	Bio       string `json:"bio,omitempty"`
	Username  string `json:"username,omitempty"`
	FullName  string `json:"full_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// ConnectedAccount represents the third-party accounts connected to a user
type ConnectedAccount struct {
	Timestamps

	ID          int64                `json:"id"`
	UserID      int64                `json:"user_id"`
	AccountType string               `json:"account_type"`
	AccountID   string               `json:"account_id"`
	Meta        ConnectedAccountMeta `json:"meta"`
}

// ConnectedAccountByTypeAndID returns the connected account by type and id
func (c *Client) ConnectedAccountByTypeAndID(accountType, accountID string) (*ConnectedAccount, error) {
	var connectedAccount ConnectedAccount

	err := c.Model(&connectedAccount).
		Where("account_type = ?", accountType).
		Where("account_id = ?", accountID).
		Where("deleted_at IS NULL").
		First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &connectedAccount, nil
}

// UpsertConnectedAccount updates or inserts the connected account of a user
func (c *Client) UpsertConnectedAccount(connectedAccount *ConnectedAccount) error {
	_, err := c.Model(connectedAccount).
		OnConflict("(account_type, account_id) DO UPDATE").
		Set(`
			meta = 
				jsonb_set('{}', array['email'], EXCLUDED.meta->'email', true)
				|| jsonb_set('{}', array['username'], EXCLUDED.meta->'username', true)
				|| jsonb_set('{}', array['bio'], EXCLUDED.meta->'bio', true)
				|| jsonb_set('{}', array['full_name'], EXCLUDED.meta->'full_name', true)
				|| jsonb_set('{}', array['avatar_url'], EXCLUDED.meta->'avatar_url', true)
		`).
		Insert()

	return err
}
