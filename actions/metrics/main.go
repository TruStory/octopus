package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	truchain "github.com/TruStory/truchain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	option "google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
	"gopkg.in/Iwark/spreadsheet.v2"
)

func fetchMetrics(endpoint, date string) (*MetricsSummary, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
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

var decShanev = sdk.NewDecFromInt(sdk.NewInt(truchain.Shanev))

func toShanev(coin sdk.Coin) sdk.Dec {
	return sdk.NewDecFromInt(coin.Amount).Quo(decShanev)
}
func main() {
	metricsEndpoint := mustEnv("METRICS_ENDPOINT")
	secretFile := mustEnv("METRICS_SECRET_FILE")
	spreadsheetID := mustEnv("METRICS_SPREADSHEET_ID")
	spreadsheetRange := mustEnv("METRICS_SPREADSHEET_RANGE")
	data, err := ioutil.ReadFile(secretFile)
	if err != nil {
		log.Fatal(err)
	}
	conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	client := conf.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	defaultDate := time.Now().Format("2006-01-02")
	date := getEnv("METRICS_DATE", defaultDate)
	metricsSummary, err := fetchMetrics(metricsEndpoint, date)
	if err != nil {
		log.Fatal(err)
	}
	values := make([][]interface{}, 0)
	for user, userMetrics := range metricsSummary.Users {
		for _, categoryMetrics := range userMetrics.CategoryMetrics {
			row := []interface{}{
				date,
				user,
				userMetrics.UserName,
				toShanev(userMetrics.Balance),
				categoryMetrics.CategoryName,
				toShanev(categoryMetrics.CredEarned),
				categoryMetrics.Metrics.TotalClaims,
				categoryMetrics.Metrics.TotalArguments,
				categoryMetrics.Metrics.TotalEndorsementsGiven,
				categoryMetrics.Metrics.TotalEndorsementsReceived,
				toShanev(categoryMetrics.Metrics.TotalAmountStaked),
				toShanev(categoryMetrics.Metrics.StakeEarned),
				toShanev(categoryMetrics.Metrics.InterestEarned),
				toShanev(categoryMetrics.Metrics.StakeLost),
				toShanev(categoryMetrics.Metrics.TotalAmountAtStake),
				toShanev(categoryMetrics.Metrics.TotalAmountBacked),
				toShanev(categoryMetrics.Metrics.TotalAmountChallenged),
			}
			values = append(values, row)
		}

	}
	_, err = srv.Spreadsheets.Values.Append(spreadsheetID, spreadsheetRange, &sheets.ValueRange{
		MajorDimension: "ROWS",
		Values:         values,
	}).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Do()
	if err != nil {
		log.Fatal(err)
	}
}
