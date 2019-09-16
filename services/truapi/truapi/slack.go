package truapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
	stripmd "github.com/writeas/go-strip-markdown"
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

func (ta *TruAPI) sendClaimToSlack(c claim.Claim) {
	permalink := fmt.Sprintf("%s/claim/%d", ta.APIContext.Config.App.URL, c.ID)
	twitterProfile, err := ta.DBClient.TwitterProfileByAddress(c.Creator.String())
	if err == nil {
		payload := fmt.Sprintf("*New claim posted by %s in %s:*\n\n>%s\n\n<%s>", twitterProfile.Username, c.CommunityID, blockquote(c.Body), permalink)
		ta.sendToSlack(payload)
	}
}

func (ta *TruAPI) sendArgumentToSlack(argument staking.Argument) {
	permalink := fmt.Sprintf("%s/claim/%d/argument/%d", ta.APIContext.Config.App.URL, argument.ClaimID, argument.ID)
	body, err := ta.DBClient.TranslateToUsersMentions(argument.Body)
	if err != nil {
		body = argument.Body
	}
	twitterProfile, err := ta.DBClient.TwitterProfileByAddress(argument.Creator.String())
	if err == nil {
		payload := fmt.Sprintf("*New argument posted by %s:*\n\n>TLDR: %s\n\n>%s\n\n<%s>", twitterProfile.Username, blockquote(argument.Summary), blockquote(stripmd.Strip(body)), permalink)
		ta.sendToSlack(payload)
	}
}

func (ta *TruAPI) sendCommentToSlack(comment db.Comment) {
	// Send new comment post to Slack
	permalink := fmt.Sprintf("%s/claim/%d", ta.APIContext.Config.App.URL, comment.ClaimID)
	if comment.ArgumentID != 0 && comment.ElementID != 0 {
		permalink = fmt.Sprintf("%s/argument/%d/element/%d", permalink, comment.ArgumentID, comment.ElementID)
	}
	permalink = fmt.Sprintf("%s/comment/%d", permalink, comment.ID)
	body, err := ta.DBClient.TranslateToUsersMentions(comment.Body)
	if err != nil {
		body = comment.Body
	}
	twitterProfile, err := ta.DBClient.TwitterProfileByAddress(comment.Creator)
	if err == nil {
		// preparing the request
		payload := fmt.Sprintf("*New comment posted by %s:*\n\n>%s\n\n<%s>", twitterProfile.Username, blockquote(stripmd.Strip(body)), permalink)
		ta.sendToSlack(payload)
	}
}

func blockquote(text string) string {
	return strings.Replace(text, "\n", "\n>", -1)
}
