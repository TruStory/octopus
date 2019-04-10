package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating flagged_stories table...")
		_, err := db.Exec(`CREATE TABLE flagged_stories(
			id SERIAL PRIMARY KEY,
			story_id bigint NOT NULL,
			creator VARCHAR (45) NOT NULL,
			created_on TIMESTAMP NOT NULL
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping flagged_stories table...")
		_, err := db.Exec(`DROP TABLE flagged_stories`)
		return err
	})
}
