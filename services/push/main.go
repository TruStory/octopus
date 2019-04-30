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
	"github.com/machinebox/graphql"

	truchain "github.com/TruStory/truchain/types"
	db "github.com/TruStory/truchain/x/db"
	"github.com/sirupsen/logrus"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

type service struct {
	db        *db.Client
	apnsTopic string
	log       logrus.FieldLogger
	// gorush
	httpClient        *http.Client
	gorushHTTPAddress string
	// graphql
	graphqlClient *graphql.Client
}

func intPtr(i int) *int {
	return &i
}
func strPtr(s string) *string {
	return &s
}
func (s *service) sendNotification(notification PushNotification, tokens []string) (*GorushResponse, error) {
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

func (s *service) notificationSender(notifications <-chan *Notification, stop <-chan struct{}) {
	for {
		select {
		case notification := <-notifications:
			msg := notification.Msg
			title := "Story Update"
			receiverProfile, err := s.db.TwitterProfileByAddress(notification.To)
			if err != nil {
				s.log.WithError(err).Errorf("could not retrieve twitter profile for address %s", notification.To)
				continue
			}
			notificationEvent := &db.NotificationEvent{
				Address:          notification.To,
				TwitterProfileID: receiverProfile.ID,
				Read:             false,
				Timestamp:        time.Now(),
				Message:          msg,
				Type:             notification.Type,
				TypeID:           notification.TypeID,
			}
			var senderImage, senderAddress *string
			if notification.From != nil {
				profile, err := s.db.TwitterProfileByAddress(*notification.From)
				if err != nil {
					s.log.WithError(err).Errorf("could not retrieve twitter profile for address %s", *notification.From)
					continue
				}
				notificationEvent.SenderProfileID = profile.ID
				title = profile.Username
				senderImage = strPtr(profile.AvatarURI)
				senderAddress = strPtr(profile.Address)
			}
			_, err = s.db.Model(notificationEvent).Returning("*").Insert()
			if err != nil {
				s.log.WithError(err).Error("error saving event in database")
			}
			receiverAddress := notification.To
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
			pushNotification := PushNotification{
				Title: title,
				Body:  msg,
				NotificationData: NotificationData{
					ID:        notificationEvent.ID,
					TypeID:    notification.TypeID,
					Timestamp: notificationEvent.Timestamp,
					UserID:    senderAddress,
					Image:     senderImage,
					Read:      notificationEvent.Read,
					Type:      notificationEvent.Type,
				},
			}
			for p, t := range tokens {
				pushNotification.Platform = p
				r, err := s.sendNotification(pushNotification, t)
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

func (s *service) getStoryParticipants(storyID int64, creator, staker string) ([]string, error) {
	participants := make([]string, 0)
	req := graphql.NewRequest(StoryParticipantsQuery)
	req.Var("storyId", storyID)

	var res StoryParticipantsResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, req, &res); err != nil {
		return nil, err
	}
	mappedParticipants := make(map[string]bool)
	for _, b := range res.Story.Backings {
		if b.Creator.Address == creator || b.Creator.Address == staker {
			continue
		}
		mappedParticipants[b.Creator.Address] = true

	}
	for _, c := range res.Story.Challenges {
		if c.Creator.Address == creator || c.Creator.Address == staker {
			continue
		}
		mappedParticipants[c.Creator.Address] = true

	}
	if res.Story.Creator.Address != creator && res.Story.Creator.Address != staker {
		mappedParticipants[res.Story.Creator.Address] = true
	}
	for p := range mappedParticipants {
		participants = append(participants, p)
	}
	return participants, nil
}

func (s *service) processTransactionEvent(pushEvent types.EventDataTx, notifications chan<- *Notification) {
	pushData := &truchain.StakeNotificationResult{}
	err := json.Unmarshal(pushEvent.Result.Data, pushData)
	if err != nil {
		s.log.WithError(err).Error("error decoding transaction event")
		return
	}

	for _, tag := range pushEvent.Result.Tags {
		action := string(tag.Value)
		var alert string
		var enableParticipants bool
		var participantsAlert string
		var hideSender bool
		switch action {
		case "back_story":
			enableParticipants = true
			alert = "Backed your story"
			participantsAlert = "Backed a story you participated in"
		case "create_challenge":
			enableParticipants = true
			alert = "Challenged your story"
			participantsAlert = "Challenged a story you participated in"
		case "like_backing_argument":
			hideSender = true
			alert = fmt.Sprintf(
				"Someone endorsed your backing argument. You earned %s %s Cred",
				pushData.Cred.Amount.Quo(sdk.NewInt(truchain.Shanev)),
				strings.Title(pushData.Cred.Denom),
			)
		case "like_challenge_argument":
			hideSender = true
			alert = fmt.Sprintf(
				"Someone endorsed your challenge argument. You earned %s %s Cred",
				pushData.Cred.Amount.Quo(sdk.NewInt(truchain.Shanev)),
				strings.Title(pushData.Cred.Denom),
			)
		}
		if alert != "" {
			from := strPtr(pushData.From.String())
			to := pushData.To.String()
			if hideSender {
				from = nil
			}
			if pushData.From.String() != to {
				notifications <- &Notification{
					From:   from,
					To:     to,
					Msg:    alert,
					TypeID: pushData.StoryID,
					Type:   db.NotificationStoryAction,
				}
			}
			if participantsAlert != "" && enableParticipants {
				participants, err := s.getStoryParticipants(pushData.StoryID, to, pushData.From.String())
				if err != nil {
					s.log.WithError(err).Error("unable to get story participants")
				}
				for _, p := range participants {
					notifications <- &Notification{
						From:   from,
						To:     p,
						Msg:    participantsAlert,
						TypeID: pushData.StoryID,
						Type:   db.NotificationStoryAction,
					}
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

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
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
	notificationsCh := make(chan *Notification)
	cNotificationsCh := make(chan *CommentNotificationRequest)
	go s.startHTTP(stop, cNotificationsCh)
	go s.processCommentsNotifications(cNotificationsCh, notificationsCh)
	go s.notificationSender(notificationsCh, stop)
	for {
		select {
		case event := <-txsCh:
			switch v := event.(type) {
			case types.EventDataTx:
				s.processTransactionEvent(v, notificationsCh)
			case types.EventDataNewBlock:
				s.processNewBlockEvent(v, notificationsCh)
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
	graphqlEndpoint := mustEnv("PUSHD_GRAPHQL_ENDPOINT")

	dbClient := db.NewDBClient()
	graphqlClient := graphql.NewClient(graphqlEndpoint)
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
		graphqlClient:     graphqlClient,
	}

	srvc.run(quit)
}
