package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating reactions table...")
		_, err := db.Exec(`CREATE TABLE reactions(
			id BIGSERIAL PRIMARY KEY,
			reactionable_type VARCHAR (65) NOT NULL,
			reactionable_id BIGINT NOT NULL,
			reaction_type INTEGER NOT NULL,
			creator VARCHAR (65) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping reactions table...")
		_, err := db.Exec(`DROP TABLE reactions`)
		return err
	})
}
