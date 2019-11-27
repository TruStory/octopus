package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding phone verification column to the users table...")
		_, err := db.Exec(`ALTER TABLE users ADD COLUMN verified_phone_hash VARCHAR(65) UNIQUE DEFAULT NULL, ADD COLUMN phone_verified_at TIMESTAMP DEFAULT NULL`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping phone verification column from the users table...")
		_, err := db.Exec(`ALTER TABLE users DROP COLUMN verified_phone_hash, DROP COLUMN phone_verified_at`)
		return err
	})
}
