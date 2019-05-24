package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating initial_stake_balances table...")
		_, err := db.Exec(`CREATE TABLE initial_stake_balances(
			address VARCHAR (65) PRIMARY KEY,
			initial_balance BIGINT NOT NULL
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping initial_stake_balances table...")
		_, err := db.Exec(`DROP TABLE initial_stake_balances`)
		return err
	})
}
