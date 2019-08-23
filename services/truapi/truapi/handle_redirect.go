package truapi

import (
	"net/http"
)

func AddCacheBustingHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

type redirectHandler struct {
	url  string
	code int
}

func (rh *redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Redirect(w, r, rh.url, rh.code)
}

func Redirect(w http.ResponseWriter, r *http.Request, urlStr string, code int) {
	AddCacheBustingHeaders(w)
	http.Redirect(w, r, urlStr, code)
}

func RedirectHandler(url string, code int) http.Handler {
	return &redirectHandler{url, code}
}
