package main

import (
	"encoding/hex"
	"fmt"

	truDB "github.com/TruStory/octopus/services/truapi/db"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("setting new hex tokens in the users table...")
		var ids []int
		err := db.Model((*truDB.User)(nil)).Column("id").Select(&ids)
		if err != nil {
			return err
		}
		for _, id := range ids {
			token, err := generateCryptoSafeRandomBytes(32)
			if err != nil {
				return err
			}
			_, err = db.Exec(`UPDATE users 
				SET token = ?
				WHERE id = ?`, hex.EncodeToString(token), id)
			if err != nil {
				return err
			}
		}

		fmt.Println("setting new hex tokens in the password reset tokens...")
		var prts []truDB.PasswordResetToken
		err = db.Model(&prts).Order("id ASC").Select()
		if err != nil {
			return err
		}
		for _, prt := range prts {
			token, err := generateCryptoSafeRandomBytes(32)
			if err != nil {
				return err
			}
			prt.Token = hex.EncodeToString(token)
			err = db.Update(&prt)
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db migrations.DB) error {
		// no going back!
		return nil
	})
}
