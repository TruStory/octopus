package truapi

import (
	"context"
	"sort"

	"github.com/TruStory/truchain/x/claim"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// claimMetricsTrending represents claim metrics within last 24 hours
type claimMetricsTrending struct {
	Claim claim.Claim
	Score float64
}

func (ta *TruAPI) filterFeedClaims(ctx context.Context, claims []claim.Claim, filter FeedFilter) []claim.Claim {
	if filter == Latest {
		// Reverse chronological order
		sort.Slice(claims, func(i, j int) bool {
			return claims[j].CreatedTime.Before(claims[i].CreatedTime)
		})
		return claims
	} else if filter == Best {
		// Total amount staked
		// Total stakers
		// Total comments
		// Smallest delta between Backing vs Challenge stake
		metrics := make([]claimMetricsBest, 0)
		for _, claim := range claims {
			totalAmountStaked := claim.TotalBacked.Add(claim.TotalChallenged).Amount
			totalStakers := claim.TotalStakers
			comments, _ := ta.DBClient.CommentsByClaimID(claim.ID)
			totalComments := len(comments)
			var backingChallengeDelta sdk.Int
			if claim.TotalBacked.IsGTE(claim.TotalChallenged) {
				backingChallengeDelta = claim.TotalBacked.Sub(claim.TotalChallenged).Amount
			} else {
				backingChallengeDelta = claim.TotalChallenged.Sub(claim.TotalBacked).Amount
			}
			metric := claimMetricsBest{
				Claim:                 claim,
				TotalAmountStaked:     totalAmountStaked,
				TotalStakers:          totalStakers,
				TotalComments:         totalComments,
				BackingChallengeDelta: backingChallengeDelta,
			}
			metrics = append(metrics, metric)
		}
		sort.Slice(metrics, func(i, j int) bool {
			if metrics[i].TotalAmountStaked.GT(metrics[j].TotalAmountStaked) {
				return true
			}
			if metrics[i].TotalAmountStaked.LT(metrics[j].TotalAmountStaked) {
				return false
			}
			if metrics[i].TotalStakers > metrics[j].TotalStakers {
				return true
			}
			if metrics[i].TotalStakers < metrics[j].TotalStakers {
				return false
			}
			if metrics[i].TotalComments > metrics[j].TotalComments {
				return true
			}
			if metrics[i].TotalComments < metrics[j].TotalComments {
				return false
			}
			if metrics[i].BackingChallengeDelta.GT(metrics[j].BackingChallengeDelta) {
				return true
			}
			return false
		})
		bestClaims := make([]claim.Claim, 0)
		for _, metric := range metrics {
			bestClaims = append(bestClaims, metric.Claim)
		}
		return bestClaims
	} else if filter == Trending {
		metrics := make([]claimMetricsTrending, 0)
		for _, claim := range claims {

			metric := claimMetricsTrending{
				Claim: claim,
				Score: ta.claimTrendingScore(ctx, claim),
			}
			metrics = append(metrics, metric)
		}
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Score > metrics[j].Score
		})
		trendingClaims := make([]claim.Claim, 0)
		for _, metric := range metrics {
			trendingClaims = append(trendingClaims, metric.Claim)
		}
		return trendingClaims
	}
	return claims
}
