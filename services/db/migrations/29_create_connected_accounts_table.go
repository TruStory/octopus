package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating connected_accounts table...")
		_, err := db.Exec(`CREATE TABLE connected_accounts(
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			account_type VARCHAR(128) NOT NULL,
			account_id VARCHAR(256) NOT NULL,
			meta JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP,
			CONSTRAINT no_duplicate_connected_accounts UNIQUE(account_type, account_id)
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping connected_accounts table...")
		_, err := db.Exec(`DROP TABLE connected_accounts`)
		return err
	})
}
