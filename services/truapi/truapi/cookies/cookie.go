package cookies

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/securecookie"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
)

const (
	// UserCookieName contains the name of the cookie that stores the user
	UserCookieName string = "tru-user"

	// AnonSessionCookieName to track anonymous users
	AnonSessionCookieName string = "tru-session"
	// SessionDuration defines expiration time so we can track users that come back
	SessionDuration time.Duration = time.Hour * 24 * 365

	// AuthenticationExpiry is the period for which,
	// the logged in user must be considered authenticated
	AuthenticationExpiry int64 = 72 // in hours
)

// AuthenticatedUser denotes the data structure of the data inside the encrypted cookie
type AuthenticatedUser struct {
	ID              int64
	Address         string
	AuthenticatedAt int64
}

// GetLoginCookie returns the http cookie that authenticates and identifies the given user
func GetLoginCookie(apiCtx truCtx.TruAPIContext, user *db.User) (*http.Cookie, error) {
	value, err := MakeLoginCookieValue(apiCtx, user)
	if err != nil {
		return nil, err
	}

	cookie := http.Cookie{
		Name:     UserCookieName,
		Path:     "/",
		HttpOnly: true,
		Value:    value,
		Expires:  time.Now().Add(time.Duration(AuthenticationExpiry) * time.Hour),
		Domain:   apiCtx.Config.Host.Name,
	}

	return &cookie, nil
}

// GetLogoutCookie returns the http cookie that overrides
// the login cookie to practically delete it.
func GetLogoutCookie(apiCtx truCtx.TruAPIContext) *http.Cookie {
	cookie := http.Cookie{
		Name:     UserCookieName,
		Path:     "/",
		HttpOnly: true,
		Value:    "",
		Expires:  time.Now(),
		Domain:   apiCtx.Config.Host.Name,
		MaxAge:   0,
	}

	return &cookie
}

// GetAuthenticatedUser gets the user from the request's http cookie
func GetAuthenticatedUser(apiCtx truCtx.TruAPIContext, r *http.Request) (*AuthenticatedUser, error) {
	cookie, err := r.Cookie(UserCookieName)
	if err != nil {
		return nil, err
	}

	s, err := getSecureCookieInstance(apiCtx)
	if err != nil {
		return nil, err
	}

	user := &AuthenticatedUser{}
	err = s.Decode(UserCookieName, cookie.Value, &user)
	if err != nil {
		return nil, err
	}

	// log out all users who are using a cookie with TwitterProfileID instead of user ID
	if user.ID == 0 {
		return nil, errors.New("Legacy twitter auth cookie found")
	}

	if isStale(user) {
		return nil, errors.New("Stale cookie found")
	}

	return user, nil
}

// MakeLoginCookieValue takes a user and encodes it into a cookie value.
func MakeLoginCookieValue(apiCtx truCtx.TruAPIContext, user *db.User) (string, error) {
	s, err := getSecureCookieInstance(apiCtx)
	if err != nil {
		return "", err
	}

	cookieValue := &AuthenticatedUser{
		ID:              user.ID,
		Address:         user.Address,
		AuthenticatedAt: time.Now().Unix(),
	}
	encodedValue, err := s.Encode(UserCookieName, cookieValue)
	if err != nil {
		return "", err
	}

	return encodedValue, nil
}

// isStale returns whether the cookie older than what is accepted
func isStale(user *AuthenticatedUser) bool {
	return time.
		// if the authentication time...
		Unix(user.AuthenticatedAt, 0).
		// ...exists before in past...
		Before(
			// ...than the valid period.
			time.Now().Add(time.Duration(-1*AuthenticationExpiry) * time.Hour))
}

func getSecureCookieInstance(apiCtx truCtx.TruAPIContext) (*securecookie.SecureCookie, error) {
	// Saves and excrypts the context in the cookie
	hashKey, err := hex.DecodeString(apiCtx.Config.Cookie.HashKey)
	if err != nil {
		return nil, err
	}
	blockKey, err := hex.DecodeString(apiCtx.Config.Cookie.EncryptKey)
	if err != nil {
		return nil, err
	}
	return securecookie.New(hashKey, blockKey), nil
}

type AnonymousSession struct {
	SessionID    string
	CreationTime time.Time
}

// GetAuthenticatedUser gets the user from the request's http cookie
func GetAnonymousSession(apiCtx truCtx.TruAPIContext, r *http.Request) (*AnonymousSession, error) {
	cookie, err := r.Cookie(AnonSessionCookieName)
	if err != nil {
		return nil, err
	}

	s, err := getSecureCookieInstance(apiCtx)
	if err != nil {
		return nil, err
	}

	session := &AnonymousSession{}
	err = s.Decode(AnonSessionCookieName, cookie.Value, &session)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.CreationTime.Add(SessionDuration)) {
		return nil, errors.New("stale cookie found")
	}
	return session, nil
}

func MakeAnonymousCookieValue(apiCtx truCtx.TruAPIContext, uuid string) (string, error) {
	s, err := getSecureCookieInstance(apiCtx)
	if err != nil {
		return "", err
	}
	cookieValue := &AnonymousSession{
		SessionID:    uuid,
		CreationTime: time.Now(),
	}
	encodedValue, err := s.Encode(AnonSessionCookieName, cookieValue)
	if err != nil {
		return "", err
	}
	return encodedValue, nil
}

// GetAnonSessionCookie returns the http cookie that authenticates and identifies the given user
func GetAnonSessionCookie(apiCtx truCtx.TruAPIContext) (*http.Cookie, error) {
	u2, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	value, err := MakeAnonymousCookieValue(apiCtx, u2.String())
	if err != nil {
		return nil, err
	}

	cookie := http.Cookie{
		Name:     AnonSessionCookieName,
		Path:     "/",
		HttpOnly: true,
		Value:    value,
		Expires:  time.Now().Add(SessionDuration),
		Domain:   apiCtx.Config.Host.Name,
	}

	return &cookie, nil
}

// AnonymousSessionHandler is a middleware to track session ids.
func AnonymousSessionHandler(apiCtx truCtx.TruAPIContext) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.Header.Get("x-mobile-request") == "true" {
				cookie := &http.Cookie{
					Name:     AnonSessionCookieName,
					Path:     "/",
					HttpOnly: true,
					MaxAge:   -1,
					Domain:   apiCtx.Config.Host.Name,
				}

				http.SetCookie(w, cookie)
				next.ServeHTTP(w, r)
				return
			}

			_, err := GetAnonymousSession(apiCtx, r)
			// cookie is present continue to next handler
			if err == nil {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := GetAnonSessionCookie(apiCtx)
			// can not create cookie but continue serving
			if err != nil {
				fmt.Println("error creating anonymous session id")
				next.ServeHTTP(w, r)
				return
			}
			http.SetCookie(w, cookie)
			next.ServeHTTP(w, r)
		})
	}
}
