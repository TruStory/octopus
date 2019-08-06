package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating users table...")
		_, err := db.Exec(`CREATE TABLE users(
			id BIGSERIAL PRIMARY KEY,
			full_name VARCHAR(128) NOT NULL,
			bio TEXT DEFAULT NULL,
			avatar_url TEXT DEFAULT NULL,
			email VARCHAR(128) UNIQUE DEFAULT NULL,
			username VARCHAR(128) UNIQUE DEFAULT NULL,
			address VARCHAR(65) DEFAULT NULL,
			password VARCHAR(256) DEFAULT NULL,
			referred_by BIGINT DEFAULT NULL,
			token VARCHAR(65) DEFAULT NULL,
			approved_at TIMESTAMP DEFAULT NULL,
			rejected_at TIMESTAMP DEFAULT NULL,
			verified_at TIMESTAMP DEFAULT NULL,
			blacklisted_at TIMESTAMP DEFAULT NULL,
			last_authenticated_at TIMESTAMP DEFAULT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		if err != nil {
			return err
		}

		fmt.Println("indexing email column on users table...")
		_, err = db.Exec(`CREATE INDEX idx_email_on_users ON users(email)`)
		if err != nil {
			return err
		}

		fmt.Println("indexing username column on users table...")
		_, err = db.Exec(`CREATE INDEX idx_username_on_users ON users(username)`)
		if err != nil {
			return err
		}

		fmt.Println("indexing address column on users table...")
		_, err = db.Exec(`CREATE INDEX idx_address_on_users ON users(address)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping users table...")
		_, err := db.Exec(`DROP TABLE users`)
		return err
	})
}
