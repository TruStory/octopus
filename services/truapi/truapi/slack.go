package truapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
)

type slackMessage struct {
	Text        string `json:"text"`
	UnfurlLinks bool   `json:"unfurl_links"`
}

func (ta *TruAPI) sendToSlack(text string, webhook string) {
	message := slackMessage{
		Text:        text,
		UnfurlLinks: true,
	}
	bz, err := json.Marshal(message)
	if err != nil {
		fmt.Println(err)
		return
	}
	// preparing the request
	slackRequest, err := http.NewRequest("POST", webhook, bytes.NewBuffer(bz))
	if err == nil {
		slackRequest.Header.Add("Content-Type", "application/json")

		// processing the request
		_, _ = ta.httpClient.Do(slackRequest)
	} else {
		fmt.Println(err)
	}
}

func (ta *TruAPI) sendClaimToSlack(c claim.Claim) {
	webhook := ta.APIContext.Config.App.SlackWebhook
	permalink := fmt.Sprintf("%s/claim/%d", ta.APIContext.Config.App.URL, c.ID)
	ta.sendToSlack(permalink, webhook)
}

func (ta *TruAPI) sendArgumentToSlack(argument staking.Argument) {
	webhook := ta.APIContext.Config.App.SlackWebhook
	permalink := fmt.Sprintf("%s/claim/%d/argument/%d", ta.APIContext.Config.App.URL, argument.ClaimID, argument.ID)
	ta.sendToSlack(permalink, webhook)
}

func (ta *TruAPI) sendCommentToSlack(comment db.Comment) {
	webhook := ta.APIContext.Config.App.SlackWebhook
	// Send new comment post to Slack
	permalink := fmt.Sprintf("%s/claim/%d", ta.APIContext.Config.App.URL, comment.ClaimID)
	if comment.ArgumentID != 0 && comment.ElementID != 0 {
		permalink = fmt.Sprintf("%s/argument/%d/element/%d", permalink, comment.ArgumentID, comment.ElementID)
	}
	permalink = fmt.Sprintf("%s/comment/%d", permalink, comment.ID)
	ta.sendToSlack(permalink, webhook)
}

func (ta *TruAPI) sendRequestTruToSlack(address string, userProfile db.UserProfile) {
	webhook := ta.APIContext.Config.App.RequestTruSlackWebhook
	permalink := fmt.Sprintf("%s/profile/%s", ta.APIContext.Config.App.URL, address)
	message := fmt.Sprintf("%s requested additional TRU\n\n%s", userProfile.Username, permalink)
	ta.sendToSlack(message, webhook)
}
