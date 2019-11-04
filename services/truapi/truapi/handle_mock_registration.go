package truapi

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
	"github.com/dghubble/go-twitter/twitter"
)

// HandleMockRegistration takes an empty request and returns a `RegistrationResponse`
func (ta *TruAPI) HandleMockRegistration(w http.ResponseWriter, r *http.Request) {
	// Get the mock Twitter User from the auth token
	twitterUser := getMockTwitterUser()

	user, _, err := RegisterTwitterUser(ta, twitterUser)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	cookie, err := cookies.GetLoginCookie(ta.APIContext, user)
	if err != nil {
		render.LoginError(w, r, ErrServerError, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie)
	response := ta.createUserResponse(r.Context(), user, false)
	render.Response(w, r, response, http.StatusOK)
}

func getMockTwitterUser() *twitter.User {
	// getting a random id
	id := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(999999999)
	mocked := &twitter.User{
		ID:              id,
		IDStr:           strconv.FormatInt(int64(id), 10),
		ScreenName:      "trustory_engineering",
		Name:            "Trustory Engineering",
		Email:           "engineering@trustory.io",
		ProfileImageURL: "https://pbs.twimg.com/profile_images/1123407649335209985/pSuqllTI_bigger.jpg",
	}

	return mocked
}
