package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating user_cred_metrics table...")
		_, err := db.Exec(`CREATE TABLE user_cred_metrics(
			id BIGSERIAL PRIMARY KEY,
			address VARCHAR (65) NOT NULL,
			as_on_date DATE NOT NULL DEFAULT CURRENT_DATE,
			cred_earned_denom VARCHAR (65) NOT NULL,
			cred_earned_amount VARCHAR (65) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping user_cred_metrics table...")
		_, err := db.Exec(`DROP TABLE user_cred_metrics`)
		return err
	})
}
