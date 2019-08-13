package db

import (
	"fmt"
	"strconv"

	"github.com/go-pg/pg"
)

// TwitterProfile is the Twitter profile associated with an account
type TwitterProfile struct {
	Timestamps
	ID          int64  `json:"id"`
	Address     string `json:"address"`
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	AvatarURI   string `json:"avatar_uri"`
	Description string `json:"description"`
}

func (t TwitterProfile) String() string {
	return fmt.Sprintf(
		"Twitter Profile<%d %s %s %s %s>",
		t.ID, t.Address, t.Username, t.FullName, t.AvatarURI)
}

// TwitterProfileByAddress implements `Datastore`
// Finds a Twitter profile by the given address
// Deprecated: use UserProfileByAddress instead
func (c *Client) TwitterProfileByAddress(addr string) (*TwitterProfile, error) {
	twitterProfile := new(TwitterProfile)
	user := new(User)
	err := c.Model(user).Where("address = ?", addr).Select()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return twitterProfile, err
	}

	twitterProfile = &TwitterProfile{
		Timestamps:  user.Timestamps,
		Address:     user.Address,
		Username:    user.Username,
		FullName:    user.FullName,
		Email:       "",
		AvatarURI:   user.AvatarURL,
		Description: user.Bio,
	}

	connectedAccount := new(ConnectedAccount)
	err = c.Model(connectedAccount).Where("user_id = ?", user.ID).Select()
	if err != nil {
		return twitterProfile, nil
	}

	twitterProfile.ID, err = strconv.ParseInt(connectedAccount.AccountID, 10, 64)
	if err != nil {
		return twitterProfile, nil
	}

	return twitterProfile, nil
}

// TwitterProfileByUsername implements `Datastore`
// Finds a Twitter profile by the given username
// Deprecated: use UserProfileByUsername instead
func (c *Client) TwitterProfileByUsername(username string) (*TwitterProfile, error) {
	twitterProfile := new(TwitterProfile)
	user := new(User)
	err := c.Model(user).Where("username = ?", username).First()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return twitterProfile, err
	}

	twitterProfile = &TwitterProfile{
		Timestamps:  user.Timestamps,
		ID:          user.ID,
		Address:     user.Address,
		Username:    user.Username,
		FullName:    user.FullName,
		Email:       "",
		AvatarURI:   user.AvatarURL,
		Description: user.Bio,
	}

	connectedAccount := new(ConnectedAccount)
	err = c.Model(connectedAccount).Where("user_id = ?", user.ID).Select()
	if err != nil {
		return twitterProfile, nil
	}

	twitterProfile.ID, err = strconv.ParseInt(connectedAccount.AccountID, 10, 64)
	if err != nil {
		return twitterProfile, nil
	}

	return twitterProfile, nil
}
