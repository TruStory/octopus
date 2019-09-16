package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding reward_ledger_entries table...")
		_, err := db.Exec(`CREATE TYPE ledger_direction AS ENUM ('credit', 'debit')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`CREATE TABLE reward_ledger_entries (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			direction ledger_direction NOT NULL,
			amount BIGINT NOT NULL,
			currency VARCHAR(65) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping reward_ledger table...")
		_, err := db.Exec(`DROP TABLE reward_ledger_entries`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`DROP TYPE ledger_direction`)
		return err
	})
}
