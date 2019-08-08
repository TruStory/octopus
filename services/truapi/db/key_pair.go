package db

import "github.com/go-pg/pg"

// KeyPair is the private key associated with an account
type KeyPair struct {
	Timestamps
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// KeyPairByUserID returns the key-pair for the user
func (c *Client) KeyPairByUserID(userID int64) (*KeyPair, error) {
	keyPair := new(KeyPair)
	err := c.Model(keyPair).Where("user_id = ?", userID).First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return keyPair, nil
}
