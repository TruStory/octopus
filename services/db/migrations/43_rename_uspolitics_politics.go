// track_events table
// followed_communities table

package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("rename uspolitics to politics in followed_communities table")
		_, err := db.Exec(`UPDATE followed_communities
			SET community_id = 'politics'
			WHERE community_id = 'uspolitics'`)
		if err != nil {
			return err
		}

		fmt.Println("rename uspolitics to politics in track_events table")
		_, err = db.Exec(`UPDATE track_events
			SET meta = meta || '{"communityId":"politics"}'
			WHERE meta @> '{"communityId":"uspolitics"}'`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("rename politics to uspolitics in followed_communities table")
		_, err := db.Exec(`UPDATE followed_communities
			SET community_id = 'uspolitics'
			WHERE community_id = 'politics'`)
		if err != nil {
			return err
		}

		fmt.Println("rename politics to uspolitics in track_events table")
		_, err = db.Exec(`UPDATE track_events
			SET meta = meta || '{"communityId":"uspolitics"}'
			WHERE meta @> '{"communityId":"politics"}'`)
		return err
	})
}
