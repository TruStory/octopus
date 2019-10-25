package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding domain_whitelist table...")

		_, err := db.Exec(`CREATE TABLE domain_whitelists (
			id BIGSERIAL PRIMARY KEY,
			domain TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping domain_whitelists table...")
		_, err := db.Exec(`DROP TABLE domain_whitelists`)
		return err
	})
}
