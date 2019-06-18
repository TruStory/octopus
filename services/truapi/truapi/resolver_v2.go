package truapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/argument"
	"github.com/TruStory/truchain/x/category"
	"github.com/TruStory/truchain/x/story"
	"github.com/TruStory/truchain/x/users"
	sdk "github.com/cosmos/cosmos-sdk/types"
	amino "github.com/tendermint/go-amino"
)

type queryByCommunityID struct {
	ID int64 `graphql:"id"`
}

type queryByCommunitySlug struct {
	CommunitySlug string `graphql:"communitySlug"`
}

type queryByClaimID struct {
	ID int64 `graphql:"id"`
}

type queryByArgumentID struct {
	ID int64 `graphql:"id"`
}

type queryByAddress struct {
	ID string `graphql:"id"`
}

type queryByCommunitySlugAndFeedFilter struct {
	CommunitySlug string     `graphql:",optional"`
	FeedFilter    FeedFilter `graphql:",optional"`
}

type argumentMeta struct {
	Vote         bool
	UpvotedCount int64
}

// claimMetricsBest represents all-time claim metrics
type claimMetricsBest struct {
	Claim                 Claim
	TotalAmountStaked     sdk.Int
	TotalStakers          int64
	TotalComments         int
	BackingChallengeDelta sdk.Int
}

// claimMetricsTrending represents claim metrics within last 24 hours
type claimMetricsTrending struct {
	Claim          Claim
	TotalArguments int64
	TotalComments  int
	TotalStakes    int64
}

// SummaryLength is amount of characters allowed when summarizing an argument
const SummaryLength = 140

func convertCategoryToCommunity(category category.Category) Community {
	return Community{
		ID:          category.ID,
		Name:        category.Title,
		Slug:        category.Slug,
		Description: category.Description,
	}
}

func (ta *TruAPI) convertStoryToClaim(ctx context.Context, story story.Story) Claim {
	totalStakers := len(ta.claimStakersResolver(ctx, Claim{ID: story.ID}))
	return Claim{
		ID:              story.ID,
		CommunityID:     story.CategoryID,
		Body:            story.Body,
		Creator:         story.Creator,
		Source:          story.Source,
		TotalBacked:     ta.totalBackingStakeByStoryID(ctx, story.ID),
		TotalChallenged: ta.totalChallengeStakeByStoryID(ctx, story.ID),
		TotalStakers:    int64(totalStakers),
		CreatedTime:     story.Timestamp.CreatedTime,
	}
}

func convertStoryArgumentToClaimArgument(storyArgument argument.Argument, argumentMeta argumentMeta) Argument {
	bodyLength := len(storyArgument.Body)
	if bodyLength > SummaryLength {
		bodyLength = SummaryLength
	}
	summary := storyArgument.Body[:bodyLength]
	var stakeType StakeType
	if argumentMeta.Vote {
		stakeType = Backing
	} else {
		stakeType = Challenge
	}
	claimArgument := Argument{
		Stake: Stake{
			ID:          storyArgument.ID,
			Creator:     storyArgument.Creator,
			CreatedTime: storyArgument.Timestamp.CreatedTime,
			Type:        stakeType,
		},
		ClaimID:      storyArgument.StoryID,
		UpvotedCount: argumentMeta.UpvotedCount,
		Body:         storyArgument.Body,
		Summary:      summary,
	}
	return claimArgument
}

func convertCommentToClaimComment(comment db.Comment) ClaimComment {
	return ClaimComment{
		ID:         comment.ID,
		ParentID:   comment.ParentID,
		ArgumentID: comment.ArgumentID,
		Body:       comment.Body,
		Creator:    comment.Creator,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
		DeletedAt:  comment.DeletedAt,
	}
}

