package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("update meta for older notifications")
		_, err := db.Exec(`UPDATE notification_events as ne 
			SET meta = json_build_object('storyId', ne.type_id) 
			WHERE ne.type = 0 and  ne.meta = '{}' `)

		return err
	}, func(db migrations.DB) error {
		fmt.Println("update meta for older notifications")
		_, err := db.Exec(`UPDATE notification_events as ne SET meta = '{}' where ne.type = 0`)
		return err
	})
}
