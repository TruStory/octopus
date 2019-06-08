package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding seen column to notification_events...")
		_, err := db.Exec(`ALTER TABLE notification_events ADD COLUMN seen BOOLEAN, ALTER COLUMN seen SET DEFAULT FALSE `)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE notification_events SET seen = read`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing seen column from notification_events...")
		_, err := db.Exec(`ALTER TABLE notification_events DROP COLUMN seen`)
		return err
	})
}
