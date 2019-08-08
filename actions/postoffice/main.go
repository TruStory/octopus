package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/TruStory/octopus/actions/postoffice/campaigns"

	"github.com/TruStory/octopus/services/truapi/postman"
)

var registry = make(map[string]campaigns.Campaign)

func init() {
	registry["waitlist-approval"] = (*campaigns.WaitlistApprovalCampaign)(nil)
}

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("please pass the campaign name and the email address from which the emails must be sent. eg. go run *.go waitlist-approval preethi@trustory.io")
		os.Exit(1)
	}

	client, err := postman.NewVanillaPostman("us-west-2", args[1])
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	campaign, ok := registry[args[0]]
	if !ok {
		log.Fatal(errors.New("no such campaign found in the registry"))
		os.Exit(1)
	}

	for _, recipient := range campaign.GetRecipients() {
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
		fmt.Printf(" ✅\n")
	}
}
