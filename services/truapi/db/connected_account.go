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

// ConnectedAccountsByUserID returns the connected accounts for a given user
func (c *Client) ConnectedAccountsByUserID(userID int64) ([]ConnectedAccount, error) {
	var connectedAccounts []ConnectedAccount

	err := c.Model(&connectedAccounts).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Select()

	if err != nil {
		return []ConnectedAccount{}, err
	}

	return connectedAccounts, nil
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
			meta = json_build_object(
				'email', EXCLUDED.meta->'email',
				'username', EXCLUDED.meta->'username',
				'bio', EXCLUDED.meta->'bio',
				'full_name', EXCLUDED.meta->'full_name',
				'avatar_url', EXCLUDED.meta->'avatar_url'
			)::jsonb,
			updated_at = NOW()
		`).
		Insert()

	return err
}
