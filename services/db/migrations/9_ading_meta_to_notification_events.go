package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding meta column to notification_events...")
		_, err := db.Exec(`ALTER TABLE notification_events ADD COLUMN meta jsonb, ALTER COLUMN read SET DEFAULT FALSE `)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE notification_events SET read = false WHERE read is NULL `)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing meta columns from notification_events...")
		_, err := db.Exec(`ALTER TABLE notification_events DROP COLUMN meta`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE notification_events SET read = NULL WHERE read is FALSE `)
		return err
		return err
	})
}
