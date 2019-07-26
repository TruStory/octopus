package truapi

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// UserResponse is a JSON response body representing the result of User
type UserResponse struct {
	UserID         string                     `json:"userId"`
	Username       string                     `json:"username"` // deprecated. Use UserTwitterProfileResponse.Username
	Fullname       string                     `json:"fullname"` // deprecated. Use UserTwitterProfileResponse.Fullname
	Address        string                     `json:"address"`
	TwitterProfile UserTwitterProfileResponse `json:"twitterProfile"`
}

// UserTwitterProfileResponse is a JSON response body representing the TwitterProfile of a user
type UserTwitterProfileResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	AvatarURI string `json:"avatarURI"`
}

// UpdateUserViaTokenRequest updates a user via one-time use token
type UpdateUserViaTokenRequest struct {
	ID       uint64 `json:"id"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleUserDetails takes a `UserRequest` and returns a `UserResponse`
func (ta *TruAPI) HandleUserDetails(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPut:
		return ta.updateUserDetails(r)
	default:
		return ta.getUserDetails(r)
	}
}

func (ta *TruAPI) updateUserDetails(r *http.Request) chttp.Response {
	// There are two scenarios when a user can be updated:
	// a.) When users are setting their username + password (signing up) after admin approval (one-time process)
	// b.) When users are updating their profile (bio, photo, etc) from their account settings
	// To update a user, either the user cookie has to be present (for [scenario b]),
	// or a token has to be present (for [scenario a]).

	// attempt to get the cookie
	_, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		// no cookie present; proceed via token
		return ta.updateUserDetailsViaRequestToken(r)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// cookie found, proceed via cookie
	return ta.updateUserDetailsViaCookie(r)
}

func (ta *TruAPI) updateUserDetailsViaRequestToken(r *http.Request) chttp.Response {
	var request UpdateUserViaTokenRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	err = ta.DBClient.SignupUser(request.ID, request.Token, request.Username, request.Password)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	return chttp.SimpleResponse(http.StatusOK, nil)

}

func (ta *TruAPI) updateUserDetailsViaCookie(r *http.Request) chttp.Response {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// TODO: implement the code to edit profile

	response, err := json.Marshal(user)
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	return chttp.SimpleResponse(http.StatusOK, response)
}

func (ta *TruAPI) getUserDetails(r *http.Request) chttp.Response {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		return chttp.SimpleErrorResponse(401, err)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	twitterProfile, err := ta.DBClient.TwitterProfileByID(user.TwitterProfileID)
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// Chain was restarted and DB was wiped so Address and TwitterProfileID contained in cookie is stale.
	if twitterProfile.ID == 0 {
		return chttp.SimpleErrorResponse(401, err)
	}

	responseBytes, _ := json.Marshal(UserResponse{
		UserID:   strconv.FormatInt(twitterProfile.ID, 10),
		Fullname: twitterProfile.FullName,
		Username: twitterProfile.Username,
		Address:  twitterProfile.Address,
		TwitterProfile: UserTwitterProfileResponse{
			Username:  twitterProfile.Username,
			FullName:  twitterProfile.FullName,
			AvatarURI: twitterProfile.AvatarURI,
		},
	})

	return chttp.SimpleResponse(200, responseBytes)
}
