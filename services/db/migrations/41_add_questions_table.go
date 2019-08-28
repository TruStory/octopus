package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating questions table...")
		_, err := db.Exec(`CREATE TABLE questions(
			id BIGSERIAL PRIMARY KEY,
			claim_id BIGINT NOT NULL,
			body TEXT NOT NULL,
			creator VARCHAR (45) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping questions table...")
		_, err := db.Exec(`DROP TABLE questions`)
		return err
	})
}
