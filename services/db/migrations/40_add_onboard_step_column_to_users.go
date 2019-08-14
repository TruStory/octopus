package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding meta column to the users table...")
		_, err := db.Exec(`ALTER TABLE users ADD COLUMN meta jsonb DEFAULT '{}'`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping meta column from the users table...")
		_, err := db.Exec(`ALTER TABLE users DROP COLUMN meta`)
		return err
	})
}
