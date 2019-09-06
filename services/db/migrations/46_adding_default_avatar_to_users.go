package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding default avatar to users without avatar...")
		_, err := db.Exec(`update users set avatar_url = 'https://trustory.s3-us-west-1.amazonaws.com/images/tru-avatar.jpg' where avatar_url is null;`)
		return err
	}, func(db migrations.DB) error {
		// no going back
		return nil
	})
}
