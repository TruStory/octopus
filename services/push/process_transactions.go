package main

import (
	"encoding/json"
	"fmt"

	"strings"

	"github.com/TruStory/octopus/services/truapi/db"
	truchain "github.com/TruStory/truchain/types"
	stripmd "github.com/writeas/go-strip-markdown"

	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/types"
)

func (s *service) processArgumentCreated(data []byte, notifications chan<- *Notification) {
	argument := staking.Argument{}
	err := staking.ModuleCodec.UnmarshalJSON(data, &argument)
	if err != nil {
		s.log.WithError(err).Error("error decoding argument created event")
		return
	}
	claimParticipants, err := s.getClaimParticipantsByArgumentId(int64(argument.ID))
	if err != nil {
		s.log.WithError(err).Error("error getting participants ")
		return
	}
	meta := db.NotificationMeta{
		ClaimID:    &claimParticipants.ClaimID,
		ArgumentID: uint64Ptr(argument.ID),
	}

	creatorAddress := argument.Creator.String()
	notified := make(map[string]bool, 0)

	// check mentions first
	parsedBody, addresses := s.parseCosmosMentions(argument.Body)
	parsedBody = stripmd.Strip(parsedBody)
	mentionType := db.MentionArgument
	addresses = unique(addresses)
	for _, address := range addresses {
		notified[address] = true
		notifications <- &Notification{
			From:   &creatorAddress,
			To:     address,
			Msg:    fmt.Sprintf("mentioned you %s: %s", mentionType.String(), argument.Summary),
			TypeID: int64(argument.ID),
			Type:   db.NotificationMentionAction,
			Meta: db.NotificationMeta{
				ClaimID:     &claimParticipants.ClaimID,
				ArgumentID:  uint64Ptr(argument.ID),
				MentionType: &mentionType,
			},
			Action: "Mentioned you in an argument",
			Trim:   true,
		}
	}

	if _, ok := notified[creatorAddress]; creatorAddress != claimParticipants.Creator && !ok {
		notified[creatorAddress] = true
		notifications <- &Notification{
			From:   strPtr(argument.Creator.String()),
			To:     claimParticipants.Creator,
			Msg:    fmt.Sprintf("added a new argument in a claim you created: %s", argument.Summary),
			TypeID: int64(argument.ID),
			Type:   db.NotificationNewArgument,
			Meta:   meta,
			Action: "New Argument",
		}
	}

	for _, p := range claimParticipants.Participants {
		if _, ok := notified[p]; ok {
			continue
		}
		notified[p] = true
		notifications <- &Notification{
			From:   strPtr(argument.Creator.String()),
			To:     p,
			Msg:    fmt.Sprintf("added a new argument in a claim you participated: %s", argument.Summary),
			TypeID: int64(argument.ID),
			Type:   db.NotificationNewArgument,
			Meta:   meta,
			Action: "New Argument",
		}
	}
}

func (s *service) processUpvote(data []byte, notifications chan<- *Notification) {
	fmt.Println("processing upvote")
	stake := staking.Stake{}
	err := staking.ModuleCodec.UnmarshalJSON(data, &stake)
	if err != nil {
		s.log.WithError(err).Error("error decoding argument created event")
		return
	}
	argument, err := s.getArgumentSummary(int64(stake.ArgumentID))
	fmt.Println("argument summary", argument)
	if err != nil {
		s.log.WithError(err).Error("error getting participants ")
		return
	}
	meta := db.NotificationMeta{
		ClaimID:    &argument.ClaimArgument.ClaimID,
		ArgumentID: uint64Ptr(stake.ArgumentID),
	}

	argumentCreatorAddress := argument.ClaimArgument.Creator.Address
	fmt.Println("sending notification from ", stake.Creator.String(), argumentCreatorAddress)
	notifications <- &Notification{
		From:   strPtr(stake.Creator.String()),
		To:     argumentCreatorAddress,
		Msg:    fmt.Sprintf("agreed with your argument: %s", argument.ClaimArgument.Summary),
		TypeID: int64(stake.ArgumentID),
		Type:   db.NotificationAgreeReceived,
		Meta:   meta,
		Action: "Agree Received",
	}
}

func (s *service) processTxEvent(evt types.EventDataTx, notifications chan<- *Notification) {
	for _, tag := range evt.Result.Tags {
		action := string(tag.Value)
		switch action {
		case "create-argument":
			s.processArgumentCreated(evt.Result.Data, notifications)
		case "create-upvote":
			s.processUpvote(evt.Result.Data, notifications)
		}
	}
}

// deprecated
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
