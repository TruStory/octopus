package truapi

import (
	"context"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/account"
	"github.com/TruStory/truchain/x/bank"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/julianshen/og"
	tcmn "github.com/tendermint/tendermint/libs/common"
	stripmd "github.com/writeas/go-strip-markdown"
)

// ArgumentFilter defines filters for claimArguments
type ArgumentFilter int64

// List of ArgumentFilter types
const (
	ArgumentAll ArgumentFilter = iota
	ArgumentCreated
	ArgumentAgreed
)

type queryByCommunityID struct {
	CommunityID string `graphql:"communityId"`
}

type queryByClaimID struct {
	ID uint64 `graphql:"id"`
}

type queryByArgumentID struct {
	ID uint64 `graphql:"id"`
}

type queryByStakeID struct {
	ID uint64 `graphql:"id"`
}

type queryByAddress struct {
	ID string `graphql:"id"`
}

type queryClaimArgumentParams struct {
	ClaimID uint64         `graphql:"id,optional"`
	Address *string        `graphql:"address,optional"`
	Filter  ArgumentFilter `graphql:"filter,optional"`
}

type queryByCommunityIDAndFeedFilter struct {
	CommunityID string     `graphql:"communityId,optional"`
	FeedFilter  FeedFilter `graphql:"feedFilter,optional"`
}

// claimMetricsBest represents all-time claim metrics
type claimMetricsBest struct {
	Claim                 claim.Claim
	TotalAmountStaked     sdk.Int
	TotalStakers          uint64
	TotalComments         int
	BackingChallengeDelta sdk.Int
}

// claimMetricsTrending represents claim metrics within last 24 hours
type claimMetricsTrending struct {
	Claim          claim.Claim
	TotalArguments int64
	TotalComments  int
	TotalStakes    int64
}

// appAccountEarningsFilter is query params for filtering the app account's earnings
type appAccountEarningsFilter struct {
	ID   string
	From string
	To   string
}

type appAccountCommunityEarning struct {
	Address      string   `json:"address"`
	CommunityID  string   `json:"community_id"`
	TotalEarned  sdk.Coin `json:"total_earned"`
	WeeklyEarned sdk.Coin `json:"weekly_earned"`
}

type appAccountEarning struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

type appAccountEarnings struct {
	NetEarnings sdk.Coin            `json:"net_earnings"`
	DataPoints  []appAccountEarning `json:"data_points"`
}

func (ta *TruAPI) appAccountResolver(ctx context.Context, q queryByAddress) *AppAccount {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("account AccAddressFromBech32 err: ", err)
		return nil
	}

	queryRoute := path.Join(account.QuerierRoute, account.QueryAppAccount)
	res, err := ta.Query(queryRoute, account.QueryAppAccountParams{Address: address}, account.ModuleCodec)
	if err != nil {
		return nil
	}

	var aa = new(account.AppAccount)
	err = account.ModuleCodec.UnmarshalJSON(res, aa)
	if err != nil {
		fmt.Println("AppAccount UnmarshalJSON err: ", err)
		return nil
	}

	var pubKey []byte

	// GetPubKey can return nil and Bytes() will panic due to nil pointer
	if aa.GetPubKey() != nil {
		pubKey = aa.GetPubKey().Bytes()
	}

	return &AppAccount{
		Address:       aa.GetAddress().String(),
		AccountNumber: aa.GetAccountNumber(),
		Coins:         aa.GetCoins(),
		Sequence:      aa.GetSequence(),
		Pubkey:        tcmn.HexBytes(pubKey),
		SlashCount:    aa.SlashCount,
		IsJailed:      aa.IsJailed,
		JailEndTime:   aa.JailEndTime,
		CreatedTime:   aa.CreatedTime,
	}
}

func (ta *TruAPI) earnedBalanceResolver(ctx context.Context, q queryByAddress) sdk.Coin {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("earnedBalanceResolver err: ", err)
		return sdk.Coin{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryTotalEarnedCoins)
	res, err := ta.Query(queryRoute, staking.QueryTotalEarnedCoinsParams{Address: address}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("earnedBalanceResolver err: ", err)
		return sdk.Coin{}
	}

	balance := new(sdk.Coin)
	err = staking.ModuleCodec.UnmarshalJSON(res, balance)
	if err != nil {
		fmt.Println("totalEarnedCoin UnmarshalJSON err: ", err)
		return sdk.Coin{}
	}

	return *balance
}

