package truapi

import (
	"net/http"
	"net/http/httputil"
	url2 "net/url"
)

// MixpanelAPIEndpoint is the endpoint on the Mixpanel's side to track the events
const MixpanelAPIEndpoint = "https://api.mixpanel.com"

func HandleMixpanel() http.Handler {
	url, err := url2.Parse(MixpanelAPIEndpoint)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	defaultDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		// remove DNT header if present
		r.Header.Del("DNT")
		defaultDirector(r)
		// set host header or mixpanel will fail
		r.Host = url.Host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
}
