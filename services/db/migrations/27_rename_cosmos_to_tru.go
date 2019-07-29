package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("rename cosmos addresses in all tables to tru addresses...")
		_, err := db.Exec(`UPDATE comments SET body = replace(body, '@cosmos', '@tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE comments SET creator = replace(creator, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE device_tokens SET address = replace(address, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE flagged_stories SET creator = replace(creator, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE initial_stake_balances SET address = replace(address, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE invites SET creator = replace(creator, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE notification_events SET address = replace(address, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET address = replace(address, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE twitter_profiles SET address = replace(address, 'cosmos', 'tru')`)
		if err != nil {
			return err
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("rename tru addresses in all tables to cosmos addresses...")
		_, err := db.Exec(`UPDATE comments SET body = replace(body, '@tru', '@cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE comments SET creator = replace(creator, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE device_tokens SET address = replace(address, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE flagged_stories SET creator = replace(creator, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE initial_stake_balances SET address = replace(address, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE invites SET creator = replace(creator, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE notification_events SET address = replace(address, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET address = replace(address, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE twitter_profiles SET address = replace(address, 'tru', 'cosmos')`)
		if err != nil {
			return err
		}
		return nil
	})
}
