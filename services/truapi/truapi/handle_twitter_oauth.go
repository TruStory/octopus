package truapi

import (
	"net/http"

	truCtx "github.com/TruStory/octopus/services/truapi/context"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	gotwitter "github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/gologin"
	oauth1Login "github.com/dghubble/gologin/oauth1"
	"github.com/dghubble/gologin/twitter"
	"github.com/dghubble/oauth1"
)

// IssueSession creates a session and redirects the logged in user to the correct page
func IssueSession(apiCtx truCtx.TruAPIContext, ta *TruAPI) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		twitterUser, err := twitter.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		referrerCode, err := cookies.GetReferrerFromCookie(req)
		if err != nil {
			referrerCode = ""
		}

		user, err := CalibrateUser(ta, twitterUser, referrerCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = ta.DBClient.TouchLastAuthenticatedAt(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cookie, err := cookies.GetLoginCookie(apiCtx, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, req, apiCtx.Config.Web.AuthLoginRedir, http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

// OAuthLoginHandler handles Twitter login requests by obtaining a request token and
// redirecting to the authorization URL.
func OAuthLoginHandler(apiCtx truCtx.TruAPIContext, config *oauth1.Config, failure http.Handler) http.Handler {
	// persistReferrer -> oauth1.LoginHandler -> oauth1.AuthRedirectHandler
	authRedirectSuccess := oauth1Login.AuthRedirectHandler(config, failure)
	loginSuccess := oauth1Login.LoginHandler(config, authRedirectSuccess, failure)
	return persistReferrer(apiCtx, loginSuccess)
}

// HandleOAuthSuccess handles Twitter callback requests by parsing the oauth token
// and verifier and adding the Twitter access token and User to the ctx. If
// authentication succeeds, handling delegates to the success handler,
// otherwise to the failure handler.
func HandleOAuthSuccess(config *oauth1.Config, success, failure http.Handler) http.Handler {
	// oauth1.EmptyTempHandler -> oauth1.CallbackHandler -> TwitterHandler -> success
	twitterHandlerSuccess := twitterHandler(config, success, failure)
	callbackSuccess := oauth1Login.CallbackHandler(config, twitterHandlerSuccess, failure)
	return oauth1Login.EmptyTempHandler(callbackSuccess)
}

// HandleOAuthFailure handles the failed oAuth requests gracefully
func HandleOAuthFailure(ta *TruAPI) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		// if the authorization was purposefully denied by the user
		if req.FormValue("denied") != "" {
			http.Redirect(w, req, ta.APIContext.Config.Web.AuthDeniedRedir, http.StatusFound)
			return
		}

		// if any other error
		ctx := req.Context()
		err := gologin.ErrorFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// should be unreachable, ErrorFromContext always returns some non-nil error
		http.Error(w, "", http.StatusInternalServerError)
	}
	return http.HandlerFunc(fn)
}

// twitterHandler is a http.Handler that gets the OAuth1 access token from
// the ctx and calls Twitter verify_credentials to get the corresponding User.
// If successful, the User is added to the ctx and the success handler is
// called. Otherwise, the failure handler is called.
func twitterHandler(config *oauth1.Config, success, failure http.Handler) http.Handler {
	if failure == nil {
		failure = gologin.DefaultFailureHandler
	}
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		accessToken, accessSecret, err := oauth1Login.AccessTokenFromContext(ctx)
		if err != nil {
			ctx = gologin.WithError(ctx, err)
			failure.ServeHTTP(w, req.WithContext(ctx))
			return
		}
		httpClient := config.Client(ctx, oauth1.NewToken(accessToken, accessSecret))
		twitterClient := gotwitter.NewClient(httpClient)
		accountVerifyParams := &gotwitter.AccountVerifyParams{
			IncludeEntities: gotwitter.Bool(false),
			SkipStatus:      gotwitter.Bool(true),
			IncludeEmail:    gotwitter.Bool(true),
		}
		user, resp, err := twitterClient.Accounts.VerifyCredentials(accountVerifyParams)
		err = validateResponse(user, resp, err)
		if err != nil {
			ctx = gologin.WithError(ctx, err)
			failure.ServeHTTP(w, req.WithContext(ctx))
			return
		}
		ctx = twitter.WithUser(ctx, user)
		success.ServeHTTP(w, req.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func persistReferrer(apiCtx truCtx.TruAPIContext, success http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		referrerCode := req.FormValue("referrer")

		if referrerCode != "" {
			http.SetCookie(w, cookies.GetReferrerCookie(apiCtx, referrerCode))
		}
		success.ServeHTTP(w, req.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// validateResponse returns an error if the given Twitter user, raw
// http.Response, or error are unexpected. Returns nil if they are valid.
func validateResponse(user *gotwitter.User, resp *http.Response, err error) error {
	if err != nil || resp.StatusCode != http.StatusOK {
		return twitter.ErrUnableToGetTwitterUser
	}
	if user == nil || user.ID == 0 || user.IDStr == "" {
		return twitter.ErrUnableToGetTwitterUser
	}
	return nil
}
