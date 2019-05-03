package truapi

import (
	"context"
	"sort"
	"time"

	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/story"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TrendingMetrics holds metrics used to sort the trending feed
type TrendingMetrics struct {
	Story        story.Story
	Participants int
	TotalStake   int64
	Status       story.Status
	ExpireTime   time.Time
}

// filterByLatest filters out stories that are expired or have at least one participant
func (ta *TruAPI) filterByLatest(ctx context.Context, feedStories []story.Story) ([]story.Story, error) {
	filteredStories := make([]story.Story, 0)

	for _, feedStory := range feedStories {
		// only include pending stories
		if feedStory.Status == story.Pending {
			// which have no backers
			backingPool := ta.backingPoolResolver(ctx, feedStory)
			if backingPool.Amount.IsZero() {
				// and no challengers
				challengePool := ta.challengePoolResolver(ctx, feedStory)
				if challengePool.Amount.IsZero() {
					// passed all criteria to appear in latest list
					filteredStories = append(filteredStories, feedStory)
				}
			}
		}
	}
	return filteredStories, nil
}

// filterByCompleted filters out stories that are not yet completed
func (ta *TruAPI) filterByCompleted(ctx context.Context, feedStories []story.Story) ([]story.Story, error) {
	filteredStories := make([]story.Story, 0)

	for _, feedStory := range feedStories {
		// only include stories that have expired
		if feedStory.Status == story.Expired {
			filteredStories = append(filteredStories, feedStory)
		}
	}
	return filteredStories, nil
}

// filterByTrending orders the stories according to story state, stake, participants and expire time
func (ta *TruAPI) filterByTrending(ctx context.Context, feedStories []story.Story) ([]story.Story, error) {
	trendingMetrics := make([]TrendingMetrics, 0)

	// for better performance, we first fetch metrics for all stories from the KV store
	// then we execute a sort using those metrics
	for _, feedStory := range feedStories {
		backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: feedStory.ID})
		challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: feedStory.ID})
		participants := len(backings) + len(challenges)
		totalBacking := sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
		for i := range backings {
			totalBacking = totalBacking.Plus(backings[i].Amount())
		}
		totalChallenge := sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
		for j := range challenges {
			totalChallenge = totalChallenge.Plus(challenges[j].Amount())
		}
		totalStake := totalBacking.Plus(totalChallenge).Amount.Int64()

		metrics := TrendingMetrics{
			Story:        feedStory,
			Participants: participants,
			TotalStake:   totalStake,
			Status:       feedStory.Status,
			ExpireTime:   feedStory.ExpireTime,
		}
		trendingMetrics = append(trendingMetrics, metrics)
	}

	// Now we sort all the stories by story state, stake, participants and expire time
	sort.Slice(trendingMetrics, func(i, j int) bool {
		if trendingMetrics[i].Status == story.Pending && trendingMetrics[j].Status == story.Expired {
			return true
		}
		if trendingMetrics[i].Status == story.Expired && trendingMetrics[j].Status == story.Pending {
			return false
		}
		if trendingMetrics[i].TotalStake > trendingMetrics[j].TotalStake {
			return true
		}
		if trendingMetrics[i].TotalStake < trendingMetrics[j].TotalStake {
			return false
		}
		if trendingMetrics[i].Participants > trendingMetrics[j].Participants {
			return true
		}
		if trendingMetrics[i].Participants < trendingMetrics[j].Participants {
			return false
		}
		return trendingMetrics[j].ExpireTime.Before(trendingMetrics[i].ExpireTime)
	})

	orderedStories := make([]story.Story, 0)
	for _, trendingMetric := range trendingMetrics {
		orderedStories = append(orderedStories, trendingMetric.Story)
	}
	return orderedStories, nil
}