func (ta *TruAPI) earnedStakeResolver(ctx context.Context, q queryByAddress) []EarnedCoin {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("earnedStakeResolver err: ", err)
		return []EarnedCoin{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryEarnedCoins)
	res, err := ta.Query(queryRoute, staking.QueryEarnedCoinsParams{Address: address}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("earnedStakeResolver err: ", err)
		return []EarnedCoin{}
	}

	coins := make([]sdk.Coin, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &coins)
	if err != nil {
		fmt.Println("earnedCoin UnmarshalJSON err: ", err)
		return []EarnedCoin{}
	}

	earnedCoins := make([]EarnedCoin, 0)
	for _, coin := range coins {
		earnedCoins = append(earnedCoins, EarnedCoin{
			coin,
			coin.Denom,
		})
	}

	return earnedCoins
}

func (ta *TruAPI) communitiesResolver(ctx context.Context) []community.Community {
	// the communities need to be curated better before they are made public
	communityBlacklist := []string{"sports", "tech", "entertainment"}

	queryRoute := path.Join(community.QuerierRoute, community.QueryCommunities)
	res, err := ta.Query(queryRoute, struct{}{}, community.ModuleCodec)
	if err != nil {
		fmt.Println("communitiesResolver err: ", err)
		return []community.Community{}
	}

	cs := make([]community.Community, 0)
	err = community.ModuleCodec.UnmarshalJSON(res, &cs)
	if err != nil {
		fmt.Println("community UnmarshalJSON err: ", err)
		return []community.Community{}
	}

	// sort in alphabetical order
	sort.Slice(cs, func(i, j int) bool {
		return (cs)[j].Name > (cs)[i].Name
	})

	// exclude blacklisted communities
	filteredCommunities := make([]community.Community, 0)
	for _, c := range cs {
		if !contains(communityBlacklist, c.ID) {
			filteredCommunities = append(filteredCommunities, c)
		}
	}

	return filteredCommunities
}

func (ta *TruAPI) communityResolver(ctx context.Context, q queryByCommunityID) *community.Community {
	queryRoute := path.Join(community.QuerierRoute, community.QueryCommunity)
	res, err := ta.Query(queryRoute, community.QueryCommunityParams{ID: q.CommunityID}, community.ModuleCodec)
	if err != nil {
		fmt.Println("getCommunityByIDResolver err: ", err)
		return nil
	}

	c := new(community.Community)
	err = community.ModuleCodec.UnmarshalJSON(res, c)
	if err != nil {
		return nil
	}
	return c
}

func (ta *TruAPI) communityIconImageResolver(ctx context.Context, q community.Community) CommunityIconImage {
	return CommunityIconImage{
		Regular: joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_normal.png", q.ID)),
		Active:  joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_active.png", q.ID)),
	}
}

func (ta *TruAPI) claimsResolver(ctx context.Context, q queryByCommunityIDAndFeedFilter) []claim.Claim {
	var res []byte
	var err error
	if q.CommunityID == "all" {
		queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaims)
		res, err = ta.Query(queryRoute, struct{}{}, claim.ModuleCodec)
	} else {
		queryRoute := path.Join(claim.QuerierRoute, claim.QueryCommunityClaims)
		res, err = ta.Query(queryRoute, claim.QueryCommunityClaimsParams{CommunityID: q.CommunityID}, claim.ModuleCodec)
	}
	if err != nil {
		fmt.Println("claimsResolver err: ", err)
		return []claim.Claim{}
	}

	claims := make([]claim.Claim, 0)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claims)
	if err != nil {
		panic(err)
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claims)
	if err != nil {
		fmt.Println("filterFlaggedClaims err: ", err)
		panic(err)
	}

	filteredClaims := ta.filterFeedClaims(ctx, unflaggedClaims, q.FeedFilter)

	return filteredClaims
}

