package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating claim_source_url_previews table...")
		_, err := db.Exec(`CREATE TABLE claim_source_url_previews(
			claim_id BIGSERIAL PRIMARY KEY,
			source_url_preview TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping claim_source_url_previews table...")
		_, err := db.Exec(`DROP TABLE claim_source_url_previews`)
		return err
	})
}
