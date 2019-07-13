package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("updating track_events")
		_, err := db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"crypto"}') WHERE (meta->>'categoryId')::INTEGER = 1`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"trustory"}') WHERE (meta->>'categoryId')::INTEGER = 2`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"general"}') WHERE (meta->>'categoryId')::INTEGER = 3`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"cosmos"}') WHERE (meta->>'categoryId')::INTEGER = 4`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"entertainment"}') WHERE (meta->>'categoryId')::INTEGER = 5`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"tech"}') WHERE (meta->>'categoryId')::INTEGER = 6`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`UPDATE track_events SET meta = (meta - 'categoryId'  || '{"communityId":"sports"}') WHERE (meta->>'categoryId')::INTEGER = 7`)
		return err
	}, func(db migrations.DB) error {
		return nil
	})
}
