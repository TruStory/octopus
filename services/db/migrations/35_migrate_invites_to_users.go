package main

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-pg/pg"

	truDB "github.com/TruStory/octopus/services/truapi/db"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("moving invites to the users table...")
		var invites []truDB.Invite
		var users []truDB.User
		err := db.Model(&invites).Order("id ASC").Select()
		if err != nil {
			return err
		}
		for _, invite := range invites {
			var user truDB.User
			err = db.Model(&user).Where("email = ?", invite.FriendEmail).First()
			// if the invited user has not signed up yet
			if err == pg.ErrNoRows {
				var referrer truDB.User
				err = db.Model(&referrer).Where("address = ?", invite.Creator).First()
				if err != nil {
					return err
				}
				token, err := generateCryptoSafeRandomBytes(32)
				if err != nil {
					return err
				}
				user = truDB.User{
					FullName:   truDB.InvitedUserDefaultName,
					Email:      invite.FriendEmail,
					ReferredBy: referrer.ID,
					Token:      base64.StdEncoding.EncodeToString(token),
					ApprovedAt: time.Now(),
				}
				_, err = db.Model(&user).Insert()
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}

		err = db.Model(&users).Order("id ASC").Select()
		if err != nil {
			return err
		}
		for _, user := range users {
			token, err := generateCryptoSafeRandomBytes(32)
			if err != nil {
				return err
			}
			user.Token = base64.StdEncoding.EncodeToString(token)
			err = db.Update(&user)
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("removing invited users from the users table...")
		_, err := db.Model((*truDB.User)(nil)).Where("full_name = ?", truDB.InvitedUserDefaultName).Where("referred_by is not null").Delete()
		return err
	})
}
