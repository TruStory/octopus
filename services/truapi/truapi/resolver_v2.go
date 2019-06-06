package truapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

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
	Stakers      []string
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

func convertStoryToClaim(story story.Story) Claim {
	return Claim{
		ID:          story.ID,
		CommunityID: story.CategoryID,
		Body:        story.Body,
		Creator:     story.Creator,
		Source:      story.Source,
		CreatedTime: story.Timestamp.CreatedTime,
	}
}

func convertStoryArgumentToClaimArgument(storyArgument argument.Argument, argumentMeta argumentMeta) Argument {
	bodyLength := len(storyArgument.Body)
	if bodyLength > SummaryLength {
		bodyLength = SummaryLength
	}
	summary := storyArgument.Body[:bodyLength]
	stakeType := Backing
	if argumentMeta.Vote == false {
		stakeType = Challenge
	}
	claimArgument := Argument{
		Stake: Stake{
			ID:          storyArgument.ID,
			Creator:     storyArgument.Creator,
			CreatedTime: storyArgument.Timestamp.CreatedTime,
			Type:        stakeType,
		},
		UpvotedCount: argumentMeta.UpvotedCount,
		Body:         storyArgument.Body,
		Summary:      summary,
	}
	return claimArgument
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
		claim := convertStoryToClaim(story)
		claims = append(claims, claim)
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claims)
	if err != nil {
		fmt.Println("Resolver err: ", err)
		panic(err)
	}

	filteredClaims := unflaggedClaims

	return filteredClaims
}

func (ta *TruAPI) claimResolver(ctx context.Context, q queryByClaimID) Claim {
	story := ta.storyResolver(ctx, story.QueryStoryByIDParams{ID: q.ID})
	return convertStoryToClaim(story)
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
	claim := convertStoryToClaim(story)
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
				Stakers:      []string{backing.Creator().String()},
			}
		} else {
			storyArguments[backing.ArgumentID].UpvotedCount++
			storyArguments[backing.ArgumentID].Stakers = append(storyArguments[backing.ArgumentID].Stakers, backing.Creator().String())
		}
	}
	for _, challenge := range challenges {
		if storyArguments[challenge.ArgumentID] == nil {
			storyArguments[challenge.ArgumentID] = &argumentMeta{
				Vote:         challenge.VoteChoice(),
				UpvotedCount: 0,
				Stakers:      []string{challenge.Creator().String()},
			}
		} else {
			storyArguments[challenge.ArgumentID].UpvotedCount++
			storyArguments[challenge.ArgumentID].Stakers = append(storyArguments[challenge.ArgumentID].Stakers, challenge.Creator().String())
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

func (ta *TruAPI) claimTotalBackedResolver(ctx context.Context, q Claim) sdk.Coin {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: q.ID})
	amount := sdk.NewCoin(app.StakeDenom, sdk.ZeroInt())
	for _, backing := range backings {
		amount = amount.Add(backing.Amount())
	}
	return amount
}

func (ta *TruAPI) claimTotalChallengedResolver(ctx context.Context, q Claim) sdk.Coin {
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: q.ID})
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

func (ta *TruAPI) claimCommentsResolver(ctx context.Context, q queryByClaimID) []db.Comment {
	arguments := ta.claimArgumentsResolver(ctx, q)
	comments := make([]db.Comment, 0)
	for _, argument := range arguments {
		argument := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: argument.ID})
		argComments := ta.commentsResolver(ctx, argument)
		comments = append(comments, argComments...)
	}
	return comments
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

func removeDuplicates(elements []int64) []int64 {
	encountered := map[int64]bool{}
	result := []int64{}

	for v := range elements {
		if encountered[elements[v]] == true {

		} else {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}
	return result
}
