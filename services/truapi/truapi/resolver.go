package truapi

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/argument"
	"github.com/TruStory/truchain/x/backing"
	"github.com/TruStory/truchain/x/category"
	"github.com/TruStory/truchain/x/challenge"
	"github.com/TruStory/truchain/x/params"
	"github.com/TruStory/truchain/x/story"
	trubank "github.com/TruStory/truchain/x/trubank"
	"github.com/TruStory/truchain/x/users"
	sdk "github.com/cosmos/cosmos-sdk/types"
	amino "github.com/tendermint/go-amino"
)

// FeedFilter is parameter for filtering the story feed
type FeedFilter int64

// List of filter types
const (
	None FeedFilter = iota
	Trending
	Latest
	Completed
	Best
)

// QueryByCategoryIDAndFeedFilter is query params for filtering a story feed by category and FeedFilter
type QueryByCategoryIDAndFeedFilter struct {
	CategoryID int64
	FeedFilter FeedFilter `graphql:",optional"`
}

// UserMetricsFilter is query params for filtering the statistia metrics
type UserMetricsFilter struct {
	From string
	To   string
}

func (ta *TruAPI) allCategoriesResolver(ctx context.Context, q struct{}) []category.Category {
	categoryBlacklist := []string{"sports", "tech", "entertainment"}

	res, err := ta.RunQuery("categories/all", struct{}{})
	if err != nil {
		fmt.Println("allCategoriesResolver err: ", err)
		return []category.Category{}
	}

	cs := new([]category.Category)
	err = json.Unmarshal(res, cs)
	if err != nil {
		panic(err)
	}

	// sort in alphabetical order
	sort.Slice(*cs, func(i, j int) bool {
		return (*cs)[j].Title > (*cs)[i].Title
	})

	// exclude blacklisted categories
	filteredCategories := make([]category.Category, 0)
	for _, c := range *cs {
		if !contains(categoryBlacklist, c.Slug) {
			filteredCategories = append(filteredCategories, c)
		}
	}

	return filteredCategories
}

func (ta *TruAPI) storiesResolver(ctx context.Context, q QueryByCategoryIDAndFeedFilter) []story.Story {
	var res []byte
	var err error
	if q.CategoryID == -1 {
		res, err = ta.RunQuery("stories/all", struct{}{})
	} else {
		res, err = ta.RunQuery("stories/category", story.QueryCategoryStoriesParams{CategoryID: q.CategoryID})
	}
	if err != nil {
		fmt.Println("storiesResolver err: ", err)
		return []story.Story{}
	}

	stories := new([]story.Story)
	err = json.Unmarshal(res, stories)
	if err != nil {
		panic(err)
	}

	unflaggedStories, err := ta.filterFlaggedStories(stories)
	if err != nil {
		fmt.Println("filterFlaggedStories err: ", err)
		panic(err)
	}

	filteredStories, err := ta.filterFeedStories(ctx, unflaggedStories, q.FeedFilter)
	if err != nil {
		fmt.Println("filterFeedStories err: ", err)
		panic(err)
	}

	return filteredStories
}

func (ta *TruAPI) argumentResolver(_ context.Context, q app.QueryArgumentByID) argument.Argument {
	res, err := ta.RunQuery(
		path.Join(argument.QueryPath, argument.QueryArgumentByID),
		app.QueryByIDParams{ID: q.ID},
	)
	if err != nil {
		fmt.Println("argumentResolver err: ", err)
		return argument.Argument{}
	}

	argument := new(argument.Argument)
	err = json.Unmarshal(res, argument)
	if err != nil {
		panic(err)
	}
	// check if raw argument was passed
	if q.Raw {
		return *argument
	}

	body, err := ta.DBClient.TranslateToUsersMentions(argument.Body)
	if err != nil {
		panic(err)
	}
	argument.Body = body

	return *argument
}

func (ta *TruAPI) likesObjectResolver(_ context.Context, q app.QueryByIDParams) []argument.Like {
	query := path.Join(argument.QueryPath, argument.QueryLikesByArgumentID)
	res, err := ta.RunQuery(query, app.QueryByIDParams{ID: q.ID})
	if err != nil {
		fmt.Println("likesObjectResolver err: ", err)
		return []argument.Like{}
	}

	likes := new([]argument.Like)
	err = json.Unmarshal(res, likes)
	if err != nil {
		panic(err)
	}

	return *likes
}

func (ta *TruAPI) backingResolver(
	_ context.Context, q app.QueryByIDParams) backing.Backing {
	res, err := ta.RunQuery("backings/id", app.QueryByIDParams{ID: q.ID})
	if err != nil {
		fmt.Println("error getting backing", err)
		return backing.Backing{}
	}
	backing := backing.Backing{}
	err = json.Unmarshal(res, &backing)
	if err != nil {
		panic(err)
	}

	return backing
}

