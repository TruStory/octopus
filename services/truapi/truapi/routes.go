package truapi

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/dghubble/oauth1"
	twitterOAuth1 "github.com/dghubble/oauth1/twitter"
	"github.com/gorilla/handlers"

	"github.com/TruStory/octopus/services/truapi/chttp"
	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// RegisterRoutes applies the TruStory API routes to the `chttp.API` router
func (ta *TruAPI) RegisterRoutes(apiCtx truCtx.TruAPIContext) {
	sessionHandler := cookies.AnonymousSessionHandler(ta.APIContext)
	ta.Use(sessionHandler)

	liveRedirectHandler := RedirectHandler(apiCtx.Config.App.LiveDebateURL, http.StatusFound)
	ta.Handle("/live", liveRedirectHandler)

	// Mixpanel support
	ta.PathPrefix("/mixpanel", http.StripPrefix("/mixpanel", HandleMixpanel()))
	api := ta.Subrouter("/api/v1")

	// Enable gzip compression
	api.Use(handlers.CompressHandler)
	api.Use(chttp.JSONResponseMiddleware)
	api.Use(WithUser(ta.APIContext))
	api.Use(ta.WithDataLoaders())
	api.Handle("/ping", WrapHandler(ta.HandlePing))

	api.Handle("/graphql", ta.GraphQLClient.Handler())
	api.Handle("/presigned", WrapHandler(ta.HandlePresigned))
	api.Handle("/unsigned", WrapHandler(ta.HandleUnsigned))
	api.HandleFunc("/register", ta.HandleRegistration)
	api.Handle("/user/search", WrapHandler(ta.HandleUsernameSearch))
	api.Handle("/notification", WrapHandler(ta.HandleNotificationEvent))
	api.HandleFunc("/deviceToken", ta.HandleDeviceTokenRegistration)
	api.HandleFunc("/deviceToken/unregister", ta.HandleUnregisterDeviceToken)
	api.HandleFunc("/upload", ta.HandleUpload)
	api.Handle("/flagStory", WrapHandler(ta.HandleFlagStory))
	api.HandleFunc("/comments", ta.HandleComment)
	api.Handle("/questions", WrapHandler(ta.HandleQuestion))
	api.HandleFunc("/comments/open/{claimID:[0-9]+}", ta.handleThreadOpened)
	api.HandleFunc("/comments/open/{claimID:[0-9]+}/{argumentID:[0-9]+}/{elementID:[0-9]+}", ta.handleThreadOpened)
	api.Handle("/reactions", WrapHandler(ta.HandleReaction))
	api.HandleFunc("/mentions/translateToCosmos", ta.HandleTranslateCosmosMentions)
	api.Handle("/track/", http.HandlerFunc(ta.HandleTrackEvent))
	api.Handle("/claim_of_the_day", WrapHandler(ta.HandleClaimOfTheDayID))
	api.Handle("/claim/image", WrapHandler(ta.HandleClaimImage))
	api.HandleFunc("/spotlight", ta.HandleSpotlight)
	api.HandleFunc("/request_tru", ta.HandleRequestTru)

	// users
	api.HandleFunc("/user", ta.HandleUserDetails)
	api.HandleFunc("/users/blacklist", BasicAuth(apiCtx, http.HandlerFunc(ta.HandleUserBlacklisting)))
	api.HandleFunc("/users/password-reset", ta.HandleUserForgotPassword)
	api.HandleFunc("/users/resend-email-verification", ta.HandleResendEmailVerification)
	api.HandleFunc("/users/validate/username", ta.HandleUniqueUsernameUtility)
	api.HandleFunc("/users/validate/email", ta.HandleUniqueEmailUtility)
	api.HandleFunc("/users/authentication", ta.HandleUserAuthentication)
	api.HandleFunc("/users/onboard", ta.HandleUserOnboard)
	api.HandleFunc("/users/journey", BasicAuth(apiCtx, http.HandlerFunc(ta.HandleUserJourney)))

	api.HandleFunc("/gift", BasicAuth(apiCtx, http.HandlerFunc(ta.HandleGift)))
	api.Handle("/communities/follow", http.HandlerFunc(ta.handleFollowCommunities)).Methods(http.MethodPost)
	api.Handle("/communities/unfollow/{communityID}",
		http.HandlerFunc(ta.handleUnfollowCommunity)).Methods(http.MethodDelete)
	api.Handle("/highlights", http.HandlerFunc(ta.HandleHighlights))

	// metrics
	api.HandleFunc("/metrics/users", ta.HandleUsersMetrics)
	api.HandleFunc("/metrics/claims", ta.HandleClaimMetrics)
	api.HandleFunc("/metrics/auth", BasicAuth(apiCtx, http.HandlerFunc(ta.HandleAuthMetrics)))
	api.HandleFunc("/metrics/invites", BasicAuth(apiCtx, http.HandlerFunc(ta.HandleInvitesMetrics)))
	api.HandleFunc("/metrics/user_base", ta.HandleUserBase)

	if apiCtx.Config.App.MockRegistration {
		api.HandleFunc("/mock_register", ta.HandleMockRegistration)
	}

	ta.RegisterOAuthRoutes(apiCtx)

	// Register routes for Trustory React web app
	fs := http.FileServer(http.Dir(apiCtx.Config.Web.Directory))

	ta.PathPrefix("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webDirectory := apiCtx.Config.Web.Directory
		// if it is not requesting a file with a valid extension serve the index
		if filepath.Ext(path.Base(r.URL.Path)) == "" {
			indexPath := filepath.Join(webDirectory, "index.html")

			absIndexPath, err := filepath.Abs(indexPath)
			if err != nil {
				log.Printf("ERROR index.html -- %s", err)
				http.Error(w, "Error serving index.html", http.StatusNotFound)
				return
			}
			indexFile, err := ioutil.ReadFile(absIndexPath)
			if err != nil {
				log.Printf("ERROR index.html -- %s", err)
				http.Error(w, "Error serving index.html", http.StatusNotFound)
				return
			}
			compiledIndexFile := CompileIndexFile(ta, indexFile, r.RequestURI)

			w.Header().Add("Content-Type", "text/html")
			_, err = fmt.Fprint(w, compiledIndexFile)
			if err != nil {
				log.Printf("ERROR index.html -- %s", err)
				http.Error(w, "Error serving index.html", http.StatusInternalServerError)
				return
			}
			return
		}
		fs.ServeHTTP(w, r)
	}))
}

// RegisterOAuthRoutes adds the proper routes needed for the oauth
func (ta *TruAPI) RegisterOAuthRoutes(apiCtx truCtx.TruAPIContext) {
	oauth1Config := &oauth1.Config{
		ConsumerKey:    apiCtx.Config.Twitter.APIKey,
		ConsumerSecret: apiCtx.Config.Twitter.APISecret,
		CallbackURL:    apiCtx.Config.Twitter.OAUTHCallback,
		Endpoint:       twitterOAuth1.AuthorizeEndpoint,
	}

	ta.Handle("/auth-twitter", OAuthLoginHandler(apiCtx, oauth1Config, nil))
	ta.Handle("/auth-twitter-callback", HandleOAuthSuccess(oauth1Config, IssueSession(apiCtx, ta), HandleOAuthFailure(ta)))
	ta.Handle("/auth-logout", Logout(apiCtx))
}
