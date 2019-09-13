package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding claim_video_url column to claim_images...")
		_, err := db.Exec(`ALTER TABLE claim_images ADD COLUMN claim_video_url TEXT`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("removing claim_video_url column from claim_images...")
		_, err := db.Exec(`ALTER TABLE claim_images DROP COLUMN claim_video_url`)
		return err
	})
}
