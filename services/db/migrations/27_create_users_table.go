package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating users table...")
		_, err := db.Exec(`CREATE TABLE users(
			id BIGSERIAL PRIMARY KEY,
			first_name VARCHAR(128) NOT NULL,
			last_name VARCHAR(128) NOT NULL,
			email VARCHAR(128) UNIQUE NOT NULL,
			username VARCHAR(128) DEFAULT NULL,
			address VARCHAR(65) DEFAULT NULL,
			password VARCHAR(256) DEFAULT NULL,
			invited_by VARCHAR(65) DEFAULT NULL,
			request_token VARCHAR(65) DEFAULT NULL,
			approved_at TIMESTAMP DEFAULT NULL,
			rejected_at TIMESTAMP DEFAULT NULL,
			signedup_at TIMESTAMP DEFAULT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping users table...")
		_, err := db.Exec(`DROP TABLE users`)
		return err
	})
}
