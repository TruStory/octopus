package truapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-pg/pg"
	"github.com/gorilla/mux"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UpdateNotificationEventRequest represents the JSON request
type UpdateNotificationEventRequest struct {
	NotificationID int64 `json:"notification_id"`
	Read           *bool `json:"read,omitempty"`
	Seen           *bool `json:"seen,omitempty"`
}

// HandleNotificationEvent takes a `UpdateNotificationEventRequest` and returns a 200 response
func (ta *TruAPI) HandleNotificationEvent(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPut:
		return ta.handleUpdateNotificationEvent(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) handleUpdateNotificationEvent(r *http.Request) chttp.Response {
	// check if we have a user before doing anything
	user := r.Context().Value(userContextKey)
	if user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	request := &UpdateNotificationEventRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}
	if request.Read == nil && request.Seen == nil {
		return chttp.SimpleErrorResponse(400, Err400MissingParameter)
	}

	// if request was made to mark all notification as read
	if request.NotificationID == -1 && request.Read != nil && *request.Read {
		return markAllAsRead(ta, r)
	}

	if request.NotificationID == -1 && request.Seen != nil && *request.Seen {
		return markAllAsSeen(ta, r)
	}

	notificationEvent := &db.NotificationEvent{ID: request.NotificationID}
	err = ta.DBClient.Find(notificationEvent)
	if err == pg.ErrNoRows {
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	if *request.Read {
		notificationEvent.Read = true
	} 

	notificationEvent.Seen = true
	err = ta.DBClient.UpdateModel(notificationEvent)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}

	return chttp.SimpleResponse(200, nil)
}

func markAllAsRead(ta *TruAPI, r *http.Request) chttp.Response {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	err = ta.DBClient.MarkAllNotificationEventsAsReadByAddress(user.Address)
	if err != nil {
		return chttp.SimpleErrorResponse(500, Err500InternalServerError)
	}

	err = ta.DBClient.MarkAllNotificationEventsAsSeenByAddress(user.Address)
	if err != nil {
		return chttp.SimpleErrorResponse(500, Err500InternalServerError)
	}

	return chttp.SimpleResponse(200, nil)
}

func (ta *TruAPI) handleThreadOpened(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars["claimID"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	claimID, err := strconv.ParseInt(vars["claimID"], 10, 64)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	// ignore if user is not present
	if err != nil || user == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	err = ta.DBClient.MarkThreadNotificationsAsRead(user.Address, claimID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func markAllAsSeen(ta *TruAPI, r *http.Request) chttp.Response {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	err = ta.DBClient.MarkAllNotificationEventsAsSeenByAddress(user.Address)
	if err != nil {
		return chttp.SimpleErrorResponse(500, Err500InternalServerError)
	}

	return chttp.SimpleResponse(200, nil)
}
