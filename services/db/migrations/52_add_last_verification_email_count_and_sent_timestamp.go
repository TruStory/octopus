package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding last_verification_attempt_at, verification_attempt_count column to the users table...")
		_, err := db.Exec(`ALTER TABLE users ADD COLUMN last_verification_attempt_at TIMESTAMP DEFAULT NULL, ADD COLUMN verification_attempt_count INTEGER DEFAULT 0`)
		if err != nil {
			return err
		}
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping last_verification_attempt_at, verification_attempt_count column from the users table...")
		_, err := db.Exec(`ALTER TABLE users DROP COLUMN last_verification_attempt_at, DROP COLUMN verification_attempt_count`)
		return err
	})
}
