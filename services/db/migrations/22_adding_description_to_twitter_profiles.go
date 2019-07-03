package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding description column to twitter_profiles...")
		_, err := db.Exec(`ALTER TABLE twitter_profiles ADD COLUMN description VARCHAR(320)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing description column from twitter_profiles...")
		_, err := db.Exec(`ALTER TABLE twitter_profiles DROP COLUMN description`)
		return err
	})
}
