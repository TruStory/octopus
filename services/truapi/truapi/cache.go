package truapi

import (
	"context"
	"encoding/json"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/truchain/x/claim"
)

func (ta *TruAPI) cacheFeed(ctx context.Context, q queryByCommunityIDAndFeedFilter, claims []claim.Claim) {
	id, err := json.Marshal(q)
	if err != nil {
		return
	}
	feed, err := json.Marshal(claims)
	if err != nil {
		return
	}
	cachedFeed := &db.CachedFeed{
		ID:   string(id),
		Feed: string(feed),
	}
	_ = ta.DBClient.AddCachedFeed(cachedFeed)
}

func (ta *TruAPI) getCachedFeed(ctx context.Context, q queryByCommunityIDAndFeedFilter) (*[]claim.Claim, error) {
	id, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	feed, err := ta.DBClient.CachedFeedByID(string(id))
	if err != nil {
		return nil, err
	}
	claims := make([]claim.Claim, 0)
	err = json.Unmarshal([]byte(feed), &claims)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}
