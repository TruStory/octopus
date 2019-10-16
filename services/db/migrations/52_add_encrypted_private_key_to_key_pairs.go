package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding encrypted_private_key to the key_pairs table...")
		_, err := db.Exec(`ALTER TABLE key_pairs ADD COLUMN encrypted_private_key TEXT`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping encrypted_private_key from the key_pairs table...")
		_, err := db.Exec(`ALTER TABLE key_pairs DROP COLUMN encrypted_private_key`)
		return err
	})
}
