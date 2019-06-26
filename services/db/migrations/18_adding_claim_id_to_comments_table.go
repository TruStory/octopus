package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding claim_id column to comments...")
		_, err := db.Exec(`ALTER TABLE comments ADD COLUMN claim_id BIGINT, ALTER COLUMN argument_id DROP NOT NULL`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing claim_id column from comments...")
		_, err := db.Exec(`ALTER TABLE comments DROP COLUMN claim_id, ALTER COLUMN argument_id SET NOT NULL`)
		return err
	})
}
