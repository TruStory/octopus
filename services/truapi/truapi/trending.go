package truapi

import (
	"context"
	"math"
	"time"

	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	firstClaimCreatedTime = time.Date(2019, 4, 13, 4, 4, 0, 0, time.UTC)
)

func (ta *TruAPI) claimTrendingScore(ctx context.Context, claim claim.Claim) float64 {
	totalStaked := claim.TotalBacked.Add(claim.TotalChallenged).Amount.Quo(sdk.NewInt(app.Shanev)).Int64()
	return ta.trendingScore(ctx, totalStaked, claim.FirstArgumentTime)
}

// argument trending score based on hacker news implementation
// https://medium.com/hacking-and-gonzo/how-hacker-news-ranking-algorithm-works-1d9b0cf2c08d
func (ta *TruAPI) argumentTrendingScore(ctx context.Context, argument staking.Argument) float64 {
	return float64(argument.UpvotedCount) / math.Pow(time.Since(argument.CreatedTime).Hours()+2, 1.8)
}

// trendingScore calculates a claims trending score mostly based off:
// https://medium.com/hacking-and-gonzo/how-reddit-ranking-algorithms-work-ef111e33d0d9
func (ta *TruAPI) trendingScore(ctx context.Context, totalStaked int64, createdTime time.Time) float64 {
	// weigh stake amount using logarithmic scale
	// in order to double importance need n^2 increase in stake
	// i.e. first 20 staked adds as much weight as next 400 staked, adds as much weight as next 8000 staked
	// this will weigh more recent claims with a single argument higher than an older claims with several arguments
	order := math.Log2(float64(totalStaked))

	// difference in seconds between when first claim was posted and this claim was posted
	// first claim was posted on 2019-04-13T04:04:17.285493139Z
	seconds := createdTime.Sub(firstClaimCreatedTime).Seconds()

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
