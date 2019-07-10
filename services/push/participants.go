package main

import (
	"context"

	"github.com/machinebox/graphql"
)

type claimParticipants struct {
	ClaimID      int64
	Creator      string
	Participants []string
}

func (s *service) getArgumentSummary(argumentId int64) (*ArgumentSummaryResponse, error) {
	req := graphql.NewRequest(argumentSummaryByIDQuery)
	req.Var("argumentId", argumentId)
	res := &ArgumentSummaryResponse{}
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, req, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *service) getClaimParticipantsByArgumentId(argumentId int64) (claimParticipants, error) {
	req := graphql.NewRequest(ClaimArgumentByIDQuery)
	req.Var("argumentId", argumentId)
	res := &ClaimArgumentResponse{}
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, req, &res); err != nil {
		return claimParticipants{}, err
	}
	mappedParticipants := make(map[string]bool)
	for _, p := range res.ClaimArgument.Claim.Participants {
		mappedParticipants[p.Address] = true
	}
	participants := make([]string, 0, len(res.ClaimArgument.Claim.Participants))
	for p := range mappedParticipants {
		if p == res.ClaimArgument.Claim.Creator.Address {
			continue
		}
		participants = append(participants, p)
	}
	return claimParticipants{
		Creator:      res.ClaimArgument.Claim.Creator.Address,
		ClaimID:      res.ClaimArgument.ClaimID,
		Participants: participants,
	}, nil
}
