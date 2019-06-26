package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("indexing argument_id column on comments table...")
		_, err := db.Exec(`CREATE INDEX idx_argument_id_on_comments ON comments(argument_id)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing claim_id column on comments table...")
		_, err = db.Exec(`CREATE INDEX idx_claim_id_on_comments ON comments(claim_id)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing address column on device_tokens table...")
		_, err = db.Exec(`CREATE INDEX idx_address_on_device_tokens ON device_tokens(address)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing story_id column on flagged_stories table...")
		_, err = db.Exec(`CREATE INDEX idx_story_id_on_flagged_stories ON flagged_stories(story_id)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing creator column on invites table...")
		_, err = db.Exec(`CREATE INDEX idx_creator_on_invites ON invites(creator)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing twitter_profile_id column on key_pairs table...")
		_, err = db.Exec(`CREATE INDEX idx_twitter_profile_id_on_key_pairs ON key_pairs(twitter_profile_id)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing address column on notification_events table...")
		_, err = db.Exec(`CREATE INDEX idx_address_on_notification_events ON notification_events(address)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing address column on user_metrics table...")
		_, err = db.Exec(`CREATE INDEX idx_address_on_user_metrics ON user_metrics(address)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing address column on twitter_profiles table...")
		_, err = db.Exec(`CREATE INDEX idx_address_on_twitter_profiles ON twitter_profiles(address)`)
		if err != nil {
			return err
		}
		fmt.Println("indexing username column on twitter_profiles table...")
		_, err = db.Exec(`CREATE INDEX idx_username_on_twitter_profiles ON twitter_profiles(username)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("drop index on argument_id column from comments table...")
		_, err := db.Exec(`DROP INDEX idx_argument_id_on_comments`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on claim_id column from comments table...")
		_, err = db.Exec(`DROP INDEX idx_claim_id_on_comments`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on address column from device_tokens table...")
		_, err = db.Exec(`DROP INDEX idx_address_on_device_tokens`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on story_id column from flagged_stories table...")
		_, err = db.Exec(`DROP INDEX idx_story_id_on_flagged_stories`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on creator column from invites table...")
		_, err = db.Exec(`DROP INDEX idx_creator_on_invites`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on twitter_profile_id column from key_pairs table...")
		_, err = db.Exec(`DROP INDEX idx_twitter_profile_id_on_key_pairs`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on address column from notification_events table...")
		_, err = db.Exec(`DROP INDEX idx_address_on_notification_events`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on address column from user_metrics table...")
		_, err = db.Exec(`DROP INDEX idx_address_on_user_metrics`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on address column from twitter_profiles table...")
		_, err = db.Exec(`DROP INDEX idx_address_on_twitter_profiles`)
		if err != nil {
			return err
		}
		fmt.Println("drop index on username column from twitter_profiles table...")
		_, err = db.Exec(`DROP INDEX idx_username_on_twitter_profiles`)
		return err
	})
}
