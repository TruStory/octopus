package truapi

import (
	"context"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/TruStory/truchain/x/bank/exported"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-pg/pg"

	"github.com/TruStory/octopus/services/truapi/db"
)

// leaderboard defaults
const (
	// leaderboardInitialDate this is the release date for beta and won't change.
	leaderboardInitialDate = "2019-07-11"
	// 30 minutes for refreshing interval
	leaderboardDefaultInterval = 30
	// display top 50
	leaderboardDefaultTopDisplaying = 50
)

type UserStatsByCommunity struct {
	EarnedCoin     sdk.Int
	Claims         int64
	Arguments      int64
	AgreesGiven    int64
	AgreesReceived int64
}
type UserStats struct {
	CommunityStats map[string]*UserStatsByCommunity
}
type LeaderboardStats struct {
	UserStats map[string]*UserStats
}

func (l *LeaderboardStats) getUserStats(address string) *UserStats {
	userStats, ok := l.UserStats[address]
	if !ok {
		userStats = &UserStats{CommunityStats: make(map[string]*UserStatsByCommunity)}
		l.UserStats[address] = userStats
	}
	return userStats
}

func (l *LeaderboardStats) getUserStatsByCommunity(address, communityID string) *UserStatsByCommunity {
	userStats := l.getUserStats(address)
	ucs, ok := userStats.CommunityStats[communityID]
	if !ok {
		ucs = &UserStatsByCommunity{
			EarnedCoin: sdk.NewInt(0),
		}
		userStats.CommunityStats[communityID] = ucs
	}
	return ucs
}

func (ta *TruAPI) statsByDate(date time.Time) (*LeaderboardStats, error) {
	// Get all claims
	claims := make([]claim.Claim, 0)
	result, err := ta.Query(
		path.Join(claim.QuerierRoute, claim.QueryClaimsBeforeTime),
		claim.QueryClaimsTimeParams{CreatedTime: date},
		claim.ModuleCodec,
	)
	if err != nil {
		return nil, err
	}
	err = claim.ModuleCodec.UnmarshalJSON(result, &claims)
	if err != nil {
		return nil, err
	}

	// For each user, get the available stake calculated.
	users := make([]db.User, 0)
	err = ta.DBClient.FindAll(&users)
	if err != nil {
		return nil, err
	}

	stats := &LeaderboardStats{UserStats: make(map[string]*UserStats)}
	for _, claim := range claims {
		if !claim.CreatedTime.Before(date) {
			continue
		}
		argumentCreatorsMappings := make(map[uint64]string)
		ucs := stats.getUserStatsByCommunity(claim.Creator.String(), claim.CommunityID)
		ucs.Claims++
		arguments, err := ta.getClaimArguments(claim.ID)
		if err != nil {
			return nil, err
		}
		for _, argument := range arguments {
			if !argument.CreatedTime.Before(date) {
				continue
			}
			ucs := stats.getUserStatsByCommunity(argument.Creator.String(), claim.CommunityID)
			ucs.Arguments++
			argumentCreatorsMappings[argument.ID] = argument.Creator.String()
		}
		stakes := ta.claimStakesResolver(context.Background(), claim)
		for _, stake := range stakes {
			if !stake.CreatedTime.Before(date) {
				continue
			}
			ucs := stats.getUserStatsByCommunity(stake.Creator.String(), claim.CommunityID)
			if stake.Type == staking.StakeUpvote {
				stats.getUserStatsByCommunity(argumentCreatorsMappings[stake.ArgumentID], claim.CommunityID).AgreesReceived++
				ucs.AgreesGiven++
			}
		}
	}
	trackedTransactions := []exported.TransactionType{
		exported.TransactionInterestArgumentCreation,
		exported.TransactionInterestUpvoteReceived,
		exported.TransactionInterestUpvoteGiven,
	}
	for _, user := range users {
		if user.Address == "" || !user.CreatedAt.Before(date) {
			continue
		}
		transactions := ta.appAccountTransactionsResolver(context.Background(), queryByAddress{ID: user.Address})
		for _, transaction := range transactions {
			if !transaction.CreatedTime.Before(date) {
				continue
			}
			if !transaction.Type.OneOf(trackedTransactions) {
				continue
			}
			if transaction.CommunityID == "" {
				return nil, fmt.Errorf("transaction %s [%d] must contain community id",
					transaction.Type.String(), transaction.ID)
			}
			ucs := stats.getUserStatsByCommunity(user.Address, transaction.CommunityID)
			switch transaction.Type {
			case exported.TransactionInterestArgumentCreation:
				fallthrough
			case exported.TransactionInterestUpvoteReceived:
				fallthrough
			case exported.TransactionInterestUpvoteGiven:
				fallthrough
			case exported.TransactionCuratorReward:
				ucs.EarnedCoin = ucs.EarnedCoin.Add(transaction.Amount.Amount)
			}
		}
	}
	return stats, nil
}

