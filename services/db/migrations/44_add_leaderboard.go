package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating leaderboard tables...")
		_, err := db.Exec(`CREATE TABLE leaderboard_processed_dates (
			id BIGSERIAL PRIMARY KEY,
			date DATE NOT NULL,
			metrics_time TIMESTAMP NOT NULL,
			full_day BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP,
			CONSTRAINT leaderboard_processed_no_duplicate UNIQUE(date)
		)`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`CREATE TABLE leaderboard_user_metrics (
			id BIGSERIAL PRIMARY KEY,
            date TIMESTAMP NOT NULL,
			address VARCHAR (65) NOT NULL,
			community_id VARCHAR(75) NOT NULL,
			earned BIGINT NOT NULL ,
			agrees_received BIGINT NOT NULL,
			agrees_given BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP,
			CONSTRAINT leaderboard_metrics_no_duplicate_date UNIQUE(date, address, community_id)
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping connected_accounts table...")
		_, err := db.Exec(`DROP TABLE leaderboard_processed_dates`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`DROP TABLE leaderboard_user_metrics`)
		return err
	})
}
