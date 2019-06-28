package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("rename community_slug column to community_id on claim_of_the_day_ids table...")
		_, err := db.Exec(`ALTER TABLE claim_of_the_day_ids RENAME COLUMN community_slug TO community_id`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("rename community_id column to community_slug on claim_of_the_day_ids table...")
		_, err := db.Exec(`ALTER TABLE claim_of_the_day_ids RENAME COLUMN community_id TO community_slug`)
		return err
	})
}
