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
			category_id INTEGER NOT NULL,
			total_claims BIGINT NOT NULL,
			total_arguments BIGINT NOT NULL,
			total_claims_backed BIGINT NOT NULL,
			total_claims_challenged BIGINT NOT NULL,
			total_amount_backed BIGINT NOT NULL,
			total_amount_challenged BIGINT NOT NULL,
			total_endorsements_given BIGINT NOT NULL,
			total_endorsements_received BIGINT NOT NULL,
			stake_earned BIGINT NOT NULL,
			stake_lost BIGINT NOT NULL,
			stake_balance BIGINT NOT NULL,
			interest_earned BIGINT NOT NULL,
			total_amount_at_stake BIGINT NOT NULL,
			total_amount_staked BIGINT NOT NULL,
			cred_earned BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP,
			CONSTRAINT no_duplicate_metric UNIQUE(address, as_on_date, category_id)
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping user_metrics table...")
		_, err := db.Exec(`DROP TABLE user_metrics`)
		return err
	})
}
