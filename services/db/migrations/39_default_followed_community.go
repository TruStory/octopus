package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding default followed community")

		_, err := db.Exec(`INSERT INTO followed_communities (address, community_id, following_since, created_at, updated_at)
				SELECT
					address,
					'general',
					now(),
					now(),
					now()
				FROM
					users ON CONFLICT DO NOTHING;`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE users SET last_authenticated_at = NULL;`)
		return err
	}, func(db migrations.DB) error {
		return nil
	})
}