func (ta *TruAPI) appAccountResolver(ctx context.Context, q queryByAddress) AppAccount {
	addresses := users.QueryUsersByAddressesParams{
		Addresses: []string{q.ID},
	}

	res, err := ta.RunQuery("users/addresses", addresses)
	if err != nil {
		return AppAccount{}
	}

	users := new([]users.User)
	err = amino.UnmarshalJSON(res, users)
	if err != nil {
		return AppAccount{}
	}
	if len(*users) == 0 {
		return AppAccount{}
	}
	u := (*users)[0]

	// split User.Coins into AppAccount.Coins and AppAccount.EarnedStake
	trustake := make(sdk.Coins, 0)
	earnedStake := make([]EarnedCoin, 0)

	trustake = append(trustake, sdk.NewCoin(app.StakeDenom, u.Coins.AmountOf(app.StakeDenom)))

	communityID := int64(1)
	for _, coin := range u.Coins {
		if coin.Denom != app.StakeDenom {
			earnedCoin := sdk.NewCoin(app.StakeDenom, coin.Amount)
			earned := EarnedCoin{
				Coin:        earnedCoin,
				CommunityID: communityID,
			}
			earnedStake = append(earnedStake, earned)
			communityID++
		}
	}

	appAccount := AppAccount{
		BaseAccount: BaseAccount{
			Address:       u.Address,
			AccountNumber: u.AccountNumber,
			Coins:         trustake,
			Sequence:      u.Sequence,
			PubKey:        u.Pubkey,
		},
		EarnedStake: earnedStake,
	}

	return appAccount
}

func (ta *TruAPI) communitiesResolver(ctx context.Context) []Community {
	res, err := ta.RunQuery("categories/all", struct{}{})
	if err != nil {
		fmt.Println("Resolver err: ", res)
		return []Community{}
	}

	cs := new([]category.Category)
	err = json.Unmarshal(res, cs)
	if err != nil {
		return []Community{}
	}

	// sort in alphabetical order
	sort.Slice(*cs, func(i, j int) bool {
		return (*cs)[j].Title > (*cs)[i].Title
	})

	communities := make([]Community, 0)
	for _, category := range *cs {
		community := convertCategoryToCommunity(category)
		communities = append(communities, community)
	}

	return communities
}

func (ta *TruAPI) communityResolver(ctx context.Context, q queryByCommunitySlug) *Community {
	community, err := ta.getCommunityBySlug(ctx, q.CommunitySlug)
	if err != nil {
		return nil
	}
	return &community
}

func (ta *TruAPI) communityIconImageResolver(ctx context.Context, q Community) CommunityIconImage {
	return CommunityIconImage{
		Regular: joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_normal.png", q.Slug)),
		Active:  joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_active.png", q.Slug)),
	}
}

func (ta *TruAPI) claimsResolver(ctx context.Context, q queryByCommunitySlugAndFeedFilter) []Claim {
	var res []byte
	var err error
	var community Community
	if q.CommunitySlug == "" {
		res, err = ta.RunQuery("stories/all", struct{}{})
	} else {
		community, err = ta.getCommunityBySlug(ctx, q.CommunitySlug)
		if err != nil {
			return []Claim{}
		}
		res, err = ta.RunQuery("stories/category", story.QueryCategoryStoriesParams{CategoryID: community.ID})
	}
	if err != nil {
		fmt.Println("Resolver err: ", res)
		return []Claim{}
	}

	stories := new([]story.Story)
	err = json.Unmarshal(res, stories)
	if err != nil {
		panic(err)
	}

	claims := make([]Claim, 0)
	for _, story := range *stories {
		claim := ta.convertStoryToClaim(ctx, story)
		claims = append(claims, claim)
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claims)
	if err != nil {
		fmt.Println("Resolver err: ", err)
		panic(err)
	}

	filteredClaims := ta.filterFeedClaims(ctx, unflaggedClaims, q.FeedFilter)

	return filteredClaims
}

func (ta *TruAPI) claimResolver(ctx context.Context, q queryByClaimID) Claim {
	story := ta.storyResolver(ctx, story.QueryStoryByIDParams{ID: q.ID})
	return ta.convertStoryToClaim(ctx, story)
}

func (ta *TruAPI) claimOfTheDayResolver(ctx context.Context, q queryByCommunitySlug) *Claim {
	slug := q.CommunitySlug
	if slug == "" {
		slug = "all"
	}
	claimOfTheDayID, err := ta.DBClient.ClaimOfTheDayIDByCommunitySlug(slug)
	if err != nil {
		return nil
	}

	story := ta.storyResolver(ctx, story.QueryStoryByIDParams{ID: claimOfTheDayID})
	claim := ta.convertStoryToClaim(ctx, story)
	return &claim
}

