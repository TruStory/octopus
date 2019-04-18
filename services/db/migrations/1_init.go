package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

// Migration constants.
const (
	InitCreateTwitterProfiles = `
		CREATE TABLE twitter_profiles (
			id BIGSERIAL NOT NULL PRIMARY KEY,
			address VARCHAR (45) NOT NULL,
			username TEXT,
			full_name TEXT,
			email TEXT,
			avatar_uri TEXT
		);
	`
	InitCreateKeyPairs = `
		CREATE TABLE key_pairs (
			id BIGSERIAL NOT NULL PRIMARY KEY,
			twitter_profile_id BIGINT,
			private_key TEXT,
			public_key TEXT
		);
	`

	InitCreateDeviceTokens = `
		CREATE TABLE device_tokens (
			id BIGSERIAL NOT NULL PRIMARY KEY,
			address VARCHAR (45) NOT NULL,
			token TEXT NOT NULL,
			platform VARCHAR(10) NOT NULL  
		);
		CREATE UNIQUE INDEX device_tokens_address_token_platform_key ON device_tokens (address,token,platform);
	`

	InitCreateNotificationEvents = `
		CREATE TABLE notification_events (
			id BIGSERIAL NOT NULL PRIMARY KEY,
			address VARCHAR (45) NOT NULL,
			twitter_profile_id BIGINT,
			message TEXT,
			timestamp TIMESTAMP,
			sender_profile_id BIGINT,
			type BIGINT NOT NULL,
			read BOOL,
			type_id BIGINT
		);
	`
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating table twitter_profiles")
		_, err := db.Exec(InitCreateTwitterProfiles)
		if err != nil {
			return err
		}

		fmt.Println("creating table key_pairs")
		_, err = db.Exec(InitCreateKeyPairs)
		if err != nil {
			return err
		}

		fmt.Println("creating table device_tokens")
		_, err = db.Exec(InitCreateDeviceTokens)
		if err != nil {
			return err
		}

		fmt.Println("creating table notification_events")
		_, err = db.Exec(InitCreateNotificationEvents)
		if err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {

		fmt.Println("dropping notification_events")
		_, err := db.Exec("DROP TABLE notification_events")
		if err != nil {
			return err
		}

		fmt.Println("dropping device_tokens")
		_, err = db.Exec("DROP TABLE device_tokens")
		if err != nil {
			return err
		}

		fmt.Println("dropping key_pairs")
		_, err = db.Exec("DROP TABLE key_pairs")
		if err != nil {
			return err
		}

		fmt.Println("dropping twitter_profiles")
		_, err = db.Exec("DROP TABLE twitter_profiles")
		if err != nil {
			return err
		}

		return nil
	})
}
