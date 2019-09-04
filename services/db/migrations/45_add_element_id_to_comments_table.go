package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding element_id column to comments...")
		_, err := db.Exec(`ALTER TABLE comments ADD COLUMN element_id BIGINT`)
		if err != nil {
			return err
		}
		fmt.Println("indexing argument_id/element_id column on comments table...")
		_, err = db.Exec(`CREATE INDEX idx_argument_id_element_id_on_comments ON comments(argument_id, element_id)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("drop index on argument_id/element_id column from comments table...")
		_, err := db.Exec(`DROP INDEX idx_argument_id_element_id_on_comments`)
		if err != nil {
			return err
		}
		fmt.Println("removing element_id column from comments...")
		_, err = db.Exec(`ALTER TABLE comments DROP COLUMN element_id`)
		return err
	})
}
