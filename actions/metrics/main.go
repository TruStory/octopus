package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/bigquery"
)

func appendDaily(client *bigquery.Client, timestamp time.Time, sourceTable, destinationTable, dataset string) error {
	ctx := context.Background()
	yesterday := timestamp.Add(time.Duration(-1) * time.Hour * 24)
	query := strings.ReplaceAll(appendDailyQuery, ":source_table:", fmt.Sprintf("%s.%s", dataset, sourceTable))
	query = strings.Replace(query, ":end_date:", timestamp.Format("2006-01-02"), 1)
	query = strings.Replace(query, ":start_date:", yesterday.Format("2006-01-02"), 1)

	q := client.Query(query)
	q.QueryConfig.Dst = client.Dataset(dataset).Table(destinationTable)
	q.Location = "US"
	q.WriteDisposition = bigquery.WriteAppend
	fmt.Printf("Append daily results from %s to %s \n", sourceTable, destinationTable)
	fmt.Printf("Query:\n%s\n", query)
	job, err := q.Run(ctx)
	if err != nil {
		return err
	}
	_, err = job.Wait(ctx)
	if err != nil {
		return err
	}
	return nil
}
func main() {
	metricsEndpoint := mustEnv("METRICS_ENDPOINT")
	metricsTable := mustEnv("METRICS_TABLE")
	metricsDailyTable := mustEnv("METRICS_DAILY_TABLE")
	httpClient := &http.Client{
		Timeout: time.Minute * 5,
	}
	// date
	defaultDate := time.Now().Format("2006-01-02")
	date := getEnv("METRICS_DATE", defaultDate)
	timestamp, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Running metrics date %s table %s\n", date, metricsTable)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?date=%s", metricsEndpoint, date), nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	ctx := context.Background()
	bigQClient, err := bigquery.NewClient(ctx, "metrics-240714")
	if err != nil {
		log.Fatal("error creating client", err)
	}
	source := bigquery.NewReaderSource(resp.Body)
	source.AutoDetect = true   // Allow BigQuery to determine schema.
	source.SkipLeadingRows = 1 // CSV has a single header line.

	datasetID := "beta_metrics"
	loader := bigQClient.Dataset(datasetID).Table(metricsTable).LoaderFrom(source)
	loader.WriteDisposition = bigquery.WriteAppend
	job, err := loader.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
	status, err := job.Wait(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := status.Err(); err != nil {
		log.Fatal(err)
	}
	err = appendDaily(bigQClient, timestamp, metricsTable, metricsDailyTable, datasetID)
	if err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}

}
