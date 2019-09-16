package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding community_id column to comments...")
		_, err := db.Exec(`ALTER TABLE comments ADD COLUMN community_id VARCHAR(75)`)
		if err != nil {
			return err
		}
		httpCli := &http.Client{
			Timeout: time.Second * 30,
		}
		date := time.Now().Add(time.Hour * 24)
		url := fmt.Sprintf("https://beta.trustory.io/api/v1/metrics/claims?date=%s", date.Format("2006-01-02"))
		fmt.Println("Fetching claims from", url)
		resp, err := httpCli.Get(url)
		if err != nil {
			return err
		}
		claims, err := csv.NewReader(resp.Body).ReadAll()
		if err != nil {
			return err
		}
		for _, claim := range claims[1:] {
			if len(claim) != 27 {
				return fmt.Errorf("malformed data")
			}
			_, err := db.Exec(fmt.Sprintf("UPDATE comments SET community_id = '%s' where claim_id=%s", claim[5], claim[4]))
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("removing community_id column from comments...")
		_, err := db.Exec(`ALTER TABLE comments DROP COLUMN community_id`)
		return err
	})
}
