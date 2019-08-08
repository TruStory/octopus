package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding user_id column to the key_pairs table...")
		_, err := db.Exec(`ALTER TABLE key_pairs ADD COLUMN user_id BIGINT`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing user_id column from the key_pairs table...")
		_, err := db.Exec(`ALTER TABLE key_pairs DROP COLUMN user_id`)
		return err
	})
}
