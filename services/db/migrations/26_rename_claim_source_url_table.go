package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("rename claim_source_url_previews table to claim_images...")
		_, err := db.Exec(`ALTER TABLE claim_source_url_previews RENAME TO claim_images`)
		if err != nil {
			return err
		}
		fmt.Println("rename source_url_preview column to claim_image_url on claim_images table...")
		_, err = db.Exec(`ALTER TABLE claim_images RENAME COLUMN source_url_preview TO claim_image_url`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("rename source_url_preview column to claim_image_url on claim_images table...")
		_, err := db.Exec(`ALTER TABLE claim_images RENAME COLUMN claim_image_url TO source_url_preview`)
		if err != nil {
			return err
		}
		fmt.Println("rename claim_images table to claim_source_url_previews...")
		_, err = db.Exec(`ALTER TABLE claim_images RENAME TO claim_source_url_previews`)
		return err
	})
}
