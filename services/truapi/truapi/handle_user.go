package truapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/TruStory/octopus/services/truapi/truapi/render"

	"github.com/TruStory/octopus/services/truapi/truapi/regex"

	"github.com/TruStory/octopus/services/truapi/db"

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

// RegisterUserRequest represents the schema of the http request to create a new user
type RegisterUserRequest struct {
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Username   string `json:"username"`
	ReferredBy string `json:"referred_by"`
}

// VerifyUserViaTokenRequest updates a user via one-time use token
type VerifyUserViaTokenRequest struct {
	ID    int64  `json:"id"`
	Token string `json:"token"`
}

// UpdateUserViaCookieRequest updates a user's profile fields
type UpdateUserViaCookieRequest struct {
	// Profile fields
	Profile *db.UserProfile `json:"profile,omitempty"`

	// Password fields
	Password *db.UserPassword `json:"password,omitempty"`
}

// HandleUserDetails takes a `UserRequest` and returns a `UserResponse`
func (ta *TruAPI) HandleUserDetails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ta.createNewUser(w, r)
	case http.MethodPut:
		ta.updateUserDetails(w, r)
	default:
		ta.getUserDetails(w, r)
	}
}

func (ta *TruAPI) createNewUser(w http.ResponseWriter, r *http.Request) {
	var request RegisterUserRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = validateRegisterRequest(request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user := &db.User{
		FullName: request.FullName,
		Email:    request.Email,
		Password: request.Password,
		Username: request.Username,
	}

	err = ta.DBClient.SignupUser(user, request.ReferredBy)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}

func (ta *TruAPI) updateUserDetails(w http.ResponseWriter, r *http.Request) {
	// There are two scenarios when a user can be updated:
	// a.) When users are verifying their email address (via token)
	// b.) When users are updating their profile (bio, photo, etc) from their account settings
	// To update a user, either the user cookie has to be present (for [scenario b]),
	// or a token has to be present (for [scenario a]).

	// attempt to get the cookie
	_, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		// no cookie present; proceed via token
		ta.verifyUserViaToken(w, r)
	}
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	// cookie found, proceed via cookie
	ta.updateUserDetailsViaCookie(w, r)
}

func (ta *TruAPI) verifyUserViaToken(w http.ResponseWriter, r *http.Request) {
	var request VerifyUserViaTokenRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = ta.DBClient.VerifyUser(request.ID, request.Token)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// MILESTONE -- successfully verified, let's give the user an address (registering user on the chain)
	keyPair, err := makeNewKeyPair()
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	// registering the keypair
	pubKeyBytes, err := hex.DecodeString(keyPair.PublicKey)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	address, err := ta.RegisterKey(pubKeyBytes, "secp256k1")
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = ta.DBClient.AddAddressToUser(request.ID, address.String())
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}

func (ta *TruAPI) updateUserDetailsViaCookie(w http.ResponseWriter, r *http.Request) {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	var request UpdateUserViaCookieRequest
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// if user wants to change their password
	if request.Password != nil {
		if request.Password.New != request.Password.NewConfirmation {
			render.Error(w, r, "new passwords do not match", http.StatusBadRequest)
			return
		}

		err = validatePassword(request.Password.New)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		err = ta.DBClient.UpdatePassword(user.ID, request.Password)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		render.Response(w, r, true, http.StatusOK)
		return
	}

	// if user wants to change their profile
	if request.Profile != nil {
		err = ta.DBClient.UpdateProfile(user.ID, request.Profile)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		render.Response(w, r, true, http.StatusOK)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}

func (ta *TruAPI) getUserDetails(w http.ResponseWriter, r *http.Request) {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	twitterProfile, err := ta.DBClient.TwitterProfileByID(user.TwitterProfileID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	// Chain was restarted and DB was wiped so Address and TwitterProfileID contained in cookie is stale.
	if twitterProfile.ID == 0 {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	response := UserResponse{
		UserID:   strconv.FormatInt(twitterProfile.ID, 10),
		Fullname: twitterProfile.FullName,
		Username: twitterProfile.Username,
		Address:  twitterProfile.Address,
		TwitterProfile: UserTwitterProfileResponse{
			Username:  twitterProfile.Username,
			FullName:  twitterProfile.FullName,
			AvatarURI: twitterProfile.AvatarURI,
		},
	}

	render.Response(w, r, response, http.StatusOK)
}

func validateRegisterRequest(request RegisterUserRequest) error {
	request.FullName = strings.TrimSpace(request.FullName)
	request.Email = strings.TrimSpace(request.Email)
	request.Username = strings.TrimSpace(request.Username)
	request.Password = strings.TrimSpace(request.Password)

	if request.FullName == "" {
		return errors.New("first name cannot be empty")
	}

	if request.Email == "" {
		return errors.New("email cannot be empty")
	}
	if !regex.IsValidEmail(request.Email) {
		return errors.New("invalid email provided")
	}

	if request.Username == "" {
		return errors.New("username cannot be empty")
	}
	if !regex.IsValidUsername(request.Username) {
		return errors.New("usernames can only contain alphabets, numbers and underscore")
	}
	// https://play.golang.org/p/DxRDseAtacL
	if regex.HasTrustory(request.Username) {
		return errors.New("usernames cannot seem to be related to trustory")
	}

	err := validatePassword(request.Password)
	if err != nil {
		return err
	}

	return nil
}

func validatePassword(password string) error {
	hasMinLength, hasUppercaseLetter, hasLowercaseLetter, hasNumber, hasSpecial := false, false, false, false, false

	for _, char := range password {
		switch {
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsUpper(char):
			hasUppercaseLetter = true
		case unicode.IsLower(char):
			hasLowercaseLetter = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if len(password) >= 8 {
		hasMinLength = true
	}

	if !hasMinLength {
		return errors.New("password must be 8 characters long")
	}

	if !hasNumber {
		return errors.New("password must have a number")
	}

	if !hasUppercaseLetter {
		return errors.New("password must have an uppercase letter")
	}

	if !hasLowercaseLetter {
		return errors.New("password must have a lowercase letter")
	}

	if !hasSpecial {
		return errors.New("password must have a special character")
	}

	return nil
}
