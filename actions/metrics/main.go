package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/bigquery"
)

func fetchMetrics(endpoint, date string) (*MetricsSummary, error) {
	client := &http.Client{
		Timeout: time.Second * 30,
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?date=%s", endpoint, date), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf("request failed %s", bodyString)

	}
	metricsSummary := &MetricsSummary{}
	err = json.NewDecoder(resp.Body).Decode(metricsSummary)
	if err != nil {
		return nil, err
	}
	return metricsSummary, nil
}

// ChainMetric ..
type ChainMetric struct {
	Date                 time.Time `bigquery:"date"`
	Address              string    `bigquery:"address"`
	Username             string    `bigquery:"username"`
	Category             string    `bigquery:"category"`
	TruStakeBalance      *big.Rat  `bigquery:"balance"`
	CredEarned           *big.Rat  `bigquery:"cred_earned"`
	Claims               int64     `bigquery:"claims_created"`
	ClaimsOpened         int64     `bigquery:"claims_opened"`
	UniqueClaimsOpened   int64     `bigquery:"unique_claims_opened"`
	Arguments            int64     `bigquery:"arguments_created"`
	EndorsementsGiven    int64     `bigquery:"endorsements_given"`
	EndorsementsReceived int64     `bigquery:"endorsements_received"`
	AmountEarned         *big.Rat  `bigquery:"amount_earned"`
	InterestEarned       *big.Rat  `bigquery:"interest_earned"`
	AmountLost           *big.Rat  `bigquery:"amount_lost"`
	AmountStaked         *big.Rat  `bigquery:"amount_staked"`
	AmountAtStake        *big.Rat  `bigquery:"amount_at_stake"`
}

// Save implements the ValueSaver interface.
func (cm *ChainMetric) Save() (map[string]bigquery.Value, string, error) {
	return map[string]bigquery.Value{
		"date":                  cm.Date,
		"address":               cm.Address,
		"username":              cm.Username,
		"category":              cm.Category,
		"balance":               cm.TruStakeBalance,
		"cred_earned":           cm.CredEarned,
		"claims_created":        cm.Claims,
		"claims_opened":         cm.ClaimsOpened,
		"unique_claims_opened":  cm.UniqueClaimsOpened,
		"arguments_created":     cm.Arguments,
		"endorsements_given":    cm.EndorsementsGiven,
		"endorsements_received": cm.EndorsementsReceived,
		"amount_earned":         cm.AmountEarned,
		"interest_earned":       cm.InterestEarned,
		"amount_lost":           cm.AmountLost,
		"amount_staked":         cm.AmountStaked,
		"amount_at_stake":       cm.AmountAtStake,
	}, "", nil
}

// Recreate recreates table
func Recreate(ctx context.Context, table *bigquery.Table) {
	schema, err := bigquery.InferSchema(ChainMetric{})
	if err != nil {
		log.Fatal("schema error", err)
	}
	table.Delete(ctx)
	if err := table.Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		log.Fatal("error creating table", err)
	}
}
func main() {
	metricsEndpoint := mustEnv("METRICS_ENDPOINT")
	metricsTable := mustEnv("METRICS_TABLE")
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "metrics-240714")
	if err != nil {
		log.Fatal("error creating client", err)
	}
	table := client.Dataset("trustory_metrics").Table(metricsTable)
	// recreate(ctx, table)
	// date, err := time.Parse("2006-01-02", "2019-05-23")
	// if err != nil {
	// 	log.Fatalf("error parsing %s", err)
	// }
	u := table.Inserter()
	items := make([]*ChainMetric, 0)

	defaultDate := time.Now().Format("2006-01-02")
	date := getEnv("METRICS_DATE", defaultDate)
	fmt.Println("running for date", date)
	metricsSummary, err := fetchMetrics(metricsEndpoint, date)
	if err != nil {
		log.Fatal(err)
	}

	timestamp, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatalf("error parsing %s", err)
	}

	for user, userMetrics := range metricsSummary.Users {
		for _, categoryMetrics := range userMetrics.CategoryMetrics {
			m := &ChainMetric{
				Date:                 timestamp.UTC(),
				Address:              user,
				Username:             userMetrics.UserName,
				Category:             categoryMetrics.CategoryName,
				TruStakeBalance:      new(big.Rat).SetInt(userMetrics.Balance.Amount.BigInt()),
				CredEarned:           new(big.Rat).SetInt(categoryMetrics.CredEarned.Amount.BigInt()),
				Claims:               categoryMetrics.Metrics.TotalClaims,
				ClaimsOpened:         categoryMetrics.Metrics.TotalOpenedClaims,
				UniqueClaimsOpened:   categoryMetrics.Metrics.TotalUniqueOpenedClaims,
				Arguments:            categoryMetrics.Metrics.TotalArguments,
				EndorsementsGiven:    categoryMetrics.Metrics.TotalEndorsementsGiven,
				EndorsementsReceived: categoryMetrics.Metrics.TotalEndorsementsReceived,
				AmountEarned:         new(big.Rat).SetInt(categoryMetrics.Metrics.StakeEarned.Amount.BigInt()),
				InterestEarned:       new(big.Rat).SetInt(categoryMetrics.Metrics.InterestEarned.Amount.BigInt()),
				AmountLost:           new(big.Rat).SetInt(categoryMetrics.Metrics.StakeLost.Amount.BigInt()),
				AmountStaked:         new(big.Rat).SetInt(categoryMetrics.Metrics.TotalAmountStaked.Amount.BigInt()),
				AmountAtStake:        new(big.Rat).SetInt(categoryMetrics.Metrics.TotalAmountAtStake.Amount.BigInt()),
			}
			items = append(items, m)
		}
	}

	err = u.Put(ctx, items)
	if err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}

	fmt.Println("no error inserted rows", len(items))
}
