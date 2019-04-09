package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/appleboy/gorush/gorush"

	"github.com/sirupsen/logrus"

	truchain "github.com/TruStory/truchain/types"
	db "github.com/TruStory/truchain/x/db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

// ChainEvent represents a parsed event comming from the chain.
type ChainEvent struct {
	From    *sdk.AccAddress
	To      sdk.AccAddress
	Msg     string
	StoryID int64
}

// NotificationData represents the data relevant to the app.
type NotificationData struct {
	ID int64 `json:"id"`
	// StoryID
	TypeID    int64               `json:"typeId"`
	Timestamp time.Time           `json:"timestamp"`
	Read      bool                `json:"read"`
	Type      db.NotificationType `json:"type"`
	// UserID is the sender
	UserID *string `json:"userId,omitempty"`
	Image  *string `json:"image,omitempty"`
}

// ToGorushData translate to gorush data format.
func (d NotificationData) ToGorushData() gorush.D {
	data := make(map[string]interface{})
	// embbed everything inside a trustory key
	data["trustory"] = d
	return data
}

// Notification represents the notification data sent to services.
type Notification struct {
	Title            string
	Body             string
	Platform         string
	NotificationData NotificationData
}

// GorushResponse represents a json payload response.
type GorushResponse struct {
	Success string                `json:"success"`
	Counts  int                   `json:"counts"`
	Logs    []gorush.LogPushEntry `json:"logs"`
}

type service struct {
	db        *db.Client
	apnsTopic string
	log       logrus.FieldLogger
	// gorush
	httpClient        *http.Client
	gorushHTTPAddress string
}

func intPtr(i int) *int {
	return &i
}
func strPtr(s string) *string {
	return &s
}
func (s *service) sendNotification(notification Notification, tokens []string) (*GorushResponse, error) {
	var p int
	if notification.Platform == "ios" {
		p = 1
	}
	// TODO: Enable when android is supported
	// if notification.Platform == "android" {
	// 	p = 2
	// }
	if p == 0 {
		return nil, fmt.Errorf("platform not supported")
	}
	pushNotification := gorush.PushNotification{
		Platform: p,
		Tokens:   tokens,
		Badge:    intPtr(1),
		Topic:    s.apnsTopic,
		Sound:    "default",
		Alert: gorush.Alert{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: notification.NotificationData.ToGorushData(),
	}
	n := &gorush.RequestPush{
		Notifications: []gorush.PushNotification{pushNotification},
	}
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(n)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodPost, s.gorushHTTPAddress, b)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	gorushResp := &GorushResponse{}
	err = json.NewDecoder(resp.Body).Decode(gorushResp)
	if err != nil {
		return nil, err
	}
	return gorushResp, err
}

func (s *service) notificationSender(chainEvents <-chan *ChainEvent, stop <-chan struct{}) {
	for {
		select {
		case chainEvent := <-chainEvents:
			msg := chainEvent.Msg
			title := "Story Update"
			receiverProfile, err := s.db.TwitterProfileByAddress(chainEvent.To.String())
			if err != nil {
				s.log.WithError(err).Errorf("could not retrieve twitter profile for address %s", chainEvent.To.String())
				continue
			}
			notificationEvent := &db.NotificationEvent{
				Address:          chainEvent.To.String(),
				TwitterProfileID: receiverProfile.ID,
				Read:             false,
				Timestamp:        time.Now(),
				Message:          msg,
				Type:             db.NotificationStoryAction,
				TypeID:           chainEvent.StoryID,
			}
			var senderImage, senderAddress *string
			if chainEvent.From != nil {
				profile, err := s.db.TwitterProfileByAddress(chainEvent.From.String())
				if err != nil {
					s.log.WithError(err).Errorf("could not retrieve twitter profile for address %s", chainEvent.From.String())
					continue
				}
				notificationEvent.SenderProfileID = profile.ID
				title = profile.FullName
				senderImage = strPtr(profile.AvatarURI)
				senderAddress = strPtr(profile.Address)
			}
			_, err = s.db.Model(notificationEvent).Returning("*").Insert()
			if err != nil {
				s.log.WithError(err).Error("error saving event in database")
			}
			receiverAddress := chainEvent.To.String()
			deviceTokens, err := s.db.DeviceTokensByAddress(receiverAddress)
			if err != nil {
				s.log.WithError(err).Error("error retrieving tokens from db")
				continue
			}
			if len(deviceTokens) == 0 {
				s.log.Infof("account address %s doesn't not have push notification tokens \n", receiverAddress)
				continue
			}
			tokens := make(map[string][]string)
			for _, deviceToken := range deviceTokens {
				currentTokens := tokens[deviceToken.Platform]
				tokens[deviceToken.Platform] = append(currentTokens, deviceToken.Token)
			}
			notification := Notification{
				Title: title,
				Body:  msg,
				NotificationData: NotificationData{
					ID:        notificationEvent.ID,
					TypeID:    chainEvent.StoryID,
					Timestamp: notificationEvent.Timestamp,
					UserID:    senderAddress,
					Image:     senderImage,
					Read:      notificationEvent.Read,
					Type:      notificationEvent.Type,
				},
			}
			for p, t := range tokens {
				notification.Platform = p
				r, err := s.sendNotification(notification, t)
				if err != nil {
					s.log.WithError(err).Error("error sending notifications")
					continue
				}
				if r != nil {
					s.log.Infof("notifications sent - status : %s count : %d", r.Success, r.Counts)
				}

			}

		case <-stop:
			s.log.Info("stopping notification sender")
			return
		}
	}
}

