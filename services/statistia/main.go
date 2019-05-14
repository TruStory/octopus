package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

func main() {

	yesterday := time.Now().Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	yesterdayMetrics := fetchMetrics(yesterday)
	todayMetrics := fetchMetrics(today)
	for address, todayMetric := range todayMetrics.Users {
		yesterdayMetric, ok := yesterdayMetrics.Users[address]
		if ok {
			perDayMetric := &UserMetrics{
				TotalClaims: big.NewInt(0).Sub(
					big.NewInt(todayMetric.TotalClaims), big.NewInt(yesterdayMetric.TotalClaims),
				).Int64(),
				TotalArguments: big.NewInt(0).Sub(
					big.NewInt(todayMetric.TotalArguments), big.NewInt(yesterdayMetric.TotalArguments),
				).Int64(),
				TotalGivenEndorsements: big.NewInt(0).Sub(
					big.NewInt(todayMetric.TotalGivenEndorsements), big.NewInt(yesterdayMetric.TotalGivenEndorsements),
				).Int64(),
			}

			fmt.Printf("\nUser -- %v\nMetrics -- %v\nPer Day -- %v\n\n", address, todayMetric, perDayMetric)
		}
	}
}

func fetchMetrics(date string) *Metrics {
	client := &http.Client{}
	request, err := http.NewRequest("GET", "http://localhost:1337/api/v1/metrics?date="+date, nil)
	if err != nil {
		panic(err)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	metrics := &Metrics{
		Users: make(map[string]*UserMetrics),
	}
	err = json.Unmarshal(body, &metrics)
	if err != nil {
		panic(err)
	}

	return metrics
}
