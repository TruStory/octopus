package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("lowercase email column in users table...")
		_, err := db.Exec(`UPDATE users SET email = LOWER(email)`)
		if err != nil {
			return err
		}
		fmt.Println("unique index on lowercase username")
		_, err = db.Exec(`CREATE UNIQUE INDEX users_lower_case_usernames ON users ((lower(username)))`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("drop index users_lower_case_usernames on the users table...")
		_, err := db.Exec(`DROP INDEX users_lower_case_usernames`)
		return err
	})
}
