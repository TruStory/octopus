package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/machinebox/graphql"

	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/octopus/services/truapi/truapi"
)

const FEATURED_DEBATE_COMMUNITY_ID = "all"

func (s *service) processBroadcastNotifications(bNotifications <-chan *app.BroadcastNotificationRequest, notifications chan<- *Notification) {
	for n := range bNotifications {
		featuredClaimID, err := s.db.ClaimOfTheDayIDByCommunityID(FEATURED_DEBATE_COMMUNITY_ID)
		if err != nil {
			s.log.WithError(err).Errorf("could not retrieve featured claim for community [%s]\n", FEATURED_DEBATE_COMMUNITY_ID)
			continue
		}
		featuredClaim, err := s.getClaim(featuredClaimID)
		if err != nil {
			s.log.WithError(err).Errorf("could not claim for id [%d]\n", featuredClaimID)
			continue
		}
		users := make([]db.User, 0)
		err = s.db.FindAll(&users)
		if err != nil {
			s.log.WithError(err).Errorf("could not retrieve users for type [%d]\n", n.Type)
			continue
		}

		if !strings.HasSuffix(featuredClaim.Claim.Body, ".") {
			featuredClaim.Claim.Body = featuredClaim.Claim.Body + "."
		}

		for _, user := range users {
			notifications <- &Notification{
				To:     user.Address,
				TypeID: featuredClaim.Claim.ID,
				Type:   db.NotificationFeaturedDebate,
				Msg:    fmt.Sprintf("New Featured Debate: %s Join the debate and share your thoughts!", featuredClaim.Claim.Body),
				Meta: db.NotificationMeta{
					ClaimID: &featuredClaim.Claim.ID,
				},
				Action: "Featured Debate",
				Trim:   true,
			}
		}
	}
}

func (s *service) getClaim(claimID int64) (ClaimResponse, error) {
	graphqlReq := graphql.NewRequest(claimByIDQuery)

	graphqlReq.Var("claimId", claimID)
	var graphqlRes ClaimResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return graphqlRes, err
	}

	return graphqlRes, nil
}
