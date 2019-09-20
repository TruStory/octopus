package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	app "github.com/TruStory/octopus/services/truapi/truapi"
)

func sendNotification(n app.RewardNotificationRequest) {
	url := fmt.Sprintf("%s/%s", getEnv("ENDPOINT_NOTIFICATION", "http://localhost:9001"), "sendRewardNotification")

	b, err := json.Marshal(&n)
	if err != nil {
		fmt.Println("error encoding reward notification request", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Println("error sending reward notification request", err)
	}
	// only read the status
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		fmt.Printf("error sending reward notification request status [%s] \n", resp.Status)
	}
	fmt.Printf("reward notification sent to user with id[%d]\n", n.RewardeeID)
}
