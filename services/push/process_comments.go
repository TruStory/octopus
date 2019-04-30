package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	db "github.com/TruStory/truchain/x/db"
)

func (s *service) processCommentsNotifications(cNotifications <-chan *CommentNotificationRequest, notifications chan<- *Notification) {
	for n := range cNotifications {
		c, err := s.db.CommentByID(n.ID)
		if err != nil {
			s.log.WithError(err).Errorf("could not retrieve comment for id [%d]\n", n.ID)
			continue
		}
		participants, err := s.db.CommentsParticipantsByArgumentID(c.ArgumentID)
		if err != nil {
			s.log.WithError(err).Errorf("could not retrieve participants for comments argument_id[%d]\n", n.ArgumentID)
			continue
		}

		if c.Creator != n.ArgumentCreator {
			notifications <- &Notification{
				From:   strPtr(c.Creator),
				To:     n.ArgumentCreator,
				TypeID: n.StoryID,
				Type:   db.NotificationStoryAction,
				Msg:    c.Body,
			}
		}

		for _, p := range participants {
			if p == c.Creator || p == n.ArgumentCreator {
				continue
			}
			notifications <- &Notification{
				From:   strPtr(c.Creator),
				To:     p,
				TypeID: n.StoryID,
				Type:   db.NotificationStoryAction,
				Msg:    c.Body,
			}
		}

	}
}

func (s *service) startHTTP(stop <-chan struct{}, notifications chan<- *CommentNotificationRequest) {
	mux := http.NewServeMux()
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
		s.log.WithField("commentId", n.ID).Info("comment notification request recevied")
		notifications <- n
		w.WriteHeader(http.StatusAccepted)
	})
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
