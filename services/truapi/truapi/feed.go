package truapi

import (
	"context"
	"math"
	"sort"
	"time"

	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/claim"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// claimMetricsTrending represents claim metrics within last 24 hours
type claimMetricsTrending struct {
	Claim claim.Claim
	Score float64
}

var (
	firstClaimCreatedTime = time.Date(2019, 4, 13, 4, 4, 0, 0, time.UTC)
)

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
			totalComments := len(ta.claimCommentsResolver(ctx, queryByClaimID{ID: claim.ID}))
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
				Score: ta.trendingScore(ctx, claim),
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

// trendingScore calculates a claims trending score mostly based off:
// https://medium.com/hacking-and-gonzo/how-reddit-ranking-algorithms-work-ef111e33d0d9
func (ta *TruAPI) trendingScore(ctx context.Context, claim claim.Claim) float64 {
	// total amount staked
	totalStaked := claim.TotalBacked.Add(claim.TotalChallenged).Amount.Quo(sdk.NewInt(app.Shanev)).Int64()

	// weigh stake amount using logarithmic scale
	// in order to double importance need n^2 increase in stake
	// i.e. first 20 staked adds as much weight as next 400 staked, adds as much weight as next 8000 staked
	// this will weigh more recent claims with a single argument higher than an older claims with several arguments
	order := math.Log2(float64(totalStaked))

	// difference in seconds between when first claim was posted and this claim was posted
	// first claim was posted on 2019-04-13T04:04:17.285493139Z
	seconds := claim.CreatedTime.Sub(firstClaimCreatedTime).Seconds()

	// every timeDecay elapsed since a claim was created it loses 1 score point
	// first 10 tru staked gives ~3.3 score points
	// first 100 tru staked gives ~6.6 score points
	// first 1000 tru staked gives ~9.9 score points

	// increasing time decay causes claims with more stakes to be preferred over newly created claims
	// default timeDecay is 45000 seconds which means that a claim created right now will have 1 score point
	// more than a claim created 12.5 hours ago, 2 more score points than a claim created 25 hours ago etc
	timeDecay := ta.APIContext.Config.Params.TrendingFeedTimeDecay

	score := order + seconds/float64(timeDecay)

	return score
}
