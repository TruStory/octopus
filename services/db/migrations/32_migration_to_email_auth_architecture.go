package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	truDB "github.com/TruStory/octopus/services/truapi/db"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("seeding users table...")
		_, err := db.Exec(`INSERT INTO users(
			full_name, 
			email,
			bio,
			avatar_url,
			username,
			address,
			created_at,
			updated_at,
			approved_at
		) SELECT 
			full_name,
			email,
			description,
			avatar_uri,
			username,
			address,
			created_at,
			updated_at,
			NOW()
		FROM twitter_profiles;`)
		if err != nil {
			return err
		}

		fmt.Println("seeding connected_accounts table...")
		_, err = db.Exec(`INSERT INTO connected_accounts(
			user_id,
			account_type,
			account_id,
			meta
		) SELECT
			users.id,
			'twitter',
			twitter_profiles.id,
			json_build_object(
				'email', twitter_profiles.email,
				'username', twitter_profiles.username,
				'bio', twitter_profiles.description,
				'full_name', twitter_profiles.full_name,
				'avatar_url', twitter_profiles.avatar_uri
			)::jsonb
		FROM users JOIN twitter_profiles ON users.address = twitter_profiles.address;`)
		if err != nil {
			return err
		}

		fmt.Println("seeding user_id column in the key_pairs table...")
		_, err = db.Exec(`UPDATE key_pairs
			SET user_id = connected_accounts.user_id
			FROM connected_accounts
			WHERE 
				connected_accounts.account_type = 'twitter'
				AND connected_accounts.account_id = key_pairs.twitter_profile_id::varchar(256)`)
		if err != nil {
			return err
		}

		fmt.Println("seeding unique tokens in all the users...")
		var users []truDB.User
		err = db.Model(&users).Order("id ASC").Select()
		if err != nil {
			return err
		}
		for _, user := range users {
			token, err := generateCryptoSafeRandomBytes(32)
			if err != nil {
				return err
			}
			user.Token = base64.StdEncoding.EncodeToString(token)
			err = db.Update(&user)
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("truncating users table...")
		_, err := db.Exec(`TRUNCATE TABLE users RESTART IDENTITY`)
		if err != nil {
			return err
		}

		fmt.Println("truncating connected_accounts table...")
		_, err = db.Exec(`TRUNCATE TABLE connected_accounts RESTART IDENTITY`)
		if err != nil {
			return err
		}

		fmt.Println("truncating user_id column in the key_pairs table...")
		_, err = db.Exec(`UPDATE key_pairs SET user_id = NULL`)
		return err
	})
}

func generateCryptoSafeRandomBytes(strength int) ([]byte, error) {
	random := make([]byte, strength)
	_, err := rand.Read(random)
	if err != nil {
		return nil, err
	}

	return random, nil
}
