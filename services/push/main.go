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
	"github.com/sirupsen/logrus"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
	stripmd "github.com/writeas/go-strip-markdown"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
)

const (
	// BodyMaxLength for notification
	BodyMaxLength = 185
)

func intPtr(i int) *int {
	return &i
}

func uint64Ptr(i uint64) *int64 {
	n := int64(i)
	return &n
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
			Title:    notification.Title,
			Subtitle: notification.Subtitle,
			Body:     notification.Body,
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
			title := notification.Type.String()
			receiver, err := s.db.UserByAddress(notification.To)
			if err != nil {
				s.log.WithError(err).Errorf("could not retrieve user for address %s", notification.To)
				continue
			}
			if receiver == nil {
				s.log.Warnf("profile doesn't exist for  %s", notification.To)
				continue
			}
			if notification.Trim && len(msg) > BodyMaxLength {
				msg = fmt.Sprintf("%s...", msg[:BodyMaxLength-3])
			}
			notificationEvent := &db.NotificationEvent{
				Address:       notification.To,
				UserProfileID: receiver.ID,
				Read:          false,
				Timestamp:     time.Now(),
				Message:       msg,
				Type:          notification.Type,
				TypeID:        notification.TypeID,
			}

			notificationEvent.Meta = notification.Meta
			var senderImage, senderAddress *string
			if notification.From != nil {
				sender, err := s.db.UserByAddress(*notification.From)
				if err != nil {
					s.log.WithError(err).Errorf("could not retrieve user for address %s", *notification.From)
					continue
				}
				notificationEvent.SenderProfileID = sender.ID
				title = sender.Username
				senderImage = strPtr(sender.AvatarURL)
				senderAddress = strPtr(sender.Address)
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
				Body:  stripmd.Strip(msg),
				NotificationData: NotificationData{
					ID:        notificationEvent.ID,
					TypeID:    notification.TypeID,
					Timestamp: notificationEvent.Timestamp,
					UserID:    senderAddress,
					Image:     senderImage,
					Read:      notificationEvent.Read,
					Type:      notificationEvent.Type,
					Meta:      notificationEvent.Meta,
				},
			}

			if notification.Action != "" {
				pushNotification.Subtitle = notification.Action
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
		netAddress, err := nodeInfo.NetAddress()
		if err != nil {
			return
		}
		s.log.Infof("connected to [%s] address: %s", nodeInfo.Moniker, netAddress.String())
	}

}

func (s *service) run(stop <-chan struct{}) {

	remote := getEnv("REMOTE_ENDPOINT", "tcp://0.0.0.0:26657")
	client := client.NewHTTP(remote, "/websocket")
	tmTxQuery := "tru.event.tx = 'Push'"
	tmBlockQuery := "tru.event.block = 'Push'"
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

	txsCh, err := client.Subscribe(ctx, "trustory-push-tx-client", tmTxQuery)
	if err != nil {
		s.log.WithError(err).Fatal("could not connect to remote endpoint")
	}
	blocksCh, err := client.Subscribe(ctx, "trustory-push-block-client", tmBlockQuery)
	if err != nil {
		s.log.WithError(err).Fatal("could not connect to remote endpoint")
	}
	s.logChainStatus(client)
	s.log.Infof("subscribing to query event %s", tmTxQuery)
	s.log.Infof("subscribing to query event %s", tmBlockQuery)
	notificationsCh := make(chan *Notification)
	cNotificationsCh := make(chan *CommentNotificationRequest)
	go s.startHTTP(stop, cNotificationsCh)
	go s.processCommentsNotifications(cNotificationsCh, notificationsCh)
	go s.notificationSender(notificationsCh, stop)
	for {
		select {
		case event := <-txsCh:
			switch v := event.Data.(type) {
			case types.EventDataTx:
				s.processTxEvent(v, notificationsCh)
			case types.EventDataNewBlock:
				s.processBlockEvent(v, notificationsCh)
			}
		case event := <-blocksCh:
			switch v := event.Data.(type) {
			case types.EventDataTx:
				s.processTxEvent(v, notificationsCh)
			case types.EventDataNewBlock:
				s.processBlockEvent(v, notificationsCh)
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

	config := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_ADDR", "localhost"),
			Port: 5432,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}
	dbClient := db.NewDBClient(config)
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
