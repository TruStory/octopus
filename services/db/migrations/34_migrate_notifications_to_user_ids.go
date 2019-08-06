package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("replacing twitter_profile_id's with user_id's in the notification_events table")
		_, err := db.Exec(`UPDATE notification_events
			SET twitter_profile_id = connected_accounts.user_id
			FROM connected_accounts
			WHERE 
				connected_accounts.account_id::BIGINT = notification_events.twitter_profile_id`)
		if err != nil {
			return err
		}

		fmt.Println("replacing sender_profile_id's with user_id's in the notification_events table")
		_, err = db.Exec(`UPDATE notification_events
			SET sender_profile_id = connected_accounts.user_id
			FROM connected_accounts
			WHERE 
				connected_accounts.account_id::BIGINT = notification_events.sender_profile_id`)
		if err != nil {
			return err
		}

		fmt.Println("rename twitter_profile_id column to user_profile_id on notification_events table...")
		_, err = db.Exec(`ALTER TABLE notification_events RENAME COLUMN twitter_profile_id TO user_profile_id`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("rename user_profile_id column to twitter_profile_id on notification_events table...")
		_, err := db.Exec(`ALTER TABLE notification_events RENAME COLUMN user_profile_id TO twitter_profile_id`)
		if err != nil {
			return err
		}

		fmt.Println("replacing user_id's with twitter_profile_id's in the notification_events table")
		_, err = db.Exec(`UPDATE notification_events
			SET twitter_profile_id = connected_accounts.account_id::BIGINT
			FROM connected_accounts
			WHERE 
				connected_accounts.user_id = notification_events.twitter_profile_id`)
		if err != nil {
			return err
		}

		fmt.Println("replacing user_id's with sender_profile_id's in the notification_events table")
		_, err = db.Exec(`UPDATE notification_events
			SET sender_profile_id = connected_accounts.account_id::BIGINT
			FROM connected_accounts
			WHERE 
				connected_accounts.user_id = notification_events.sender_profile_id`)
		return err
	})
}
