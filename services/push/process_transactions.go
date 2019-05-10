package main

import (
	"encoding/json"
	"fmt"

	"strings"

	truchain "github.com/TruStory/truchain/types"
	db "github.com/TruStory/octopus/services/api/db"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/types"
)

func (s *service) processTransactionEvent(pushEvent types.EventDataTx, notifications chan<- *Notification) {
	pushData := &truchain.StakeNotificationResult{}
	err := json.Unmarshal(pushEvent.Result.Data, pushData)
	if err != nil {
		s.log.WithError(err).Error("error decoding transaction event")
		return
	}

	for _, tag := range pushEvent.Result.Tags {
		action := string(tag.Value)
		var alert string
		var enableParticipants bool
		var participantsAlert string
		var hideSender bool
		switch action {
		case "back_story":
			enableParticipants = true
			alert = "Backed your story"
			participantsAlert = "Backed a story you participated in"
			s.checkArgumentMentions(pushData.From.String(), pushData.MsgResult.ID, true)
		case "create_challenge":
			enableParticipants = true
			alert = "Challenged your story"
			participantsAlert = "Challenged a story you participated in"
			s.checkArgumentMentions(pushData.From.String(), pushData.MsgResult.ID, false)
		case "like_backing_argument":
			hideSender = true
			alert = fmt.Sprintf(
				"Someone endorsed your backing argument. You earned %s %s Cred",
				pushData.Cred.Amount.Quo(sdk.NewInt(truchain.Shanev)),
				strings.Title(pushData.Cred.Denom),
			)
		case "like_challenge_argument":
			hideSender = true
			alert = fmt.Sprintf(
				"Someone endorsed your challenge argument. You earned %s %s Cred",
				pushData.Cred.Amount.Quo(sdk.NewInt(truchain.Shanev)),
				strings.Title(pushData.Cred.Denom),
			)
		}
		if alert != "" {
			from := strPtr(pushData.From.String())
			to := pushData.To.String()
			meta := db.NotificationMeta{
				StoryID: &pushData.StoryID,
			}
			if hideSender {
				from = nil
			}
			if pushData.From.String() != to {
				notifications <- &Notification{
					From:   from,
					To:     to,
					Msg:    alert,
					TypeID: pushData.StoryID,
					Type:   db.NotificationStoryAction,
					Meta:   meta,
				}
			}
			if participantsAlert != "" && enableParticipants {
				participants, err := s.getStoryParticipants(pushData.StoryID, to, pushData.From.String())
				if err != nil {
					s.log.WithError(err).Error("unable to get story participants")
				}
				for _, p := range participants {
					notifications <- &Notification{
						From:   from,
						To:     p,
						Msg:    participantsAlert,
						TypeID: pushData.StoryID,
						Type:   db.NotificationStoryAction,
						Meta:   meta,
					}
				}
			}
		}
	}
}
