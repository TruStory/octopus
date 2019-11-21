package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("utru migration")
		_, err := db.Exec(`UPDATE reward_ledger_entries SET amount = (amount*0.001)::BIGINT, currency = 'utru' WHERE currency = 'tru'`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE leaderboard_user_metrics SET earned = (earned*0.001)::BIGINT`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("rollback to ntru")
		_, err := db.Exec(`UPDATE reward_ledger_entries SET amount = (amount*1000)::BIGINT, currency = 'tru' WHERE currency = 'utru'`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE leaderboard_user_metrics SET earned = (earned*1000)::BIGINT`)
		return err
	})
}
