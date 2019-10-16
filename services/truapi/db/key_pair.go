package db

import (
	"errors"

	"github.com/go-pg/pg"
)

// KeyPair is the private key associated with an account
type KeyPair struct {
	Timestamps
	ID                  int64  `json:"id"`
	UserID              int64  `json:"user_id"`
	PrivateKey          string `json:"private_key"`
	PublicKey           string `json:"public_key"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
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

func (c *Client) ReplacePrivateKeyWithEncryptedPrivateKey(id int64, encryptedPrivateKey string) error {
	var keyPair KeyPair
	result, err := c.Model(&keyPair).
		Where("id = ?", id).
		Set("encrypted_private_key = ?, private_key = ''", encryptedPrivateKey).
		Update()
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("key pair not found")
	}

	return nil
}
