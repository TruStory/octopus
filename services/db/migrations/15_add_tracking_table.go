package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating track_events table...")
		_, err := db.Exec(`CREATE TABLE track_events(
			id BIGSERIAL PRIMARY KEY,
			address VARCHAR(65),
			twitter_profile_id BIGINT,
			event VARCHAR (65) NOT NULL,
			meta jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping track_events table...")
		_, err := db.Exec(`DROP TABLE track_events`)
		return err
	})
}
