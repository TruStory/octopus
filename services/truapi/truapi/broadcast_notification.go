package truapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (ta *TruAPI) sendBroadcastNotification(n BroadcastNotificationRequest) {
	if !ta.notificationsInitialized || ta.broadcastNotificationsCh == nil {
		return
	}
	ta.broadcastNotificationsCh <- n
}

func (ta *TruAPI) runBroadcastNotificationSender(notifications <-chan BroadcastNotificationRequest, pushEndpoint string) {
	pushURL := fmt.Sprintf("%s/%s", strings.TrimRight(strings.TrimSpace(pushEndpoint), "/"), "sendBroadcastNotification")

	for n := range notifications {
		httpClient := &http.Client{
			Timeout: time.Second * 10,
		}
		b, err := json.Marshal(&n)
		if err != nil {
			fmt.Println("error encoding broadcast notification request", err)
			continue
		}
		request, err := http.NewRequest(http.MethodPost, pushURL, bytes.NewBuffer(b))
		if err != nil {
			fmt.Println("error creating http request", err)
		}
		request.Header.Add("Accept", "application/json")
		request.Header.Add("Content-Type", "application/json")
		resp, err := httpClient.Do(request)
		if err != nil {
			fmt.Println("error sending broadcast notification request", err)
			continue
		}
		// only read the status
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			fmt.Printf("error sending broadcast notification request status [%s] \n", resp.Status)
			continue
		}
		fmt.Printf("broadcast notification sent type[%d]\n", n.Type)
	}
}
