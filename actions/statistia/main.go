package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-pg/pg"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	db "github.com/TruStory/octopus/services/truapi/db"
	"github.com/machinebox/graphql"
)

type service struct {
	metricsEndpoint string
	httpClient      *http.Client
	dbClient        *db.Client
	graphqlClient   *graphql.Client
}

func main() {
	dbConfig := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_ADDR", "localhost"),
			Port: 5432,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}
	statistia := &service{
		metricsEndpoint: getEnv("STATISTIA_METRICS_ENDPOINT", "http://localhost:1337/api/v1/metrics"),
		httpClient:      &http.Client{},
		dbClient:        db.NewDBClient(dbConfig),
		graphqlClient:   graphql.NewClient(getEnv("STATISTIA_GRAPHQL_ENDPOINT", "http://localhost:1337/api/v1/graphql")),
	}
	defer statistia.dbClient.Close()

	statistia.run()
}

func (statistia *service) run() {

	areEmpty, err := statistia.dbClient.AreUserMetricsEmpty()
	if err != nil {
		panic(err)
	}

	// Running as the historical seeder
	// OR
	// Running as daily cron job
	var from, to time.Time
	if areEmpty {
		// starting from the April Fool's day of 2019
		from, err = time.Parse("2006-01-02", "2019-04-01")
		if err != nil {
			panic(err)
		}
	} else {
		// starting from today only
		from = time.Now()
	}
	to = time.Now()

	statistia.seedInitialBalances()
	statistia.seedBetween(from, to)
}

// seedInitialBalances seeds the initial balances of all the users
// whose initial balances are not tracked yet
func (statistia *service) seedInitialBalances() {
	fmt.Printf("----------SEEDING INITIAL BALANCES----------\n")
	users, err := statistia.fetchUsers()
	if err != nil {
		panic(err)
	}

	tomorrow := time.Now().Add(24 * 1 * time.Hour)
	tmMetrics := statistia.fetchMetrics(tomorrow)
	for address, user := range users {
		initialBalance := db.InitialStakeBalance{
			Address:        address,
			InitialBalance: filterStakeCoin(user.Coins).Amount,
		}

		// if any activity by the user is found, we will reverse calculate the initial balance
		userMetrics, ok := tmMetrics.Users[address]
		if ok {
			var totalStakeEarned, totalStakeLost, totalInterestEarned, totalAtStake uint64
			for _, categoryMetric := range userMetrics.CategoryMetrics {
				totalStakeEarned += categoryMetric.Metrics.StakeEarned.Amount
				totalStakeLost += categoryMetric.Metrics.StakeLost.Amount
				totalInterestEarned += categoryMetric.Metrics.InterestEarned.Amount
				totalAtStake += categoryMetric.Metrics.TotalAmountAtStake.Amount
			}

			// initial balance = current balance - total earned + total lost - total interest earned + at stake
			initialBalance.InitialBalance = userMetrics.Balance.Amount - totalStakeEarned + totalStakeLost - totalInterestEarned + totalAtStake
		}
		fmt.Println(address, initialBalance)
		err = statistia.dbClient.UpsertInitialStakeBalance(initialBalance)
		if err != nil {
			panic(err)
		}
	}
}

