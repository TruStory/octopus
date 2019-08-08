package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("filling referred_by from the invites table...")
		_, err := db.Exec(`UPDATE users SET referred_by = referrers.id
			FROM invites JOIN users referrers ON referrers.address = invites.creator
			WHERE invites.friend_email = users.email;`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("nulling referred_by from the invites table")
		_, err := db.Exec(`UPDATE users SET referred_by = NULL
			FROM invites JOIN users referrers ON referrers.address = invites.creator
			WHERE invites.friend_email = users.email;`)
		return err
	})
}