func (ta *TruAPI) claimResolver(ctx context.Context, q queryByClaimID) claim.Claim {
	queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaim)
	res, err := ta.Query(queryRoute, claim.QueryClaimParams{ID: q.ID}, claim.ModuleCodec)
	if err != nil {
		fmt.Println("claimResolver err: ", err)
		return claim.Claim{}
	}

	c := new(claim.Claim)
	err = claim.ModuleCodec.UnmarshalJSON(res, c)
	if err != nil {
		fmt.Println("claim UnmarshalJSON err: ", err)
		return claim.Claim{}
	}

	return *c
}

func (ta *TruAPI) claimOfTheDayResolver(ctx context.Context, q queryByCommunityID) *claim.Claim {
	claimOfTheDayID, err := ta.DBClient.ClaimOfTheDayIDByCommunityID(q.CommunityID)
	if err != nil {
		return nil
	}

	claim := ta.claimResolver(ctx, queryByClaimID{ID: uint64(claimOfTheDayID)})
	return &claim
}

func (ta *TruAPI) filterFlaggedClaims(claims []claim.Claim) ([]claim.Claim, error) {
	unflaggedClaims := make([]claim.Claim, 0)
	for _, claim := range claims {
		claimFlags, err := ta.DBClient.FlaggedStoriesByStoryID(int64(claim.ID))
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

func (ta *TruAPI) claimArgumentsResolver(ctx context.Context, q queryClaimArgumentParams) []staking.Argument {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimArguments)
	res, err := ta.Query(queryRoute, staking.QueryClaimArgumentsParams{ClaimID: q.ClaimID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentsResolver err: ", err)
		return []staking.Argument{}
	}

	arguments := make([]staking.Argument, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &arguments)
	if err != nil {
		fmt.Println("[]staking.Argument UnmarshalJSON err: ", err)
		return []staking.Argument{}
	}
	filteredArguments := make([]staking.Argument, 0)
	for _, argument := range arguments {
		if q.Filter == ArgumentCreated {
			if argument.Creator.String() == *q.Address {
				filteredArguments = append(filteredArguments, argument)
			}
		} else if q.Filter == ArgumentAgreed {
			stakes := ta.claimArgumentStakesResolver(ctx, argument)
			for _, stake := range stakes {
				if stake.Creator.String() == *q.Address {
					filteredArguments = append(filteredArguments, argument)
					break
				}
			}
		} else {
			filteredArguments = append(filteredArguments, argument)
		}
	}

	return filteredArguments
}

func (ta *TruAPI) claimArgumentResolver(ctx context.Context, q queryByArgumentID) *staking.Argument {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimArgument)
	res, err := ta.Query(queryRoute, staking.QueryClaimArgumentParams{ArgumentID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentResolver err: ", err)
		return nil
	}

	argument := new(staking.Argument)
	err = staking.ModuleCodec.UnmarshalJSON(res, argument)
	if err != nil {
		return nil
	}

	return argument
}

func (ta *TruAPI) topArgumentResolver(ctx context.Context, q claim.Claim) *staking.Argument {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimTopArgument)
	res, err := ta.Query(queryRoute, staking.QueryClaimTopArgumentParams{ClaimID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("topArgumentResolver err: ", err)
		return nil
	}

	argument := new(staking.Argument)
	err = staking.ModuleCodec.UnmarshalJSON(res, argument)
	if err != nil {
		fmt.Println("staking.Argument UnmarshalJSON err: ", err)
		return nil
	}

	// no top argument
	if argument.ID == 0 {
		return nil
	}

	return argument
}

func (ta *TruAPI) claimStakersResolver(ctx context.Context, q claim.Claim) []AppAccount {
	stakers := make([]AppAccount, 0)
	arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: q.ID})
	for _, argument := range arguments {
		stakers = append(stakers, ta.claimArgumentStakersResolver(ctx, argument)...)
	}
	return stakers
}

func (ta *TruAPI) claimParticipantsResolver(ctx context.Context, q claim.Claim) []AppAccount {
	participants := ta.claimStakersResolver(ctx, q)
	comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: q.ID})
	for _, comment := range comments {
		if !participantExists(participants, comment.Creator) {
			participants = append(participants, *ta.appAccountResolver(ctx, queryByAddress{ID: comment.Creator}))
		}
	}
	return participants
}

