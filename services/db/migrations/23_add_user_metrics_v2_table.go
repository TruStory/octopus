package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating user_metrics_v2 table...")
		_, err := db.Exec(`CREATE TABLE user_metrics_v2(
			id BIGSERIAL PRIMARY KEY,
			address VARCHAR (65) NOT NULL,
			as_on_date DATE NOT NULL DEFAULT CURRENT_DATE,
			community_id VARCHAR (65) NOT NULL,
			total_amount_at_stake BIGINT NOT NULL,
			stake_earned BIGINT NOT NULL,
			stake_lost BIGINT NOT NULL,
			total_amount_staked BIGINT NOT NULL,
			available_stake BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP,
			CONSTRAINT no_duplicate_user_metric UNIQUE(address, as_on_date, community_id)
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping user_metrics_v2 table...")
		_, err := db.Exec(`DROP TABLE user_metrics_v2`)
		return err
	})
}
