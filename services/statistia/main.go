package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/TruStory/truchain/x/db"
	"github.com/gorilla/mux"
)

type service struct {
	port            string
	metricsEndpoint string
	router          *mux.Router
	httpClient      *http.Client
	dbClient        *db.Client
}

func main() {
	statistia := &service{
		port:            getEnv("STATISTIA_PORT", "6284"),
		metricsEndpoint: mustEnv("STATISTIA_METRICS_ENDPOINT"),
		router:          mux.NewRouter(),
		httpClient:      &http.Client{},
		dbClient:        db.NewDBClient(),
	}

	// Daily cronjob - go run *.go today
	// Historical seeding - go run *.go 2019-04-14--today
	args := os.Args[1:]

	if len(args) == 1 && args[0] == "today" {
		today := time.Now()
		seed(statistia, today)
	}
}

func seed(statistia *service, date time.Time) {
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
			dUserMetric := DailyUserMetric{
				Address:                   address,
				AsOnDate:                  today,
				CategoryID:                categoryID,
				TotalClaims:               tCategoryMetric.Metrics.TotalClaims,
				TotalArguments:            tCategoryMetric.Metrics.TotalArguments,
				TotalEndorsementsGiven:    tCategoryMetric.Metrics.TotalGivenEndorsements,
				TotalEndorsementsReceived: tCategoryMetric.Metrics.TotalReceivedEndorsements,
				TotalAmountStaked:         tCategoryMetric.Metrics.TotalAmountStaked.Amount,
				TotalAmountAtStake:        tCategoryMetric.Metrics.TotalAmountAtStake.Amount,
				StakeEarned:               tCategoryMetric.Metrics.StakeEarned.Amount,
				StakeLost:                 tCategoryMetric.Metrics.StakeLost.Amount,
				InterestEarned:            tCategoryMetric.Metrics.InterestEarned.Amount,
				StakeBalance:              tUserMetric.Balance.Amount,
			}

			// if any activity is found on the previous day,
			// we'll calculate the difference to get the given day's metrics.
			yUserMetric, ok := yMetrics.Users[address]
			if ok {
				yCategoryMetric, ok := yUserMetric.CategoryMetrics[categoryID]
				if ok {
					dUserMetric.TotalClaims = tCategoryMetric.Metrics.TotalClaims - yCategoryMetric.Metrics.TotalClaims
					dUserMetric.TotalArguments = tCategoryMetric.Metrics.TotalArguments - yCategoryMetric.Metrics.TotalArguments
					dUserMetric.TotalEndorsementsGiven = tCategoryMetric.Metrics.TotalGivenEndorsements - yCategoryMetric.Metrics.TotalGivenEndorsements
					dUserMetric.TotalEndorsementsReceived = tCategoryMetric.Metrics.TotalReceivedEndorsements - yCategoryMetric.Metrics.TotalReceivedEndorsements
					dUserMetric.TotalAmountStaked = tCategoryMetric.Metrics.TotalAmountStaked.Minus(yCategoryMetric.Metrics.TotalAmountStaked).Amount
					dUserMetric.TotalAmountAtStake = tCategoryMetric.Metrics.TotalAmountAtStake.Minus(yCategoryMetric.Metrics.TotalAmountAtStake).Amount
					dUserMetric.StakeEarned = tCategoryMetric.Metrics.StakeEarned.Minus(yCategoryMetric.Metrics.StakeEarned).Amount
					dUserMetric.StakeLost = tCategoryMetric.Metrics.StakeLost.Minus(yCategoryMetric.Metrics.StakeLost).Amount
					dUserMetric.InterestEarned = tCategoryMetric.Metrics.InterestEarned.Minus(yCategoryMetric.Metrics.InterestEarned).Amount
				}
			}

			fmt.Printf("\t\tSaving...\n")
			err := statistia.saveMetrics(dUserMetric)
			if err != nil {
				fmt.Printf("\t\tSaving FAILED\n")
				panic(err)
			}
		}
	}
}

func (statistia *service) saveMetrics(metrics DailyUserMetric) error {
	err := UpsertDailyUserMetric(statistia.dbClient, metrics)
	if err != nil {
		return err
	}

	return nil
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

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	metrics := &MetricsSummary{
		Users: make(map[string]*UserMetrics),
	}
	err = json.Unmarshal(body, &metrics)
	if err != nil {
		panic(err)
	}

	return metrics
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}
