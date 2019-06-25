package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating claim_comments table...")
		_, err := db.Exec(`CREATE TABLE claim_comments(
			id BIGSERIAL PRIMARY KEY,
			parent_id BIGINT,
			claim_id BIGINT NOT NULL,
			body TEXT NOT NULL,
			creator VARCHAR (45) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping claim_comments table...")
		_, err := db.Exec(`DROP TABLE claim_comments`)
		return err
	})
}
