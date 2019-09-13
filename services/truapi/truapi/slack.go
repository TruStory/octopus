package truapi

import (
	"bytes"
	"fmt"
	"net/http"
)

func (ta *TruAPI) sendToSlack(payload []byte) {
	// preparing the request
	slackRequest, err := http.NewRequest("POST", ta.APIContext.Config.App.SlackWebhook, bytes.NewBuffer(payload))
	if err == nil {
		slackRequest.Header.Add("Content-Type", "application/json")

		// processing the request
		_, _ = ta.httpClient.Do(slackRequest)
	} else {
		fmt.Println(err)
	}
}