func (s *service) processTransactionEvent(pushEvent types.EventDataTx, events chan<- *ChainEvent) {
	pushData := &truchain.StakeNotificationResult{}
	err := json.Unmarshal(pushEvent.Result.Data, pushData)
	if err != nil {
		s.log.WithError(err).Error("error decoding transaction event")
		return
	}

	for _, tag := range pushEvent.Result.Tags {
		action := string(tag.Value)
		var alert string
		switch action {
		case "back_story":
			alert = "Backed your story"
		case "create_challenge":
			alert = "Challenged your story"
		case "like_backing_argument":
			alert = "Endorsed your backing argument"
		case "like_challenge_argument":
			alert = "Endorsed your challenge argument"
		}
		if alert != "" {
			stake := pushData.Amount.Amount.Quo(sdk.NewInt(truchain.Shanev))

			alert = fmt.Sprintf("%s with %s TruStake", alert, stake)
			if pushData.From.String() == pushData.To.String() {
				continue
			}
			events <- &ChainEvent{From: &pushData.From, To: pushData.To, Msg: alert, StoryID: pushData.StoryID}
		}
	}
}

func (s *service) processNewBlockEvent(newBlockEvent types.EventDataNewBlock, events chan<- *ChainEvent) {
	for _, tag := range newBlockEvent.ResultEndBlock.Tags {
		if string(tag.Key) == "tru.event.completedStories" {
			completed := &truchain.CompletedStoriesNotificationResult{}
			err := json.Unmarshal(tag.Value, completed)
			if err != nil {
				s.log.WithError(err).Error("error decoding completed stories")
				continue
			}
			for _, story := range completed.Stories {
				events <- &ChainEvent{To: story.Creator, Msg: "A claim you made has completed", StoryID: story.ID}
				for _, backer := range story.Backers {
					if story.Creator.String() == backer.String() {
						continue
					}
					events <- &ChainEvent{To: backer, Msg: "A claim you backed has completed", StoryID: story.ID}
				}
				for _, challenger := range story.Challengers {
					if story.Creator.String() == challenger.String() {
						continue
					}
					events <- &ChainEvent{To: challenger, Msg: "A claim you challenged has completed", StoryID: story.ID}
				}
			}
		}
	}
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}

func setupSignals() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
	}()
	return stop
}

func (s *service) logChainStatus(c *client.HTTP) {
	if c == nil {
		return
	}
	status, err := c.Status()
	if err != nil {
		s.log.WithError(err).Error("error connecting to chain")
		return
	}
	if status != nil {
		nodeInfo := status.NodeInfo
		s.log.Infof("connected to [%s] address: %s", nodeInfo.Moniker, nodeInfo.NetAddress().String())
	}

}
func (s *service) run(stop <-chan struct{}) {

	remote := getEnv("REMOTE_ENDPOINT", "tcp://0.0.0.0:26657")
	client := client.NewHTTP(remote, "/websocket")
	tmQuery := query.MustParse("tru.event = 'Push'")
	err := client.Start()
	if err != nil {
		s.log.WithError(err).Fatal("error starting client")
	}
	defer func() {
		// program is exiting
		_ = client.Stop()
	}()

	// fail fast and let the service restart
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	txsCh := make(chan interface{})
	err = client.Subscribe(ctx, "trustory-push-client", tmQuery, txsCh)
	if err != nil {
		s.log.WithError(err).Fatal("could not connect to remote endpoint")
	}
	s.logChainStatus(client)
	s.log.Infof("subscribing to query event %s", tmQuery.String())
	chainEventsCh := make(chan *ChainEvent)
	go s.notificationSender(chainEventsCh, stop)
	for {
		select {
		case event := <-txsCh:
			switch v := event.(type) {
			case types.EventDataTx:
				s.processTransactionEvent(v, chainEventsCh)
			case types.EventDataNewBlock:
				s.processNewBlockEvent(v, chainEventsCh)
			}
		case <-stop:
			// program is exiting
			_ = client.Stop()
			s.log.Info("service stopped")
			return
		case <-time.After(30 * time.Second):
			s.logChainStatus(client)

		}
	}
}

func main() {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	gorushHTTPAddress := getEnv("GORUSH_ADDRESS", "http://localhost:9000/api/push")
	topic := getEnv("NOTIFICATION_TOPIC", "io.trustory.app.devnet")

	dbClient := db.NewDBClient()

	log.Info("pushd connected to db and starting")

	quit := setupSignals()
	srvc := &service{
		apnsTopic: topic,
		db:        dbClient,
		log:       log,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
		gorushHTTPAddress: gorushHTTPAddress,
	}

	srvc.run(quit)
}