// seedBetween seeds the user daily metrics between the two given dates
func (statistia *service) seedBetween(from, to time.Time) {
	fmt.Printf("----------SEEDING USER METRICS----------\n")

	// adding two days because date.Before checks "<" and not "<="
	to = to.Add(24 * 1 * time.Hour)

	err := statistia.dbClient.RunInTransaction(func(tx *pg.Tx) error {
		// set date to starting date and keep adding 1 day to it as long as it comes before to
		for date := from; date.Before(to); date = date.AddDate(0, 0, 1) {
			err := statistia.seedInTxFor(tx, date)

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

// seedFor seeds the user daily metrics for the given date
func (statistia *service) seedInTxFor(tx *pg.Tx, date time.Time) error {
	today := date
	yesterday := date.Add(-24 * 1 * time.Hour)

	yMetrics := statistia.fetchMetrics(yesterday)
	tMetrics := statistia.fetchMetrics(today)
	fmt.Printf("Seeding for... %s in comparison with...%s\n", today.Format("2006-01-02"), yesterday.Format("2006-01-02"))

	for address, tUserMetric := range tMetrics.Users {
		fmt.Printf("\tCalculating for User... %s\n", address)
		for categoryID, tCategoryMetric := range tUserMetric.CategoryMetrics {
			fmt.Printf("\t\tCalculating for Category... %s\n", tCategoryMetric.CategoryName)

			// by default, assume that the user has no previous activity,
			// thus, today's metrics become the daily metrics,
			// thus, creating and initializing a default struct.
			dUserMetric := db.UserMetric{
				Address:                   address,
				AsOnDate:                  today,
				CategoryID:                categoryID,
				TotalClaims:               tCategoryMetric.Metrics.TotalClaims,
				TotalArguments:            tCategoryMetric.Metrics.TotalArguments,
				TotalClaimsBacked:         tCategoryMetric.Metrics.TotalBackings,
				TotalClaimsChallenged:     tCategoryMetric.Metrics.TotalChallenges,
				TotalAmountBacked:         tCategoryMetric.Metrics.TotalAmountBacked.Amount,
				TotalAmountChallenged:     tCategoryMetric.Metrics.TotalAmountChallenged.Amount,
				TotalEndorsementsGiven:    tCategoryMetric.Metrics.TotalGivenEndorsements,
				TotalEndorsementsReceived: tCategoryMetric.Metrics.TotalReceivedEndorsements,
				TotalAmountStaked:         tCategoryMetric.Metrics.TotalAmountStaked.Amount,
				TotalAmountAtStake:        tCategoryMetric.Metrics.TotalAmountAtStake.Amount,
				StakeEarned:               tCategoryMetric.Metrics.StakeEarned.Amount,
				StakeLost:                 tCategoryMetric.Metrics.StakeLost.Amount,
				InterestEarned:            tCategoryMetric.Metrics.InterestEarned.Amount,
				StakeBalance:              tUserMetric.RunningBalance.Amount,
				CredEarned:                tCategoryMetric.CredEarned.Amount,
			}

			// if any activity is found on the previous day,
			// we'll calculate the difference to get the given day's metrics.
			yUserMetric, ok := yMetrics.Users[address]
			if ok {
				dUserMetric.StakeBalance = tUserMetric.RunningBalance.Minus(yUserMetric.RunningBalance).Amount
				yCategoryMetric, ok := yUserMetric.CategoryMetrics[categoryID]
				if ok {
					dUserMetric.TotalClaims = tCategoryMetric.Metrics.TotalClaims - yCategoryMetric.Metrics.TotalClaims
					dUserMetric.TotalArguments = tCategoryMetric.Metrics.TotalArguments - yCategoryMetric.Metrics.TotalArguments
					dUserMetric.TotalClaimsBacked = tCategoryMetric.Metrics.TotalBackings - yCategoryMetric.Metrics.TotalBackings
					dUserMetric.TotalClaimsChallenged = tCategoryMetric.Metrics.TotalChallenges - yCategoryMetric.Metrics.TotalChallenges
					dUserMetric.TotalEndorsementsGiven = tCategoryMetric.Metrics.TotalGivenEndorsements - yCategoryMetric.Metrics.TotalGivenEndorsements
					dUserMetric.TotalEndorsementsReceived = tCategoryMetric.Metrics.TotalReceivedEndorsements - yCategoryMetric.Metrics.TotalReceivedEndorsements
					dUserMetric.TotalAmountBacked = tCategoryMetric.Metrics.TotalAmountBacked.Minus(yCategoryMetric.Metrics.TotalAmountBacked).Amount
					dUserMetric.TotalAmountChallenged = tCategoryMetric.Metrics.TotalAmountChallenged.Minus(yCategoryMetric.Metrics.TotalAmountChallenged).Amount
					dUserMetric.TotalAmountStaked = tCategoryMetric.Metrics.TotalAmountStaked.Minus(yCategoryMetric.Metrics.TotalAmountStaked).Amount
					dUserMetric.TotalAmountAtStake = tCategoryMetric.Metrics.TotalAmountAtStake.Minus(yCategoryMetric.Metrics.TotalAmountAtStake).Amount
					dUserMetric.StakeEarned = tCategoryMetric.Metrics.StakeEarned.Minus(yCategoryMetric.Metrics.StakeEarned).Amount
					dUserMetric.StakeLost = tCategoryMetric.Metrics.StakeLost.Minus(yCategoryMetric.Metrics.StakeLost).Amount
					dUserMetric.InterestEarned = tCategoryMetric.Metrics.InterestEarned.Minus(yCategoryMetric.Metrics.InterestEarned).Amount
					dUserMetric.CredEarned = tCategoryMetric.CredEarned.Minus(yCategoryMetric.CredEarned).Amount
				}
			}

			fmt.Printf("\t\tSaving...\n")
			err := statistia.saveMetrics(tx, dUserMetric)
			if err != nil {
				fmt.Printf("\t\tSaving FAILED\n")
				return err
			}
		}
	}

	return nil
}

// saveMetrics dumps the per-day-basis metrics for the user in database
func (statistia *service) saveMetrics(tx *pg.Tx, metrics db.UserMetric) error {
	err := db.UpsertDailyUserMetricInTx(tx, metrics)
	if err != nil {
		return err
	}

	return nil
}

// fetchUsers fetches the users and their coin holdings from the chain
func (statistia *service) fetchUsers() (map[string]User, error) {
	users := make([]db.TwitterProfile, 0)
	addresses := make([]string, 0)
	err := statistia.dbClient.FindAll(&users)
	if err != nil {
		panic(err)
	}
	for _, user := range users {
		addresses = append(addresses, user.Address)
	}

	graphqlReq := graphql.NewRequest(UsersByAddressesQuery)

	graphqlReq.Var("addresses", addresses)
	var usersMap = make(map[string]User)
	var graphqlRes UsersByAddressesResponse
	ctx := context.Background()
	if err := statistia.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return usersMap, err
	}

	for _, user := range graphqlRes.Users {
		usersMap[user.ID] = user
	}
	return usersMap, nil
}

// fetchMetrics fetches the metrics for a given day from the metrics endpoint
func (statistia *service) fetchMetrics(date time.Time) *MetricsSummary {
	request, err := http.NewRequest(
		"GET", statistia.metricsEndpoint+"?date="+date.Format("2006-01-02"), nil,
	)
	if err != nil {
		panic(err)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	response, err := statistia.httpClient.Do(request)
	if err != nil {
		panic(err)
	}

	metrics := &MetricsSummary{
		Users: make(map[string]*UserMetrics),
	}
	err = json.NewDecoder(response.Body).Decode(metrics)
	if err != nil {
		panic(err)
	}

	return metrics
}

// filterStakeCoin returns the trustake/trusteak coin
func filterStakeCoin(coins []Coin) Coin {
	var stake Coin

	// assuming that everyone will have the stake coin balance
	for _, coin := range coins {
		if coin.Denom == "trusteak" || coin.Denom == "trustake" {
			stake = coin
		}
	}

	return stake
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}