func (ta *TruAPI) leaderboardSeed() (time.Time, error) {
	t, err := time.Parse("2006-01-02", leaderboardInitialDate)
	if err != nil {
		return t, err
	}
	endDate := getMidnightHour(t)
	err = ta.DBClient.FeedLeaderboardInTransaction(func(tx *pg.Tx) error {
		stats, err := ta.statsByDate(endDate)
		if err != nil {
			return err
		}
		for user, userStats := range stats.UserStats {
			for communityID, cStats := range userStats.CommunityStats {

				m := db.LeaderboardUserMetric{
					Date:           t,
					Earned:         cStats.EarnedCoin.Int64(),
					CommunityID:    communityID,
					Address:        user,
					AgreesGiven:    cStats.AgreesGiven,
					AgreesReceived: cStats.AgreesReceived,
				}
				err := ta.DBClient.UpsertLeaderboardMetric(tx, &m)
				if err != nil {
					return err
				}
			}
		}
		err = ta.DBClient.UpsertLeaderboardProcessedDate(tx, &db.LeaderboardProcessedDate{
			Date:        t,
			FullDay:     true,
			MetricsTime: endDate,
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return t, err
	}
	return t, nil
}

func (ta *TruAPI) leaderboardStatsBetween(start, end time.Time) error {
	endStats, err := ta.statsByDate(end)
	if err != nil {
		return err
	}
	startStats, err := ta.statsByDate(start)
	if err != nil {
		return err
	}

	err = ta.DBClient.FeedLeaderboardInTransaction(func(tx *pg.Tx) error {
		for user, endUserStats := range endStats.UserStats {

			for communityID, endUserCommunityStats := range endUserStats.CommunityStats {
				startUserCommunityStats := startStats.getUserStatsByCommunity(user, communityID)
				m := db.LeaderboardUserMetric{
					Date:           start,
					Earned:         endUserCommunityStats.EarnedCoin.Sub(startUserCommunityStats.EarnedCoin).Int64(),
					CommunityID:    communityID,
					Address:        user,
					AgreesGiven:    endUserCommunityStats.AgreesGiven - startUserCommunityStats.AgreesGiven,
					AgreesReceived: endUserCommunityStats.AgreesReceived - startUserCommunityStats.AgreesReceived,
				}
				err := ta.DBClient.UpsertLeaderboardMetric(tx, &m)
				if err != nil {
					return err
				}
			}
		}
		err = ta.DBClient.UpsertLeaderboardProcessedDate(tx, &db.LeaderboardProcessedDate{
			Date:        start,
			FullDay:     getMidnightHour(start) == end,
			MetricsTime: end,
		})
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func getZeroHour(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func getMidnightHour(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 0, time.UTC)
}
func getLeaderboardBetweenDates(t time.Time) (start, end time.Time) {
	return getZeroHour(t), getMidnightHour(t)
}

func (ta *TruAPI) processStats() error {
	log.Println("Running leaderboard stats")
	lastDate, err := ta.DBClient.LastLeaderboardProcessedDate()
	if err != nil {
		return err
	}

	var lastProcessedDateTime time.Time
	if lastDate == nil {
		log.Println("Processing seed metrics")
		date, err := ta.leaderboardSeed()
		if err != nil {
			return err
		}
		lastProcessedDateTime = date
	}

	if lastDate != nil {
		lastProcessedDateTime = lastDate.Date
	}
	now := time.Now().UTC()
	start := getZeroHour(now)
	dateToProcess := lastProcessedDateTime.Add(time.Duration(24) * time.Hour)
	for dateToProcess.Before(start) {
		log.Println("Syncing pending date", dateToProcess.Format("2006-01-02"))
		s, e := getLeaderboardBetweenDates(dateToProcess)
		err := ta.leaderboardStatsBetween(s, e)
		if err != nil {
			return err
		}
		dateToProcess = dateToProcess.Add(time.Duration(24) * time.Hour)
	}

	// calculating current day metrics
	err = ta.leaderboardStatsBetween(start, now)
	if err != nil {
		return err
	}
	log.Println("Completed leaderboard stats")
	return nil
}

func (ta *TruAPI) leaderboardScheduler() {
	if !ta.APIContext.Config.Leaderboard.Enabled {
		log.Println("leaderboard is disabled")
		return
	}
	interval := leaderboardDefaultInterval
	if ta.APIContext.Config.Leaderboard.Interval > 0 {
		interval = ta.APIContext.Config.Leaderboard.Interval
	}
	log.Printf("leaderboard: update interval of %d minutes \n", interval)
	// try to sync when truapi just started
	err := ta.processStats()
	if err != nil {
		log.Println("an error occurred processing stats, waiting for next interval", err)
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	for range ticker.C {
		err := ta.processStats()
		if err != nil {
			log.Println("an error occurred processing stats", err)
		}
	}
}

type queryByDateAndMetricFilter struct {
	DateFilter LeaderboardDateFilter   `graphql:"dateFilter,optional"`
	Metric     LeaderboardMetricFilter `graphql:"metricFilter,optional"`
}

func (ta *TruAPI) leaderboardResolver(ctx context.Context, q queryByDateAndMetricFilter) []db.LeaderboardTopUser {
	limit := leaderboardDefaultTopDisplaying
	if ta.APIContext.Config.Leaderboard.TopDisplaying > 0 {
		limit = ta.APIContext.Config.Leaderboard.TopDisplaying
	}
	since := getZeroHour(time.Now().Add(q.DateFilter.Value()))
	// all time
	if q.DateFilter.Value() == 0 {
		since = time.Time{}
	}
	sortBy := q.Metric.Value()
	topUsers, err := ta.DBClient.Leaderboard(since, sortBy, limit, ta.APIContext.Config.Community.InactiveCommunities)
	if err != nil {
		log.Println("couldn't get leaderboard results", err)
	}
	return topUsers
}
