package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding session id and is_anonymous column to track_Events...")
		_, err := db.Exec(`ALTER TABLE track_events ADD COLUMN session_id UUID,ADD COLUMN  is_anonymous BOOLEAN default FALSE`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing session id and is_anonymous column to track_events....")
		_, err := db.Exec(`ALTER TABLE track_events  DROP COLUMN session_id, DROP COLUMN is_anonymous`)
		return err
	})
}