func (ta *TruAPI) filterFlaggedClaims(claims []Claim) ([]Claim, error) {
	unflaggedClaims := make([]Claim, 0)
	for _, claim := range claims {
		claimFlags, err := ta.DBClient.FlaggedStoriesByStoryID(claim.ID)
		if err != nil {
			return nil, err
		}
		if len(claimFlags) > 0 {
			if claimFlags[0].Creator == ta.APIContext.Config.Flag.Admin {
				continue
			}
		}
		if len(claimFlags) < ta.APIContext.Config.Flag.Limit {
			unflaggedClaims = append(unflaggedClaims, claim)
		}
	}

	return unflaggedClaims, nil
}

func (ta *TruAPI) getCommunityBySlug(ctx context.Context, slug string) (Community, error) {
	// Client displays claims under each community AND all claims on the homepage
	// When querying claims for the homepage the client sends no community slug
	// In this case we create a empty string community with the title "All" which is rendered on client
	if slug == "" {
		return Community{
			ID:   -1,
			Slug: "all",
			Name: "All",
		}, nil
	}

	cs := ta.allCategoriesResolver(ctx, struct{}{})

	var cat category.Category
	for _, category := range cs {
		if category.Slug == slug {
			cat = category
			break
		}
	}
	if cat.ID == 0 {
		return Community{}, errors.New("Category not found")
	}

	community := convertCategoryToCommunity(cat)
	return community, nil
}

func (ta *TruAPI) getCommunityByID(ctx context.Context, q queryByCommunityID) *Community {
	category := ta.categoryResolver(ctx, category.QueryCategoryByIDParams{ID: q.ID})
	community := convertCategoryToCommunity(category)
	return &community
}

func (ta *TruAPI) claimArgumentsResolver(ctx context.Context, q queryByClaimID) []Argument {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: q.ID})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: q.ID})
	storyArguments := map[int64]*argumentMeta{}
	for _, backing := range backings {
		if storyArguments[backing.ArgumentID] == nil {
			storyArguments[backing.ArgumentID] = &argumentMeta{
				Vote:         backing.VoteChoice(),
				UpvotedCount: 0,
			}
		} else {
			storyArguments[backing.ArgumentID].UpvotedCount++
		}
	}
	for _, challenge := range challenges {
		if storyArguments[challenge.ArgumentID] == nil {
			storyArguments[challenge.ArgumentID] = &argumentMeta{
				Vote:         challenge.VoteChoice(),
				UpvotedCount: 0,
			}
		} else {
			storyArguments[challenge.ArgumentID].UpvotedCount++
		}
	}
	arguments := make([]Argument, 0)
	for argumentID, argumentMeta := range storyArguments {
		storyArgument := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: argumentID})
		argument := convertStoryArgumentToClaimArgument(storyArgument, *argumentMeta)
		arguments = append(arguments, argument)
	}

	return arguments
}

func (ta *TruAPI) topArgumentResolver(ctx context.Context, q Claim) *Argument {
	arguments := ta.claimArgumentsResolver(ctx, queryByClaimID{ID: q.ID})
	if len(arguments) == 0 {
		return nil
	}
	return &arguments[0]
}

func (ta *TruAPI) totalBackingStakeByStoryID(ctx context.Context, ID int64) sdk.Coin {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: ID})
	amount := sdk.NewCoin(app.StakeDenom, sdk.ZeroInt())
	for _, backing := range backings {
		amount = amount.Add(backing.Amount())
	}
	return amount
}

func (ta *TruAPI) totalChallengeStakeByStoryID(ctx context.Context, ID int64) sdk.Coin {
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: ID})
	amount := sdk.NewCoin(app.StakeDenom, sdk.ZeroInt())
	for _, challenge := range challenges {
		amount = amount.Add(challenge.Amount())
	}
	return amount
}

func (ta *TruAPI) claimStakersResolver(ctx context.Context, q Claim) []AppAccount {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: q.ID})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: q.ID})
	appAccounts := make([]AppAccount, 0)
	for _, backing := range backings {
		appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: backing.Creator().String()}))
	}
	for _, challenge := range challenges {
		appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: challenge.Creator().String()}))
	}
	return appAccounts
}

func (ta *TruAPI) claimParticipantsResolver(ctx context.Context, q Claim) []AppAccount {
	participants := ta.claimStakersResolver(ctx, q)
	comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: q.ID})
	for _, comment := range comments {
		if !participantExists(participants, comment.Creator) {
			participants = append(participants, ta.appAccountResolver(ctx, queryByAddress{ID: comment.Creator}))
		}
	}
	if !participantExists(participants, q.Creator.String()) {
		participants = append(participants, ta.appAccountResolver(ctx, queryByAddress{ID: q.Creator.String()}))
	}
	return participants
}