func participantExists(participants []AppAccount, participantToAddAddress string) bool {
	for _, participant := range participants {
		if participant.Address == participantToAddAddress {
			return true
		}
	}
	return false
}

func (ta *TruAPI) stakeResolver(ctx context.Context, q queryByStakeID) *staking.Stake {
	queryRoute := path.Join(staking.ModuleName, staking.QueryStake)
	res, err := ta.Query(queryRoute, staking.QueryStakeParams{StakeID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("stakeResolver err: ", err)
		return nil
	}

	stake := new(staking.Stake)
	err = staking.ModuleCodec.UnmarshalJSON(res, stake)
	if err != nil {
		return nil
	}

	return stake
}

func (ta *TruAPI) claimArgumentStakesResolver(ctx context.Context, q staking.Argument) []staking.Stake {
	queryRoute := path.Join(staking.ModuleName, staking.QueryArgumentStakes)
	res, err := ta.Query(queryRoute, staking.QueryArgumentStakesParams{ArgumentID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentStakesResolver err: ", err)
		return []staking.Stake{}
	}

	stakes := make([]staking.Stake, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &stakes)
	if err != nil {
		fmt.Println("[]staking.Stake UnmarshalJSON err: ", err)
		return []staking.Stake{}
	}

	return stakes
}

func (ta *TruAPI) claimArgumentStakersResolver(ctx context.Context, q staking.Argument) []AppAccount {
	stakes := ta.claimArgumentStakesResolver(ctx, q)
	appAccounts := make([]AppAccount, 0)
	for _, stake := range stakes {
		if stake.Type == staking.StakeUpvote {
			appAccounts = append(appAccounts, *ta.appAccountResolver(ctx, queryByAddress{ID: stake.Creator.String()}))
		}
	}
	return appAccounts
}

func (ta *TruAPI) appAccountStakeResolver(ctx context.Context, q staking.Argument) *staking.Stake {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if ok {
		stakes := ta.claimArgumentStakesResolver(ctx, q)
		for _, stake := range stakes {
			if user.Address == stake.Creator.String() {
				return &stake
			}
		}
	}
	return nil
}

func (ta *TruAPI) claimCommentsResolver(ctx context.Context, q queryByClaimID) []db.Comment {
	comments, err := ta.DBClient.CommentsByClaimID(q.ID)
	if err != nil {
		panic(err)
	}
	return comments
}

func (ta *TruAPI) appAccountClaimsCreatedResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("appAccountClaimsCreatedResolver err: ", err)
		return []claim.Claim{}
	}

	queryRoute := path.Join(claim.QuerierRoute, claim.QueryCreatorClaims)
	res, err := ta.Query(queryRoute, claim.QueryCreatorClaimsParams{Creator: creator}, claim.ModuleCodec)
	if err != nil {
		return []claim.Claim{}
	}

	claimsCreated := make([]claim.Claim, 0)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claimsCreated)
	if err != nil {
		fmt.Println("[]claim.Claim UnmarshalJSON err: ", err)
		return []claim.Claim{}
	}

	return claimsCreated
}

func (ta *TruAPI) appAccountClaimsWithArgumentsResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		return []claim.Claim{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryUserArguments)
	res, err := ta.Query(queryRoute, staking.QueryUserArgumentsParams{Address: creator}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("appAccountClaimsWithArguments err: ", err)
		return []claim.Claim{}
	}

	arguments := make([]staking.Argument, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &arguments)
	if err != nil {
		fmt.Println("[]staking.Argument UnmarshalJSON err: ", err)
		return []claim.Claim{}
	}

	// TODO: instead of a loop, pass a list of claim IDs to fetch from cosmos
	claimsWithArgument := make([]claim.Claim, 0)
	for _, argument := range arguments {
		claim := ta.claimResolver(ctx, queryByClaimID{ID: argument.ClaimID})
		claimsWithArgument = append(claimsWithArgument, claim)
	}
	return claimsWithArgument
}

func (ta *TruAPI) appAccountClaimsWithAgreesResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	stakes := ta.agreesResolver(ctx, q)

	claims := make([]claim.Claim, 0)
	for _, stake := range stakes {
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{ID: stake.ArgumentID})
		if argument != nil {
			claim := ta.claimResolver(ctx, queryByClaimID{ID: argument.ClaimID})
			claims = append(claims, claim)
		}
	}

	return claims
}