func (ta *TruAPI) backingsResolver(_ context.Context, q app.QueryByIDParams) []backing.Backing {
	res, err := ta.RunQuery("backings/storyID", q)
	if err != nil {
		fmt.Println("backingsResolver err: ", err)
		return []backing.Backing{}
	}

	backings := new([]backing.Backing)
	err = json.Unmarshal(res, backings)
	if err != nil {
		panic(err)
	}

	return *backings
}

func (ta *TruAPI) backingPoolResolver(_ context.Context, q story.Story) sdk.Coin {
	res, err := ta.RunQuery(path.Join(backing.QueryPath, backing.QueryBackingAmountByStoryID), app.QueryByIDParams{ID: q.ID})
	if err != nil {
		fmt.Println("backingPoolResolver err: ", err)
		return sdk.Coin{}
	}

	amount := new(sdk.Coin)
	err = amino.UnmarshalJSON(res, amount)
	if err != nil {
		panic(err)
	}

	return *amount
}

func (ta *TruAPI) challengePoolResolver(_ context.Context, q story.Story) sdk.Coin {
	res, err := ta.RunQuery(path.Join(challenge.QueryPath, challenge.QueryChallengeAmountByStoryID), app.QueryByIDParams{ID: q.ID})
	if err != nil {
		fmt.Println("challengePoolResolver err: ", err)
		return sdk.Coin{}
	}

	amount := new(sdk.Coin)
	err = amino.UnmarshalJSON(res, amount)
	if err != nil {
		panic(err)
	}

	return *amount
}

func (ta *TruAPI) categoryResolver(ctx context.Context, q category.QueryCategoryByIDParams) category.Category {
	res, err := ta.RunQuery("categories/id", q)
	if err != nil {
		fmt.Println("categoryResolver err: ", err)
		return category.Category{}
	}

	c := new(category.Category)
	err = json.Unmarshal(res, c)
	if err != nil {
		panic(err)
	}

	return *c
}

func (ta *TruAPI) challengeResolver(_ context.Context, q app.QueryByIDParams) challenge.Challenge {
	res, err := ta.RunQuery("challenges/id", q)
	if err != nil {
		fmt.Println("error getting challenge", err)
		return challenge.Challenge{}
	}
	challenge := challenge.Challenge{}
	err = json.Unmarshal(res, &challenge)
	if err != nil {
		panic(err)
	}

	return challenge
}

func (ta *TruAPI) challengesResolver(_ context.Context, q app.QueryByIDParams) []challenge.Challenge {
	res, err := ta.RunQuery(path.Join(challenge.QueryPath, challenge.QueryByStoryID), q)
	if err != nil {
		fmt.Println("challengesResolver err: ", err)
		return []challenge.Challenge{}
	}

	challenges := new([]challenge.Challenge)
	err = json.Unmarshal(res, challenges)
	if err != nil {
		panic(err)
	}

	return *challenges
}

func (ta *TruAPI) paramsResolver(_ context.Context) params.Params {
	res, err := ta.RunQuery("params", nil)
	if err != nil {
		fmt.Println("paramsResolver err: ", err)
		return params.Params{}
	}

	p := new(params.Params)
	err = json.Unmarshal(res, p)
	if err != nil {
		panic(err)
	}

	return *p
}

func (ta *TruAPI) storyCategoryResolver(ctx context.Context, q story.Story) category.Category {
	return ta.categoryResolver(ctx, category.QueryCategoryByIDParams{ID: q.CategoryID})
}

func (ta *TruAPI) storyResolver(_ context.Context, q story.QueryStoryByIDParams) story.Story {
	res, err := ta.RunQuery("stories/id", q)
	if err != nil {
		fmt.Println("storyResolver err: ", err)
		return story.Story{}
	}

	s := new(story.Story)
	err = json.Unmarshal(res, s)
	if err != nil {
		panic(err)
	}

	return *s
}

func (ta *TruAPI) twitterProfileResolver(ctx context.Context, addr string) db.TwitterProfile {
	twitterProfile, err := ta.DBClient.TwitterProfileByAddress(addr)
	if twitterProfile == nil {
		return db.TwitterProfile{}
	}
	if err != nil {
		// TODO [shanev]: Add back after adding error handling to resolvers
		// fmt.Println("Resolver err: ", err)
		return db.TwitterProfile{}
	}

	return *twitterProfile
}

func (ta *TruAPI) usersResolver(ctx context.Context, q users.QueryUsersByAddressesParams) []users.User {
	res, err := ta.RunQuery("users/addresses", q)
	if err != nil {
		fmt.Println("usersResolver err: ", err)
		return []users.User{}
	}

	u := new([]users.User)
	err = amino.UnmarshalJSON(res, u)
	if err != nil {
		panic(err)
	}

	return *u
}

