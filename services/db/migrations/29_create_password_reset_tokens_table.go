package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating password_reset_tokens table...")
		_, err := db.Exec(`CREATE TABLE password_reset_tokens(
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			token VARCHAR(128) NOT NULL,
			used_at TIMESTAMP DEFAULT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping password_reset_tokens table...")
		_, err := db.Exec(`DROP TABLE password_reset_tokens`)
		return err
	})
}