func (ta *TruAPI) agreesResolver(ctx context.Context, q queryByAddress) []staking.Stake {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		return []staking.Stake{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryUserStakes)
	res, err := ta.Query(queryRoute, staking.QueryUserStakesParams{Address: creator}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("agreesResolver err: ", err)
		return []staking.Stake{}
	}

	stakes := make([]staking.Stake, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &stakes)
	if err != nil {
		return []staking.Stake{}
	}

	agrees := make([]staking.Stake, 0)
	for _, stake := range stakes {
		if stake.Type == staking.StakeUpvote {
			agrees = append(agrees, stake)
		}
	}

	return agrees
}

func (ta *TruAPI) appAccountTransactionsResolver(ctx context.Context, q queryByAddress) []bank.Transaction {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		return []bank.Transaction{}
	}

	queryRoute := path.Join(bank.QuerierRoute, bank.QueryTransactionsByAddress)
	res, err := ta.Query(queryRoute, bank.QueryTransactionsByAddressParams{Address: creator}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("appAccountTransactionsResolver err: ", err)
		return []bank.Transaction{}
	}

	transactions := make([]bank.Transaction, 0)
	err = bank.ModuleCodec.UnmarshalJSON(res, &transactions)
	if err != nil {
		return []bank.Transaction{}
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[j].CreatedTime.Before(transactions[i].CreatedTime)
	})

	return transactions
}

func (ta *TruAPI) transactionReferenceResolver(ctx context.Context, t bank.Transaction) TransactionReference {
	var tr TransactionReference
	switch t.Type {
	case bank.TransactionRegistration:
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceNone,
			Title:       TransactionTypeTitle[t.Type],
			Body:        "",
		}
	case bank.TransactionRewardPayout:
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceNone,
			Title:       TransactionTypeTitle[t.Type],
			Body:        "",
		}
	case bank.TransactionUpvote:
		fallthrough
	case bank.TransactionUpvoteReturned:
		fallthrough
	case bank.TransactionInterestUpvoteGiven:
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{t.ReferenceID})
		creatorTwitterProfile := ta.twitterProfileResolver(ctx, argument.Creator.String())
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       fmt.Sprintf(TransactionTypeTitle[t.Type], creatorTwitterProfile.Username),
			Body:        stripmd.Strip(argument.Summary),
		}
	case bank.TransactionInterestUpvoteReceived:
		stake := ta.stakeResolver(ctx, queryByStakeID{ID: t.ReferenceID})
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{stake.ArgumentID})
		stakerTwitterProfile := ta.twitterProfileResolver(ctx, stake.Creator.String())
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       fmt.Sprintf(TransactionTypeTitle[t.Type], stakerTwitterProfile.Username),
			Body:        stripmd.Strip(argument.Summary),
		}
	case bank.TransactionBacking:
		fallthrough
	case bank.TransactionBackingReturned:
		fallthrough
	case bank.TransactionChallenge:
		fallthrough
	case bank.TransactionChallengeReturned:
		fallthrough
	case bank.TransactionInterestArgumentCreation:
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{t.ReferenceID})
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       TransactionTypeTitle[t.Type],
			Body:        stripmd.Strip(argument.Summary),
		}
	default:
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceNone,
			Title:       "",
			Body:        "",
		}
	}
	return tr
}

func (ta *TruAPI) sourceURLPreviewResolver(ctx context.Context, q claim.Claim) string {
	sourceURLPreview, err := ta.DBClient.ClaimSourceURLPreview(q.ID)
	if err == nil && sourceURLPreview != "" {
		// found sourceURLPreview in the database, exit early
		return sourceURLPreview
	}

	n := (q.ID % 5) // random but deterministic placeholder image 0-4
	defaultPreview := joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("sourceUrlPreview_default_%d.png", n))

	if q.Source.String() == "" {
		sourceURLPreview = defaultPreview
	} else {
		// fetch open graph image from source url website
		ogImage := og.OgImage{}
		err = og.GetPageDataFromUrl(q.Source.String(), &ogImage)

		if err != nil || ogImage.Url == "" {
			// no open graph image exists
			sourceURLPreview = defaultPreview
		} else {
			sourceURLPreview = ogImage.Url
		}
	}

	_ = ta.DBClient.AddClaimSourceURLPreview(&db.ClaimSourceURLPreview{
		ClaimID:          q.ID,
		SourceURLPreview: sourceURLPreview,
	})

	return sourceURLPreview
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

