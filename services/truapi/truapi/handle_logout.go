package truapi

import (
	"net/http"
	"os"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// Logout deletes a session and redirects the logged in user to the correct page
func Logout(apiCtx truCtx.TruAPIContext) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		cookie := cookies.GetLogoutCookie(apiCtx)
		http.SetCookie(w, cookie)
		http.Redirect(w, req, os.Getenv("AUTH_LOGOUT_REDIR"), http.StatusFound)
	}
	return http.HandlerFunc(fn)
}
