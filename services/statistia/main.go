package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-pg/pg"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	db "github.com/TruStory/octopus/services/truapi/db"
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
		port:            getEnv("STATISTIA_PORT", "6284"),
		metricsEndpoint: mustEnv("STATISTIA_METRICS_ENDPOINT"),
		router:          mux.NewRouter(),
		httpClient:      &http.Client{},
		dbClient:        db.NewDBClient(dbConfig),
	}
	defer statistia.dbClient.Close()

	args := os.Args[1:]

	if len(args) == 0 {
		// Running as a background service - go run *.go
		statistia.run()
	} else if len(args) == 2 {
		// Running as the historical seeder - go run *.go 2019-04-14 today
		// OR running as daily cron job - go run *.go today today
		var from, to time.Time
		var err error

		if args[0] == "today" {
			from = time.Now()
		} else {
			from, err = time.Parse("2006-01-02", args[0])
			if err != nil {
				panic(err)
			}
		}

		if args[1] == "today" {
			to = time.Now()
		} else {
			to, err = time.Parse("2006-01-02", args[1])
			if err != nil {
				panic(err)
			}
		}

		statistia.seedBetween(from, to)
	}
}

func (statistia *service) run() {
	http.Handle("/", statistia.router)

	fmt.Printf("\nRunning on... %s\n", "http://0.0.0.0:"+statistia.port)
	err := http.ListenAndServe(":"+statistia.port, nil)
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

// seedBetween seeds the user daily metrics between the two given dates
func (statistia *service) seedBetween(from, to time.Time) {
	// adding one day because date.Before checks "<" and not "<="
	to.Add(24 * 1 * time.Hour)

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
	fmt.Println(tMetrics)
	fmt.Printf("Seeding for... %s in comparison with...%s\n", today.Format("2006-01-02"), yesterday.Format("2006-01-02"))

	for address, tUserMetric := range tMetrics.Users {
		fmt.Printf("\tCalculating for User... %s\n", address)
		for categoryID, tCategoryMetric := range tUserMetric.CategoryMetrics {
			fmt.Printf("\t\tCalculating for Category... %s\n", tCategoryMetric.CategoryName)

			// by default, assume that the user has no previous activity,
			// thus, today's metrics become the daily metrics,
			// thus, creating and initializing a default struct.
			dUserMetric := UserMetric{
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
				StakeBalance:              tUserMetric.Balance.Amount,
				CredEarned:                tCategoryMetric.CredEarned.Amount,
			}

			// if any activity is found on the previous day,
			// we'll calculate the difference to get the given day's metrics.
			yUserMetric, ok := yMetrics.Users[address]
			if ok {
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

func (statistia *service) saveMetrics(tx *pg.Tx, metrics UserMetric) error {
	err := UpsertDailyUserMetricInTx(tx, metrics)
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

	metrics := &MetricsSummary{
		Users: make(map[string]*UserMetrics),
	}
	err = json.NewDecoder(response.Body).Decode(metrics)
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
