package main

import (
	"context"
	"fmt"

	db "github.com/TruStory/truchain/x/db"
	"github.com/machinebox/graphql"
	stripmd "github.com/writeas/go-strip-markdown"
)

func (s *service) checkArgumentMentions(from string, stakeID int64, backing bool) {
	s.argumentMentionsCh <- argumentMention{from, stakeID, backing}
}

type argumentMention struct {
	From    string
	StakeID int64
	Backing bool
}

func (s *service) mentionChecker(notifications chan<- *Notification, stop <-chan struct{}) {
	for {
		select {
		case argumentMention := <-s.argumentMentionsCh:
			argument, err := s.getArgument(argumentMention.StakeID, argumentMention.Backing)
			if err != nil {
				s.log.WithError(err).Errorf("error getting argument for stakeId[%d] backing[%t]",
					argumentMention.StakeID,
					argumentMention.Backing)
			}
			parsedBody, addresses := s.parseCosmosMentions(argument.Body)
			parsedBody = stripmd.Strip(parsedBody)
			mentionType := db.MentionArgument
			addresses = unique(addresses)
			for _, address := range addresses {
				notifications <- &Notification{
					From:   &argumentMention.From,
					To:     address,
					Msg:    parsedBody,
					TypeID: argument.ID,
					Type:   db.NotificationMentionAction,
					Meta: db.NotificationMeta{
						ArgumentID:  &argument.ID,
						StoryID:     &argument.StoryID,
						MentionType: &mentionType,
					},
					Action: "Mentioned you in an argument",
					Trim:   true,
				}
			}
		case <-stop:
			s.log.Info("stopping mention checker")
			return

		}
	}
}

func (s *service) getChallengeArgument(req *graphql.Request) (*Argument, error) {
	res := ChallengeResponse{}
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, req, &res); err != nil {
		return nil, err
	}
	return &res.ChallengeArgument.Argument, nil
}

func (s *service) getBackingArgument(req *graphql.Request) (*Argument, error) {
	res := BackingResponse{}
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, req, &res); err != nil {
		return nil, err
	}
	return &res.BackingArgument.Argument, nil
}

func (s *service) getArgument(stakeID int64, backing bool) (*Argument, error) {
	queryType := "challenge"
	if backing {
		queryType = "backing"
	}
	query := fmt.Sprintf(ArgumentByStakeIDQuery, queryType)
	req := graphql.NewRequest(query)
	req.Var("id", stakeID)
	if backing {
		return s.getBackingArgument(req)
	}
	return s.getChallengeArgument(req)
}
