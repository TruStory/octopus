package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-pg/pg"
	"github.com/joho/godotenv"

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
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file", err)
	}
	dbConfig := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_ADDR_NO_PORT", "localhost"),
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
		from = time.Now()
	} else {
		// starting from today only
		from = time.Now()
	}
	to = time.Now()

	statistia.seedBetween(from, to)
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
	metrics := statistia.fetchMetrics(date)
	fmt.Printf("Seeding for... %s\n", date.Format("2006-01-02"))

	for address, userMetric := range metrics.Users {
		fmt.Printf("\tCalculating for User... %s\n", address)
		for communityID, communityMetric := range userMetric.CommunityMetrics {

			// by default, assume that the user has no previous activity,
			// thus, today's metrics become the daily metrics,
			// thus, creating and initializing a default struct.
			dayMetric := db.UserMetric{
				Address:            address,
				AsOnDate:           date,
				CommunityID:        communityID,
				TotalAmountStaked:  communityMetric.Metrics.TotalAmountStaked.Amount,
				StakeEarned:        communityMetric.Metrics.StakeEarned.Amount,
				StakeLost:          communityMetric.Metrics.StakeLost.Amount,
				TotalAmountAtStake: communityMetric.Metrics.TotalAmountAtStake.Amount,
				AvailableStake:     communityMetric.Metrics.AvailableStake.Amount,
			}

			fmt.Printf("\t\tSaving...\n")
			err := statistia.saveMetrics(tx, dayMetric)
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
func (statistia *service) fetchMetrics(date time.Time) *SystemMetrics {
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

	metrics := &SystemMetrics{
		Users: make(map[string]*UserMetricsV2),
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
