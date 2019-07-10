package main

import (
	"fmt"

	"github.com/TruStory/octopus/services/truapi/db"

	"github.com/TruStory/truchain/x/staking"
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
	notified := make(map[string]bool)

	// check mentions first
	_, addresses := s.parseCosmosMentions(argument.Body)
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
			Msg:    fmt.Sprintf("added a new argument on a claim you created: %s", argument.Summary),
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
			Msg:    fmt.Sprintf("added a new argument on a claim you participated: %s", argument.Summary),
			TypeID: int64(argument.ID),
			Type:   db.NotificationNewArgument,
			Meta:   meta,
			Action: "New Argument",
		}
	}
}

func (s *service) processUpvote(data []byte, notifications chan<- *Notification) {
	stake := staking.Stake{}
	err := staking.ModuleCodec.UnmarshalJSON(data, &stake)
	if err != nil {
		s.log.WithError(err).Error("error decoding argument created event")
		return
	}
	argument, err := s.getArgumentSummary(int64(stake.ArgumentID))
	if err != nil {
		s.log.WithError(err).Error("error getting participants ")
		return
	}
	meta := db.NotificationMeta{
		ClaimID:    &argument.ClaimArgument.ClaimID,
		ArgumentID: uint64Ptr(stake.ArgumentID),
	}

	argumentCreatorAddress := argument.ClaimArgument.Creator.Address
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
