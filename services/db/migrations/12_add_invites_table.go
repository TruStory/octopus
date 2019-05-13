package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating invites table...")
		_, err := db.Exec(`CREATE TABLE invites(
			id BIGSERIAL PRIMARY KEY,
			creator VARCHAR (65) NOT NULL,
			friend_twitter_username TEXT UNIQUE,
			friend_email TEXT UNIQUE,
			paid BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping invites table...")
		_, err := db.Exec(`DROP TABLE invites`)
		return err
	})
}
