package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {

	yesterday := time.Now().Add(-24 * 5 * time.Hour).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	fmt.Printf("%v -- %v\n\n", yesterday, today)

	yesterdayMetrics := fetchMetrics(yesterday)
	todayMetrics := fetchMetrics(today)
	// dailyMetrics := &MetricsSummary{
	// 	Users: make(map[string]*UserMetrics),
	// }
	for address, todayMetric := range todayMetrics.Users {
		for categoryID, todayCategoryMetric := range todayMetric.CategoryMetrics {
			yesterdayMetric, ok := yesterdayMetrics.Users[address]
			if ok {
				yesterdayCategoryMetric, ok := yesterdayMetric.CategoryMetrics[categoryID]
				if ok {
					fmt.Printf("\n\n%v\n%v", todayCategoryMetric, yesterdayCategoryMetric)
					dailyCategoryMetric := &Metrics{
						TotalClaims:               todayCategoryMetric.Metrics.TotalClaims - yesterdayCategoryMetric.Metrics.TotalClaims,
						TotalArguments:            todayCategoryMetric.Metrics.TotalArguments - yesterdayCategoryMetric.Metrics.TotalArguments,
						TotalGivenEndorsements:    todayCategoryMetric.Metrics.TotalGivenEndorsements - yesterdayCategoryMetric.Metrics.TotalGivenEndorsements,
						TotalReceivedEndorsements: todayCategoryMetric.Metrics.TotalReceivedEndorsements - yesterdayCategoryMetric.Metrics.TotalReceivedEndorsements,
						TotalAmountStaked:         todayCategoryMetric.Metrics.TotalAmountStaked.Minus(yesterdayCategoryMetric.Metrics.TotalAmountStaked),
						TotalAmountAtStake:        todayCategoryMetric.Metrics.TotalAmountAtStake.Minus(yesterdayCategoryMetric.Metrics.TotalAmountAtStake),
						StakeEarned:               todayCategoryMetric.Metrics.StakeEarned.Minus(yesterdayCategoryMetric.Metrics.StakeEarned),
						StakeLost:                 todayCategoryMetric.Metrics.StakeLost.Minus(yesterdayCategoryMetric.Metrics.StakeLost),
						InterestEarned:            todayCategoryMetric.Metrics.InterestEarned.Minus(yesterdayCategoryMetric.Metrics.InterestEarned),
					}

					fmt.Printf("\nUser -- %v\nMetrics -- %v\nPer Day -- %v\n\n", address, todayMetric, dailyCategoryMetric)
				}

			}
		}
	}
}

func fetchMetrics(date string) *MetricsSummary {
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

	metrics := &MetricsSummary{
		Users: make(map[string]*UserMetrics),
	}
	err = json.Unmarshal(body, &metrics)
	if err != nil {
		panic(err)
	}

	return metrics
}