func (ta *TruAPI) filterFeedClaims(ctx context.Context, claims []claim.Claim, filter FeedFilter) []claim.Claim {
	if filter == Latest {
		// Reverse chronological order, up to 1 week
		latestClaims := make([]claim.Claim, 0)
		for _, claim := range claims {
			if claim.CreatedTime.After(time.Now().AddDate(0, 0, -7)) {
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
		bestClaims := make([]claim.Claim, 0)
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
		trendingClaims := make([]claim.Claim, 0)
		for _, metric := range metrics {
			trendingClaims = append(trendingClaims, metric.Claim)
		}
		return trendingClaims
	}
	return claims
}

func (ta *TruAPI) appAccountCommunityEarningsResolver(ctx context.Context, q queryByAddress) []appAccountCommunityEarning {
	appAccount := ta.appAccountResolver(ctx, q)

	to := time.Now()
	from := to.Add(-7 * 24 * time.Hour)
	metrics, err := ta.DBClient.AggregateUserMetricsByAddressBetweenDates(appAccount.Address, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		panic(err)
	}

	communityEarnings := make([]appAccountCommunityEarning, 0)
	mappedCommunityEarnings := make(map[string]sdk.Coin)

	for _, metric := range metrics {
		runningEarning := mappedCommunityEarnings[metric.CommunityID]
		if runningEarning.Denom == "" {
			// not previously found
			mappedCommunityEarnings[metric.CommunityID] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(int64(metric.StakeEarned)))
		} else {
			// previously found
			mappedCommunityEarnings[metric.CommunityID] = runningEarning.Add(sdk.NewCoin(app.StakeDenom, sdk.NewInt(int64(metric.StakeEarned))))
		}
	}

	for communityID, earning := range mappedCommunityEarnings {
		communityEarnings = append(communityEarnings, appAccountCommunityEarning{
			Address:      q.ID,
			CommunityID:  communityID,
			WeeklyEarned: earning,
			TotalEarned:  sdk.NewCoin(app.StakeDenom, sdk.NewInt(int64(metrics[len(metrics)-1].StakeEarned))), // last item of the array
		})
	}

	return communityEarnings
}

func (ta *TruAPI) appAccountEarningsResolver(ctx context.Context, q appAccountEarningsFilter) appAccountEarnings {
	appAccount := ta.appAccountResolver(ctx, queryByAddress{ID: q.ID})

	metrics, err := ta.DBClient.AggregateUserMetricsByAddressBetweenDates(appAccount.Address, q.From, q.To)
	if err != nil {
		panic(err)
	}

	dataPoints := make([]appAccountEarning, 0)
	mappedDataPoints := make(map[string]sdk.Coin)
	var mappedSortedKeys []string
	// seeding empty dates
	from, err := time.Parse("2006-01-02", q.From)
	if err != nil {
		panic(err)
	}
	to, err := time.Parse("2006-01-02", q.To)
	if err != nil {
		panic(err)
	}
	for date := from; date.Before(to); date = date.AddDate(0, 0, 1) {
		key := date.Format("2006-01-02")
		mappedDataPoints[key] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
		mappedSortedKeys = append(mappedSortedKeys, key)
	}

	for _, metric := range metrics {
		mappedDataPoints[metric.AsOnDate.Format("2006-01-02")] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(int64(metric.AvailableStake)))
	}

	for _, key := range mappedSortedKeys {
		dataPoints = append(dataPoints, appAccountEarning{
			Date:   key,
			Amount: mappedDataPoints[key].Amount.Quo(sdk.NewInt(app.Shanev)).ToDec().RoundInt64(),
		})
	}

	return appAccountEarnings{
		NetEarnings: sdk.NewCoin(app.StakeDenom, sdk.NewInt(int64(metrics[len(metrics)-1].AvailableStake))), // last item of the array
		DataPoints:  dataPoints,
	}
}
