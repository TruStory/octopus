package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {

		communities := []string{
			"general",
			"crypto",
			"environment",
			"livedebates",
			"politics",
			"privacy",
			"programming",
			"trustory",
		}
		query := `INSERT INTO 
		followed_communities(address, community_id, following_since, created_at, updated_at)
	   SELECT address, '%s' community_id, now() following_since, now() created_at, now()  updated_at FROM users
	   WHERE address is not null
	   ON CONFLICT DO NOTHING`
		for _, c := range communities {
			q := fmt.Sprintf(query, c)
			_, err := db.Exec(q)
			if err != nil {
				return err
			}
		}
		return nil

	}, func(db migrations.DB) error {
		return nil
	})
}
