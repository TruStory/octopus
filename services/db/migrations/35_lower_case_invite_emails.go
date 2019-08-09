package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("lowercase friend_email column in invites table...")
		_, err := db.Exec(`UPDATE invites SET friend_email = LOWER(friend_email)`)
		return err
	}, func(db migrations.DB) error {
		// no going back!
		return nil
	})
}