func participantExists(participants []AppAccount, participantToAdd string) bool {
	for _, participant := range participants {
		if participantToAdd == participant.Address {
			return true
		}
	}
	return false
}

func (ta *TruAPI) claimArgumentStakersResolver(ctx context.Context, q Argument) []AppAccount {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: q.ClaimID})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: q.ClaimID})
	appAccounts := make([]AppAccount, 0)
	for _, backing := range backings {
		if backing.ArgumentID == q.ID && !backing.Creator().Equals(q.Creator) {
			appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: backing.Creator().String()}))
		}
	}
	for _, challenge := range challenges {
		if challenge.ArgumentID == q.ID && !challenge.Creator().Equals(q.Creator) {
			appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: challenge.Creator().String()}))
		}
	}
	return appAccounts
}

func (ta *TruAPI) claimCommentsResolver(ctx context.Context, q queryByClaimID) []ClaimComment {
	arguments := ta.claimArgumentsResolver(ctx, q)
	comments := make([]db.Comment, 0)
	for _, argument := range arguments {
		argument := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: argument.ID})
		argComments := ta.commentsResolver(ctx, argument)
		comments = append(comments, argComments...)
	}
	claimComments := make([]ClaimComment, 0)
	for _, comment := range comments {
		claimComments = append(claimComments, convertCommentToClaimComment(comment))
	}
	return claimComments
}

func (ta *TruAPI) stakesResolver(_ context.Context, q queryByArgumentID) []Stake {
	return []Stake{}
}

func (ta *TruAPI) settingsResolver(_ context.Context) Settings {
	return Settings{
		MinClaimLength:    25,
		MaxClaimLength:    140,
		MinArgumentLength: 25,
		MaxArgumentLength: 1250,
		MinSummaryLength:  25,
		MaxSummaryLength:  140,
		MinCommentLength:  5,
		MaxCommentLength:  1000,
		BlockIntervalTime: 5000,
		DefaultStake:      sdk.NewCoin(app.StakeDenom, sdk.NewInt(30*app.Shanev)),
	}
}

func (ta *TruAPI) filterFeedClaims(ctx context.Context, claims []Claim, filter FeedFilter) []Claim {
	if filter == Latest {
		// Reverse chronological order, up to 1 week
		latestClaims := make([]Claim, 0)
		for _, claim := range claims {
			if claim.CreatedTime.Before(time.Now().AddDate(0, 0, -7)) {
				latestClaims = append(latestClaims, claim)
			}
		}
		sort.Slice(latestClaims, func(i, j int) bool {
			return latestClaims[j].CreatedTime.Before(latestClaims[i].CreatedTime)
		})
		return latestClaims
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
		bestClaims := make([]Claim, 0)
		for _, metric := range metrics {
			bestClaims = append(bestClaims, metric.Claim)
		}
		return bestClaims
	} else if filter == Trending {
		// highest volume of activity in last 24 hours
		// # of new arguments       TODO: need tendermint tags
		// # of new comments
		// # of new agree stakes    TODO: need tendermint tags
		metrics := make([]claimMetricsTrending, 0)
		for _, claim := range claims {
			comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: claim.ID})
			totalComments := 0
			for _, comment := range comments {
				if comment.CreatedAt.Before(time.Now().AddDate(0, 0, -1)) {
					totalComments++
				}
			}
			metric := claimMetricsTrending{
				Claim:         claim,
				TotalComments: totalComments,
			}
			metrics = append(metrics, metric)
		}
		sort.Slice(metrics, func(i, j int) bool {
			if metrics[i].TotalArguments > metrics[j].TotalArguments {
				return true
			}
			if metrics[i].TotalArguments < metrics[j].TotalArguments {
				return false
			}
			if metrics[i].TotalComments > metrics[j].TotalComments {
				return true
			}
			if metrics[i].TotalComments < metrics[j].TotalComments {
				return false
			}
			return metrics[j].TotalStakes-metrics[i].TotalStakes > 0
		})
		trendingClaims := make([]Claim, 0)
		for _, metric := range metrics {
			trendingClaims = append(trendingClaims, metric.Claim)
		}
		return trendingClaims
	}
	return claims
}
