package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/octopus/services/truapi/truapi"
)

func (s *service) processRewardsNotifications(rNotifications <-chan *app.RewardNotificationRequest, notifications chan<- *Notification) {
	for n := range rNotifications {
		s.log.Infoln("processing a reward notification", n)
		user, err := s.db.UserByID(n.RewardeeID)
		if err != nil {
			s.log.WithError(err).Errorf("could not retrieve rewardee for id [%d]\n", n.RewardeeID)
			continue
		}

		var causer *db.User
		if n.CauserID != 0 {
			causer, err = s.db.UserByID(n.CauserID)
			if err != nil {
				s.log.WithError(err).Errorf("could not retrieve causer for id [%d]\n", n.CauserID)
				continue
			}
		}

		nType, ok := getNotificationTypeFromRequest(*n)
		if !ok {
			s.log.Warn("Unknown reward type")
			continue
		}
		notifications <- &Notification{
			To:     user.Address,
			TypeID: 0,
			Type:   nType,
			Msg:    fmt.Sprintf("You were rewarded with %s because %s", getRewardStringFromRequest(*n), getRewardReasonFromRequest(*n, causer)),
			Meta: db.NotificationMeta{
				RewardCauserID: &n.CauserID,
			},
			Action: "Reward unlocked",
			Trim:   true,
		}
	}
}

func getNotificationTypeFromRequest(n app.RewardNotificationRequest) (db.NotificationType, bool) {
	switch n.RewardType {
	case app.RewardTypeInvite:
		return db.NotificationRewardInviteUnlocked, true
	case app.RewardTypeTru:
		return db.NotificationRewardTruUnlocked, true
	}

	return 0, false
}

func getRewardStringFromRequest(n app.RewardNotificationRequest) string {
	switch n.RewardType {
	case app.RewardTypeInvite:
		return fmt.Sprintf("%s invites", n.RewardAmount)
	case app.RewardTypeTru:
		amount, err := sdk.ParseCoin(n.RewardAmount)
		if err != nil {
			return n.RewardAmount
		}
		return fmt.Sprintf("%s %s", humanReadable(amount), db.CoinDisplayName)
	}

	return ""
}

func getRewardReasonFromRequest(n app.RewardNotificationRequest, causer *db.User) string {
	switch n.RewardType {
	case app.RewardTypeInvite:
		reason := "%s became an active user on TruStory."
		causedBy := "you"
		if causer != nil {
			causedBy = causer.Username
		}
		return fmt.Sprintf(reason, causedBy)

	case app.RewardTypeTru:
		reason := "%s %s on TruStory."
		stepCompleted := ""
		switch n.CauserAction {
		case app.RewardCauserActionSignedUp:
			stepCompleted = "signed up"
		case app.RewardCauserActionOneArgument:
			stepCompleted = "has written at least one argument"
		case app.RewardCauserActionReceiveFiveAgrees:
			stepCompleted = "has received at least five agrees"
		}
		return fmt.Sprintf(reason, causer.Username, stepCompleted)
	}

	return ""
}
