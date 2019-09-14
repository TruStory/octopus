package truapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type slackMessage struct {
	Text string `json:"text"`
}

func (ta *TruAPI) sendToSlack(text string) {
	message := slackMessage{
		Text: text,
	}
	bz, err := json.Marshal(message)
	if err != nil {
		fmt.Println(err)
		return
	}
	// preparing the request
	slackRequest, err := http.NewRequest("POST", ta.APIContext.Config.App.SlackWebhook, bytes.NewBuffer(bz))
	if err == nil {
		slackRequest.Header.Add("Content-Type", "application/json")

		// processing the request
		_, _ = ta.httpClient.Do(slackRequest)
	} else {
		fmt.Println(err)
	}
}
