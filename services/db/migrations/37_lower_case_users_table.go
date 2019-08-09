package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("lowercase email and username column in users table...")
		_, err := db.Exec(`UPDATE users SET email = LOWER(email), username = LOWER(username)`)
		return err
	}, func(db migrations.DB) error {
		// no going back!
		return nil
	})
}
