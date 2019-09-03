package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding element_id column to comments...")
		_, err := db.Exec(`ALTER TABLE comments ADD COLUMN element_id BIGINT`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing element_id column from comments...")
		_, err := db.Exec(`ALTER TABLE comments DROP COLUMN element_id`)
		return err
	})
}
