package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding authenticated_count column to the users table...")
		_, err := db.Exec(`ALTER TABLE users ADD COLUMN authenticated_count BIGINT NOT NULL DEFAULT 0`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping authenticated_count column from the users table...")
		_, err := db.Exec(`ALTER TABLE users DROP COLUMN authenticated_count`)
		return err
	})
}
