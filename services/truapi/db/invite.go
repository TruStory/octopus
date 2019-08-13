package db

import (
	"strings"

	"github.com/go-pg/pg"
)

// Invite represents an invite from a friend in the DB
type Invite struct {
	ID                    int64  `json:"id"`
	Creator               string `json:"creator"`
	FriendTwitterUsername string `json:"friend_twitter_username"`
	FriendEmail           string `json:"friend_email"`
	Paid                  bool   `json:"paid"`
	Timestamps
}

// Invites returns all invites in theDB
func (c *Client) Invites() ([]Invite, error) {
	invites := make([]Invite, 0)
	err := c.Model(&invites).Select()
	if err != nil {
		return nil, err
	}

	return invites, nil
}

// InvitesByAddress returns all invites for a specific address
func (c *Client) InvitesByAddress(addr string) ([]Invite, error) {
	invites := make([]Invite, 0)
	err := c.Model(&invites).Where("creator = ?", addr).Select()
	if err != nil {
		return nil, err
	}

	return invites, nil
}

// InvitesByFriendEmail returns the invite for a specific friend's email
func (c *Client) InvitesByFriendEmail(email string) (*Invite, error) {
	var invite Invite
	err := c.Model(&invite).Where("friend_email = ?", email).First()
	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &invite, nil
}

// AddInvite inserts an invitation
func (c *Client) AddInvite(invite *Invite) error {
	if invite != nil {
		invite.FriendEmail = strings.ToLower((*invite).FriendEmail)
		_, err := c.Model(invite).
			OnConflict("DO NOTHING").
			Insert()

		return err
	}
	return nil
}
