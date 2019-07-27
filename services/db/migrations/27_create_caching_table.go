package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating cached_feeds table...")
		_, err := db.Exec(`CREATE TABLE cached_feeds(
			id VARCHAR (65) PRIMARY KEY,
			feed TEXT NOT NULL
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping cached_feeds table...")
		_, err := db.Exec(`DROP TABLE cached_feeds`)
		return err
	})
}
