package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating comments table...")
		_, err := db.Exec(`CREATE TABLE comments(
			id SERIAL PRIMARY KEY,
			parent_id INTEGER,
			argument_id bigint NOT NULL,
			creator VARCHAR (45) NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping comments table...")
		_, err := db.Exec(`DROP TABLE comments`)
		return err
	})
}
