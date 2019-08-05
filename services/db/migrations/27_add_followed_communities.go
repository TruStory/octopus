package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating followed_communities table...")
		_, err := db.Exec(`CREATE TABLE followed_communities(
			id BIGSERIAL PRIMARY KEY NOT NULL,
			address VARCHAR(65) NOT NULL,
			community_id VARCHAR(65) NOT NULL,
			following_since TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP,
			CONSTRAINT no_duplicate_address_community UNIQUE(address, community_id)
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping followed_communities table...")
		_, err := db.Exec(`DROP TABLE followed_communities`)
		return err
	})
}
