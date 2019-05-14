package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating user_stake_metrics table...")
		_, err := db.Exec(`CREATE TABLE user_stake_metrics(
			id BIGSERIAL PRIMARY KEY,
			address VARCHAR (65) NOT NULL,
			category_id INTEGER NOT NULL,
			as_on_date DATE NOT NULL DEFAULT CURRENT_DATE,
			stakes_earned VARCHAR (65) NOT NULL,
			stakes_lost VARCHAR (65) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping user_stake_metrics table...")
		_, err := db.Exec(`DROP TABLE user_stake_metrics`)
		return err
	})
}
