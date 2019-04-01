package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	truchain "github.com/TruStory/truchain/types"
	db "github.com/TruStory/truchain/x/db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sideshow/apns2/certificate"

	"github.com/sideshow/apns2"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

// Notification represents a notification to be sent to the end service.
type Notification struct {
	From *sdk.AccAddress
	To   sdk.AccAddress
	Msg  string
}

type service struct {
	db         *db.Client
	apnsClient *apns2.Client
	apnsTopic  string
	log        logrus.FieldLogger
}

func (s *service) notificationSender(notifications <-chan *Notification, stop <-chan struct{}) {
	for {
		select {
		case notification := <-notifications:
			apnsNotficiation := &apns2.Notification{}
			msg := notification.Msg
			if notification.From != nil {
				profile, err := s.db.TwitterProfileByAddress(notification.From.String())
				if err != nil {
					s.log.WithError(err).Errorf("could not retrieve twitter profile for address %s", notification.From.String())
					continue
				}
				msg = fmt.Sprintf(notification.Msg, profile.FullName)
			}
			payload := fmt.Sprintf(`{"aps":{"alert":"%s"}}`, msg)
			receiverAddress := notification.To.String()
			deviceTokens, err := s.db.DeviceTokensByAddress(receiverAddress)
			if err != nil {
				s.log.WithError(err).Error("error retrieving tokens from db")
				continue
			}
			if len(deviceTokens) == 0 {
				s.log.Infof("account address %s doesn't not have push notification tokens \n", receiverAddress)
				continue
			}
			for _, deviceToken := range deviceTokens {
				apnsNotficiation.Payload = []byte(payload)
				apnsNotficiation.Topic = s.apnsTopic
				apnsNotficiation.DeviceToken = deviceToken.Token
				res, err := s.apnsClient.Push(apnsNotficiation)
				if err != nil {
					s.log.WithError(err).Error("error sending notification")
					continue
				}
				s.log.Infof("notification sent to %s: status code: %d response-id: %s reason : %s\n",
					notification.To.String(),
					res.StatusCode, res.ApnsID,
					res.Reason)
			}

		case <-stop:
			s.log.Info("stopping notification sender")
			return

		}
	}
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
		switch action {
		case "back_story":
			alert = "%s backed your story"
		case "create_challenge":
			alert = "%s challenged your story"
		case "like_backing_argument":
			alert = "%s supported your backing argument"
		case "like_challenge_argument":
			alert = "%s supported your challenge argument"
		}
		if alert != "" {
			notifications <- &Notification{From: &pushData.From, To: pushData.To, Msg: alert}
		}
	}
}

func (s *service) processNewBlockEvent(newBlockEvent types.EventDataNewBlock, notifications chan<- *Notification) {
	for _, tag := range newBlockEvent.ResultEndBlock.Tags {
		if string(tag.Key) == "tru.event.completedStories" {
			completed := &truchain.CompletedStoriesNotificationResult{}
			err := json.Unmarshal(tag.Value, completed)
			if err != nil {
				s.log.WithError(err).Error("error decoding completed stories")
				continue
			}
			for _, story := range completed.Stories {
				notifications <- &Notification{To: story.Creator, Msg: "A claim you made has completed"}
				for _, backer := range story.Backers {
					notifications <- &Notification{To: backer, Msg: "A claim you backed has completed"}
				}
				for _, challenger := range story.Challengers {
					notifications <- &Notification{To: challenger, Msg: "A claim you challenged has completed"}
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

// MustEnv will panic if the variable is not set.
func MustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("should provide variable %s", env))
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

	s.log.Infof("subscribing to query event %s", tmQuery.String())
	notificationsChan := make(chan *Notification)
	go s.notificationSender(notificationsChan, stop)
	for {
		select {
		case event := <-txsCh:
			switch v := event.(type) {
			case types.EventDataTx:
				s.processTransactionEvent(v, notificationsChan)
			case types.EventDataNewBlock:
				s.processNewBlockEvent(v, notificationsChan)
			}
		case <-stop:
			// program is exiting
			_ = client.Stop()
			s.log.Info("service stopped")
			return
		case <-time.After(15 * time.Second):
			status, err := client.Status()
			if err != nil {
				s.log.WithError(err).Error("error connecting to chain")
				continue
			}
			if status != nil {
				nodeInfo := status.NodeInfo
				s.log.Infof("connected to chain id: %s version: %s address: %s", nodeInfo.ID(), nodeInfo.Version, nodeInfo.ListenAddr)
			}

		}
	}
}

func main() {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	dbClient := db.NewDBClient()
	certLocation := MustEnv("CERT_LOCATION")
	certPassword := MustEnv("CERT_PASSWORD")
	prodEnabled := os.Getenv("APNS_PROD") == "true"
	topic := getEnv("NOTIFICATION_TOPIC", "io.trustory.alpha2")

	cert, err := certificate.FromP12File(certLocation, certPassword)
	if err != nil {
		log.WithError(err).Fatal("error reading certificate from p12 file")
	}

	apnsClient := apns2.NewClient(cert).Development()
	if prodEnabled {
		apnsClient.Production()
	}
	log.Info("pushd connected to db and starting")

	quit := setupSignals()
	srvc := &service{
		apnsClient: apnsClient,
		apnsTopic:  topic,
		db:         dbClient,
		log:        log,
	}
	srvc.run(quit)
}
