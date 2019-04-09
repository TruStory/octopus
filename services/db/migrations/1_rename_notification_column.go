package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("renaming story_id column to type_id...")
		_, err := db.Exec(`ALTER TABLE notification_events RENAME COLUMN story_id TO type_id`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("renaming type_id column to story_id...")
		_, err := db.Exec(`ALTER TABLE notification_events RENAME COLUMN type_id TO story_id`)
		return err
	})
}
