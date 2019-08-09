package truapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TruStory/truchain/x/community"
	"github.com/gorilla/mux"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// FollowCommunitiesRequest is the request sent by an user to follow communities.
type FollowCommunitiesRequest struct {
	Communities []string `json:"communities"`
}

func validCommunity(communities community.Communities, c string) bool {
	for _, community := range communities {
		if c == community.ID {
			return true
		}
	}
	return false
}

func (ta *TruAPI) handleFollowCommunities(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		render.Error(w, r, "Unauthorized", http.StatusUnauthorized)
		return
	}
	followCommunitiesRequest := &FollowCommunitiesRequest{}
	err := json.NewDecoder(r.Body).Decode(followCommunitiesRequest)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	communities := ta.communitiesResolver(r.Context())
	for _, c := range followCommunitiesRequest.Communities {
		if !validCommunity(communities, c) {
			render.Error(w, r, fmt.Sprintf("Invalid community id %s", c), http.StatusBadRequest)
			return
		}
	}
	err = ta.DBClient.FollowCommunities(user.Address, followCommunitiesRequest.Communities)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	render.Response(w, r, followCommunitiesRequest.Communities, http.StatusOK)
}

type UnfollowCommunityResponse struct {
	CommunityID string `json:"community_id"`
}

func (ta *TruAPI) handleUnfollowCommunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID := vars["communityID"]
	if communityID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		render.Error(w, r, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err := ta.DBClient.UnfollowCommunity(user.Address, communityID)
	if err == db.ErrNotFollowingCommunity {
		render.Error(w, r, "You don't follow this community", http.StatusBadRequest)
		return
	}
	if err == db.ErrFollowAtLeastOneCommunity {
		render.Error(w, r, "Must follow at least one community", http.StatusBadRequest)
		return
	}

	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	render.Response(w, r, &UnfollowCommunityResponse{CommunityID: communityID}, http.StatusOK)
}
