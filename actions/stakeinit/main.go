package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	db "github.com/TruStory/octopus/services/truapi/db"
)

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

	dbClient := db.NewDBClient(dbConfig)
	seedInitialBalances(dbClient)
}

func seedInitialBalances(dbClient *db.Client) {
	today := time.Now()
	tomorrow := today.Add(24 * 1 * time.Hour)

	tmMetrics := fetchMetrics(tomorrow)
	for address, userMetrics := range tmMetrics.Users {
		var totalStakeEarned, totalStakeLost, totalInterestEarned, totalAtStake uint64
		for _, categoryMetric := range userMetrics.CategoryMetrics {
			totalStakeEarned += categoryMetric.Metrics.StakeEarned.Amount
			totalStakeLost += categoryMetric.Metrics.StakeLost.Amount
			totalInterestEarned += categoryMetric.Metrics.InterestEarned.Amount
			totalAtStake += categoryMetric.Metrics.TotalAmountAtStake.Amount
		}

		// initial balance = current balance - total earned + total lost - total interest earned + at stake
		initialBalance := db.InitialStakeBalance{
			Address:        address,
			InitialBalance: userMetrics.Balance.Amount - totalStakeEarned + totalStakeLost - totalInterestEarned + totalAtStake,
		}

		fmt.Println(address, initialBalance)
		err := dbClient.UpsertInitialStakeBalance(initialBalance)
		if err != nil {
			panic(err)
		}
	}
}

// fetchMetrics fetches the metrics for a given day from the metrics endpoint
func fetchMetrics(date time.Time) *MetricsSummary {
	request, err := http.NewRequest(
		"GET", mustEnv("STAKEINIT_METRICS_ENDPOINT")+"?date="+date.Format("2006-01-02"), nil,
	)
	if err != nil {
		panic(err)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
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
