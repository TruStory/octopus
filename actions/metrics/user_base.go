package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
)

func userBase() {
	fmt.Println("Running user base")
	metricsEndpoint := mustEnv("METRICS_ENDPOINT")
	metricsUsersTable := mustEnv("METRICS_USER_BASE_TABLE")

	httpClient := &http.Client{
		Timeout: time.Minute * 5,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/user_base", metricsEndpoint), nil)
	req.Header.Set("Metrics-Secret", mustEnv("METRICS_SECRET"))
	if err != nil {
		log.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Error running user base", resp.Status)
	}
	ctx := context.Background()
	bigQClient, err := bigquery.NewClient(ctx, "metrics-240714")
	if err != nil {
		log.Fatal("error creating client", err)
	}
	source := bigquery.NewReaderSource(resp.Body)
	source.AutoDetect = true   // Allow BigQuery to determine schema.
	source.SkipLeadingRows = 1 // CSV has a single header line.
	source.AllowQuotedNewlines = true

	datasetID := "beta_metrics"
	loader := bigQClient.Dataset(datasetID).Table(metricsUsersTable).LoaderFrom(source)
	loader.WriteDisposition = bigquery.WriteTruncate

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
}
