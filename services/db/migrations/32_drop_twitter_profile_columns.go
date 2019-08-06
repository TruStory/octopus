package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("dropping twitter_profile_id column to the key_pairs table...")
		_, err := db.Exec(`ALTER TABLE key_pairs DROP COLUMN twitter_profile_id`)
		if err != nil {
			return err
		}

		fmt.Println("dropping twitter_profile_id column to the track_events table...")
		_, err = db.Exec(`ALTER TABLE track_events DROP COLUMN twitter_profile_id`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("adding twitter_profile_id column to the key_pairs table...")
		_, err := db.Exec(`ALTER TABLE key_pairs ADD COLUMN twitter_profile_id BIGINT`)
		if err != nil {
			return err
		}
		fmt.Println("indexing twitter_profile_id column on key_pairs table...")
		_, err = db.Exec(`CREATE INDEX idx_twitter_profile_id_on_key_pairs ON key_pairs(twitter_profile_id)`)
		if err != nil {
			return err
		}

		fmt.Println("adding twitter_profile_id column to the track_events table...")
		_, err = db.Exec(`ALTER TABLE track_events ADD COLUMN twitter_profile_id BIGINT`)
		return err
	})
}
