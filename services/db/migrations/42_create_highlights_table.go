package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating highlights table...")
		_, err := db.Exec(`CREATE TABLE highlights(
			id BIGSERIAL PRIMARY KEY,
			highlightable_type VARCHAR (65) NOT NULL,
			highlightable_id BIGINT NOT NULL,
			text TEXT NOT NULL,
			image_url TEXT DEFAULT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping highlights table...")
		_, err := db.Exec(`DROP TABLE highlights`)
		return err
	})
}
