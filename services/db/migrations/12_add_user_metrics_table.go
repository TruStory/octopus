package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating user_metrics table...")
		_, err := db.Exec(`CREATE TABLE user_metrics(
			id BIGSERIAL PRIMARY KEY,
			address VARCHAR (65) NOT NULL,
			as_on_date DATE NOT NULL DEFAULT CURRENT_DATE,
			total_claims VARCHAR (65) NOT NULL,
			total_arguments VARCHAR (65) NOT NULL,
			total_backed VARCHAR (65) NOT NULL,
			total_challenged VARCHAR (65) NOT NULL,
			total_given_endorsments VARCHAR (65) NOT NULL,
			stake_earned VARCHAR (65) NOT NULL,
			stake_lost VARCHAR (65) NOT NULL,
			interest_earned VARCHAR (65) NOT NULL,
			total_amount_at_stake VARCHAR (65) NOT NULL,
			total_amount_staked VARCHAR (65) NOT NULL,
			balance VARCHAR (65) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping user_metrics table...")
		_, err := db.Exec(`DROP TABLE user_stake_metrics`)
		return err
	})
}
