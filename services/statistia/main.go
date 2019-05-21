package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-pg/pg"

	db "github.com/TruStory/truchain/x/db"
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

	statistia.calculateBalance(from, to)
	return

	statistia.dbClient.RunInTransaction(func(tx *pg.Tx) error {
		// set date to starting date and keep adding 1 day to it as long as it comes before to
		for date := from; date.Before(to); date = date.AddDate(0, 0, 1) {
			err := statistia.seedInTxFor(tx, date)

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (statistia *service) calculateBalance(from, to time.Time) {
	today := time.Now()
	tomorrow := today.Add(24 * 1 * time.Hour)
	fmt.Println(tomorrow.Format("2006-01-02"))

	tmMetrics := statistia.fetchMetrics(tomorrow)

	userMetrics := tmMetrics.Users["cosmos1xqc5gwzpg3fyv5en2fzyx36z2se5ks33tt57e7"]

	var totalStakeEarned, totalStakeLost, totalInterestEarned, totalAtStake uint64
	for _, categoryMetric := range userMetrics.CategoryMetrics {
		totalStakeEarned += categoryMetric.Metrics.StakeEarned.Amount
		totalStakeLost += categoryMetric.Metrics.StakeLost.Amount
		totalInterestEarned += categoryMetric.Metrics.InterestEarned.Amount
		totalAtStake += categoryMetric.Metrics.TotalAmountAtStake.Amount
	}

	// initial balance = current balance - total earned + total lost - total interest earned + at stake
	initialBalance := int64(userMetrics.Balance.Amount - totalStakeEarned + totalStakeLost - totalInterestEarned + totalAtStake)

	fmt.Println(userMetrics.Balance.Amount, initialBalance)

	// set date to starting date and keep adding 1 day to it as long as it comes before to
	var dailyBalances = []int64{initialBalance}
	for date := from; date.Before(to); date = date.AddDate(0, 0, 1) {
		today := date
		yesterday := date.Add(-24 * 1 * time.Hour)

		yMetrics := statistia.fetchMetrics(yesterday)
		tMetrics := statistia.fetchMetrics(today)
		tUserMetrics := tMetrics.Users["cosmos1xqc5gwzpg3fyv5en2fzyx36z2se5ks33tt57e7"]

		totalStakeEarned, totalStakeLost, totalInterestEarned = 0, 0, 0
		for categoryID, tCategoryMetric := range tUserMetrics.CategoryMetrics {
			totalStakeEarned += tCategoryMetric.Metrics.StakeEarned.Amount
			totalStakeLost += tCategoryMetric.Metrics.StakeLost.Amount
			totalInterestEarned += tCategoryMetric.Metrics.InterestEarned.Amount

			yUserMetric, ok := yMetrics.Users["cosmos1xqc5gwzpg3fyv5en2fzyx36z2se5ks33tt57e7"]
			if ok {
				yCategoryMetric, ok := yUserMetric.CategoryMetrics[categoryID]
				if ok {
					totalStakeEarned -= yCategoryMetric.Metrics.StakeEarned.Amount
					totalStakeLost -= yCategoryMetric.Metrics.StakeLost.Amount
					totalInterestEarned -= yCategoryMetric.Metrics.InterestEarned.Amount
				}
			}
		}

		// daily balance = PREVIOUS BALANCE  + (total earned - total lost - total interest earned + at stake)
		// fmt.Println(dailyBalances[len(dailyBalances)-1], totalStakeEarned, totalStakeLost, totalInterestEarned, totalAtStake)
		dailyBalance := dailyBalances[len(dailyBalances)-1] + int64(totalStakeEarned-totalStakeLost+totalInterestEarned)
		dailyBalances = append(dailyBalances, dailyBalance)
		fmt.Println(dailyBalance)
	}

	fmt.Println(dailyBalances)
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
			dUserMetric := UserMetric{
				Address:        address,
				AsOnDate:       today,
				CategoryID:     categoryID,
				TotalClaims:    tCategoryMetric.Metrics.TotalClaims,
				TotalArguments: tCategoryMetric.Metrics.TotalArguments,
				// TotalClaimsBacked:         tCategoryMetric.Metrics.TotalClaimsBacked,
				// TotalClaimsChallenged:     tCategoryMetric.Metrics.TotalClaimsChallenged,
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
					// dUserMetric.TotalClaimsBacked = tCategoryMetric.Metrics.TotalClaimsBacked - yCategoryMetric.Metrics.TotalClaimsBacked
					// dUserMetric.TotalClaimsChallenged = tCategoryMetric.Metrics.TotalClaimsChallenged - yCategoryMetric.Metrics.TotalClaimsChallenged
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
