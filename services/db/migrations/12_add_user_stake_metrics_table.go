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
			as_on_date DATE NOT NULL DEFAULT CURRENT_DATE,
			total_claims BIGINT NOT NULL,
			total_arguments BIGINT NOT NULL,
			total_backed BIGINT NOT NULL,
			total_challenged BIGINT NOT NULL,
			total_given_endorsments BIGINT NOT NULL,
			stake_earned BIGINT NOT NULL,
			stake_lost BIGINT NOT NULL,
			interest_earned BIGINT NOT NULL,
			total_amount_at_stake BIGINT NOT NULL,
			total_amount_staked BIGINT NOT NULL,
			balance BIGINT NOT NULL,
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