func (ta *TruAPI) transactionsResolver(_ context.Context, q app.QueryByCreatorParams) []trubank.Transaction {
	res, err := ta.RunQuery(path.Join(trubank.QueryPath, trubank.QueryTransactionsByCreator), q)
	if err != nil {
		fmt.Println("transactionsResolver err: ", err)
		return []trubank.Transaction{}
	}

	transactions := new([]trubank.Transaction)
	err = json.Unmarshal(res, transactions)
	if err != nil {
		panic(err)
	}

	storyTransactions := make([]trubank.Transaction, 0)
	for _, transaction := range *transactions {
		if transaction.TransactionType != trubank.InviteAFriend {
			storyTransactions = append(storyTransactions, transaction)
		}
	}

	return storyTransactions
}

func (ta *TruAPI) unreadNotificationsCountResolver(ctx context.Context, q struct{}) *db.NotificationsCountResponse {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return &db.NotificationsCountResponse{
			Count: 0,
		}
	}
	response, err := ta.DBClient.UnreadNotificationEventsCountByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return response
}

func (ta *TruAPI) unseenNotificationsCountResolver(ctx context.Context, q struct{}) *db.NotificationsCountResponse {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return &db.NotificationsCountResponse{
			Count: 0,
		}
	}
	response, err := ta.DBClient.UnseenNotificationEventsCountByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return response
}

func (ta *TruAPI) notificationsResolver(ctx context.Context, q struct{}) []db.NotificationEvent {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return make([]db.NotificationEvent, 0)
	}
	evts, err := ta.DBClient.NotificationEventsByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return evts
}

func (ta *TruAPI) addressesWhoFlaggedResolver(ctx context.Context, q story.Story) []string {
	flaggedStories, err := ta.DBClient.FlaggedStoriesByStoryID(q.ID)
	if err != nil {
		return []string{}
	}
	var addressesWhoFlagged []string
	for _, story := range flaggedStories {
		addressesWhoFlagged = append(addressesWhoFlagged, story.Creator)
	}
	return addressesWhoFlagged
}

func (ta *TruAPI) filterFeedStories(ctx context.Context, feedStories []story.Story, filter FeedFilter) ([]story.Story, error) {
	switch filter {
	case Latest:
		return ta.filterByLatest(ctx, feedStories)
	case Trending:
		return ta.filterByTrending(ctx, feedStories)
	case Completed:
		return ta.filterByCompleted(ctx, feedStories)
	}
	return feedStories, nil
}

func (ta *TruAPI) filterFlaggedStories(stories *[]story.Story) ([]story.Story, error) {
	unflaggedStories := make([]story.Story, 0)
	for _, story := range *stories {
		storyFlags, err := ta.DBClient.FlaggedStoriesByStoryID(story.ID)
		if err != nil {
			return nil, err
		}
		if len(storyFlags) > 0 {
			if storyFlags[0].Creator == ta.APIContext.Config.Flag.Admin {
				continue
			}
		}
		if len(storyFlags) < ta.APIContext.Config.Flag.Limit {
			unflaggedStories = append(unflaggedStories, story)
		}
	}

	return unflaggedStories, nil
}

func (ta *TruAPI) commentsResolver(ctx context.Context, q argument.Argument) []db.Comment {
	comments, err := ta.DBClient.CommentsByArgumentID(q.ID)
	if err != nil {
		panic(err)
	}
	return comments
}

func (ta *TruAPI) invitesResolver(ctx context.Context) []db.Invite {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return make([]db.Invite, 0)
	}

	twitterProfile, err := ta.DBClient.TwitterProfileByID(user.TwitterProfileID)
	if err != nil {
		panic(err)
	}

	// TODO: pull this in from an ENV
	if strings.EqualFold(twitterProfile.Username, "lilrushishah") ||
		strings.EqualFold(twitterProfile.Username, "patel0phone") ||
		strings.EqualFold(twitterProfile.Username, "iam_preethi") ||
		strings.EqualFold(twitterProfile.Username, "truted2") {
		invites, err := ta.DBClient.Invites()
		if err != nil {
			panic(err)
		}
		return invites
	}
	invites, err := ta.DBClient.InvitesByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return invites
}

func (ta *TruAPI) reactionsCountResolver(ctx context.Context, rxnable db.Reactionable) []db.ReactionsCount {
	reactionsCount, err := ta.DBClient.ReactionsCountByReactionable(rxnable)
	if err != nil {
		panic(err)
	}
	return reactionsCount
}

func (ta *TruAPI) reactionsResolver(ctx context.Context, rxnable db.Reactionable) []db.Reaction {
	reactions, err := ta.DBClient.ReactionsByReactionable(rxnable)
	if err != nil {
		panic(err)
	}
	return reactions
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
