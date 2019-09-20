package truapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (ta *TruAPI) sendCommentNotification(n CommentNotificationRequest) {
	if !ta.notificationsInitialized || ta.commentsNotificationsCh == nil {
		return
	}
	ta.commentsNotificationsCh <- n
}

func (ta *TruAPI) runCommentNotificationSender(notifications <-chan CommentNotificationRequest, endpoint string) {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(strings.TrimSpace(endpoint), "/"), "sendCommentNotification")

	for n := range notifications {
		claim := ta.claimResolver(ta.createContext(context.Background()), queryByClaimID{ID: uint64(n.ClaimID)})
		if claim.ID == 0 {
			fmt.Println("error retrieving claim id", n.ClaimID)
			continue
		}
		n.ClaimCreator = claim.Creator.String()
		if n.ArgumentID != 0 {
			argument := ta.claimArgumentResolver(ta.createContext(context.Background()), queryByArgumentID{ID: uint64(n.ArgumentID)})
			if argument.ID == 0 {
				fmt.Println("error retrieving argument id", n.ArgumentID)
				continue
			}
			n.ArgumentCreator = argument.Creator.String()
		}
		b, err := json.Marshal(&n)
		if err != nil {
			fmt.Println("error encoding comment notification request", err)
			continue
		}
		httpClient := &http.Client{
			Timeout: time.Second * 10,
		}
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
		request.Header.Add("Accept", "application/json")
		request.Header.Add("Content-Type", "application/json")
		resp, err := httpClient.Do(request)
		if err != nil {
			fmt.Println("error sending comment notification request", err)
			continue
		}
		// only read the status
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			fmt.Printf("error sending comment notification request status [%s] \n", resp.Status)
			continue
		}
		fmt.Printf("comment notification sent id[%d]\n", n.ID)
	}
}
