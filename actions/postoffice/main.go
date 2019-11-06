package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/TruStory/octopus/actions/postoffice/campaigns"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

var registry = make(map[string]campaigns.Campaign)

func init() {
	registry["waitlist-approval"] = (*campaigns.WaitlistApprovalCampaign)(nil)
	registry["verification-attempt"] = (*campaigns.VerificationAttemptCampaign)(nil)
}

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("please pass the campaign name and the email address from which the emails must be sent. eg. go run *.go waitlist-approval community@trustory.io")
		os.Exit(1)
	}

	client, err := postman.NewVanillaPostman("us-west-2", args[1], mustEnv("AWS_ACCESS_KEY"), mustEnv("AWS_ACCESS_SECRET"))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	campaign, ok := registry[args[0]]
	if !ok {
		log.Fatal(errors.New("no such campaign found in the registry"))
		os.Exit(1)
	}

	dbPort, err := strconv.Atoi(getEnv("PG_PORT", "5432"))
	if err != nil {
		log.Fatalln(err)
	}
	config := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_HOST", "localhost"),
			Port: dbPort,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}
	dbClient := db.NewDBClient(config)

	recipients, err := campaign.GetRecipients(dbClient)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	for _, recipient := range recipients {
		fmt.Printf("Sending email to... %s", recipient.Email)

		message, err := campaign.GetMessage(client, recipient)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		if err = client.Deliver(*message); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		fmt.Printf("... post processing...")
		err = campaign.RunPostProcess(dbClient, recipient)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		fmt.Printf(" ✅\n")
	}
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
