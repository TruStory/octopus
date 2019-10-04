package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	app "github.com/TruStory/octopus/services/truapi/truapi"
)

func (s *service) startHTTPServer(
	stop <-chan struct{},
	commentNotifications chan<- *CommentNotificationRequest,
	rewardNotifications chan<- *app.RewardNotificationRequest,
	broadcastNotifications chan<- *app.BroadcastNotificationRequest,
) {
	mux := http.NewServeMux()
	s.addHTTPCommentNotificationHandler(mux, commentNotifications)
	s.addHTTPRewardNotificationHandler(mux, rewardNotifications)
	s.addHTTPBroadcastNotificationHandler(mux, broadcastNotifications)
	server := &http.Server{
		Addr:    ":9001",
		Handler: mux,
	}
	go func() {
		<-stop
		// we are at shutdown
		_ = server.Close()
	}()
	err := server.ListenAndServe()
	if err != nil {
		s.log.WithError(err).Fatal("error starting http service")
	}
}

func (s *service) addHTTPCommentNotificationHandler(mux *http.ServeMux, notifications chan<- *CommentNotificationRequest) {
	mux.HandleFunc("/sendCommentNotification", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fmt.Printf("only POST method allowed received [%s]\n", r.Method)
			return
		}
		n := &CommentNotificationRequest{}
		err := json.NewDecoder(r.Body).Decode(n)
		if err != nil {
			s.log.WithError(err).Error("error decoding request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.log.WithField("commentId", n.ID).Info("comment notification request received")
		notifications <- n
		w.WriteHeader(http.StatusAccepted)
	})
}

func (s *service) addHTTPRewardNotificationHandler(mux *http.ServeMux, notifications chan<- *app.RewardNotificationRequest) {
	mux.HandleFunc("/sendRewardNotification", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fmt.Printf("only POST method allowed received [%s]\n", r.Method)
			return
		}
		n := &app.RewardNotificationRequest{}
		err := json.NewDecoder(r.Body).Decode(n)
		if err != nil {
			s.log.WithError(err).Error("error decoding request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.log.WithField("rewardee_id", n.RewardeeID).Info("reward notification request received")
		notifications <- n
		w.WriteHeader(http.StatusAccepted)
	})
}

func (s *service) addHTTPBroadcastNotificationHandler(mux *http.ServeMux, notifications chan<- *app.BroadcastNotificationRequest) {
	mux.HandleFunc("/sendBroadcastNotification", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fmt.Printf("only POST method allowed received [%s]\n", r.Method)
			return
		}
		n := &app.BroadcastNotificationRequest{}
		err := json.NewDecoder(r.Body).Decode(n)
		if err != nil {
			s.log.WithError(err).Error("error decoding request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.log.WithField("type", n.Type).Info("broadcast notification request received")
		notifications <- n
		w.WriteHeader(http.StatusAccepted)
	})
}
