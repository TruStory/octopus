package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"strings"

	db "github.com/TruStory/truchain/x/db"
	"github.com/gernest/mention"
	stripmd "github.com/writeas/go-strip-markdown"
)

func unique(values []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range values {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func (s *service) parseCosmosMentions(body string) (string, []string) {
	parsedBody := body
	usernameByAddress := map[string]string{}
	terminators := []rune(" \n\r.,():!?")
	addresses := mention.GetTagsAsUniqueStrings('@', body, terminators...)
	for _, address := range addresses {
		twitterProfile, err := s.db.TwitterProfileByAddress(address)
		if err != nil {
			s.log.WithError(err).Errorf("could not find profile for address %s", address)
			continue
		}
		usernameByAddress[address] = twitterProfile.Username
	}
	for address, username := range usernameByAddress {
		parsedBody = strings.ReplaceAll(parsedBody, address, username)
	}
	return parsedBody, addresses
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

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

		parsedComment, addresses := s.parseCosmosMentions(c.Body)
		parsedComment = stripmd.Strip(parsedComment)
		participants = append(participants, addresses...)
		participants = unique(participants)
		meta := db.NotificationMeta{
			ArgumentID: &c.ArgumentID,
			StoryID:    &n.StoryID,
			CommentID:  &n.ID,
		}
		if c.Creator != n.ArgumentCreator {
			notifications <- &Notification{
				From:   &c.Creator,
				To:     n.ArgumentCreator,
				TypeID: c.ArgumentID,
				Type:   db.NotificationCommentAction,
				Msg:    parsedComment,
				Meta:   meta,
				Trim:   true,
			}
		}

		mentionType := db.MentionComment
		meta.MentionType = &mentionType
		for _, p := range participants {
			if p == c.Creator || p == n.ArgumentCreator {
				continue
			}
			var action string
			if contains(addresses, p) {
				action = "Mentioned you in a comment"
			}
			notifications <- &Notification{
				From:   &c.Creator,
				To:     p,
				TypeID: n.ArgumentID,
				Type:   db.NotificationMentionAction,
				Msg:    parsedComment,
				Meta:   meta,
				Action: action,
				Trim:   true,
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
