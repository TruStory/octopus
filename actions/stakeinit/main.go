package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	db "github.com/TruStory/octopus/services/truapi/db"
	"github.com/machinebox/graphql"
)

type service struct {
	httpClient    *http.Client
	dbClient      *db.Client
	graphqlClient *graphql.Client
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

	stakeinit := &service{
		httpClient:    &http.Client{},
		dbClient:      db.NewDBClient(dbConfig),
		graphqlClient: graphql.NewClient(mustEnv("STAKEINIT_GRAPHQL_ENDPOINT")),
	}
	stakeinit.seedInitialBalances()
}

func (stakeinit *service) seedInitialBalances() {
	users, err := stakeinit.fetchUsers()
	if err != nil {
		panic(err)
	}

	tomorrow := time.Now().Add(24 * 1 * time.Hour)
	tmMetrics := stakeinit.fetchMetrics(tomorrow)

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
		err = stakeinit.dbClient.UpsertInitialStakeBalance(initialBalance)
		if err != nil {
			panic(err)
		}
	}
}

// fetchUsers fetches the users and their coin holdings from the chain
func (stakeinit *service) fetchUsers() (map[string]User, error) {
	users := make([]db.TwitterProfile, 0)
	addresses := make([]string, 0)
	err := stakeinit.dbClient.FindAll(&users)
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
	if err := stakeinit.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return usersMap, err
	}

	for _, user := range graphqlRes.Users {
		usersMap[user.ID] = user
	}
	return usersMap, nil
}

// fetchMetrics fetches the metrics for a given day from the metrics endpoint
func (stakeinit *service) fetchMetrics(date time.Time) *MetricsSummary {
	request, err := http.NewRequest(
		"GET", mustEnv("STAKEINIT_METRICS_ENDPOINT")+"?date="+date.Format("2006-01-02"), nil,
	)
	if err != nil {
		panic(err)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	response, err := stakeinit.httpClient.Do(request)
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

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}
