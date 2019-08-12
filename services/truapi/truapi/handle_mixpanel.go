package truapi

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// MixpanelAPIEndpoint is the endpoint on the Mixpanel's side to track the events
const MixpanelAPIEndpoint = "https://api.mixpanel.com"

// HandleMixpanel proxies the request from the clients to Mixpanel
func (ta *TruAPI) HandleMixpanel(w http.ResponseWriter, r *http.Request) {
	// target, err := url.Parse(MixpanelAPIEndpoint)
	// if err != nil {
	// 	render.Error(w, r, err.Error(), http.StatusBadRequest)
	// }
	// proxy := httputil.NewSingleHostReverseProxy(target)
	// proxy.Director = func(request *http.Request) {
	// 	// request.Header.Add("X-Forwarded-Host", request.Host)
	// 	// request.Header.Add("X-Origin-Host", target.Host)
	// 	request.URL.Scheme = target.Scheme
	// 	request.URL.Host = target.Host
	// 	request.URL.Path = parsePath(request.URL.Path)
	// 	q := request.URL.Query()
	// 	q.Set("ip", "0")
	// 	request.URL.RawQuery = q.Encode()

	// 	fmt.Println(request)
	// }
	// proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
	// 	render.Error(w, r, err.Error(), http.StatusBadRequest)
	// }
	// proxy.ServeHTTP(w, r)

	// firing up the http client
	client := &http.Client{}

	// // preparing the request
	request, err := http.NewRequest(r.Method, MixpanelAPIEndpoint+parsePath(r.URL.Path), r.Body)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
	}
	request.Header = r.Header
	request.Header.Set("X-Forwarded-For", getClientIP(r))
	q := r.URL.Query()
	q.Set("ip", "0")
	request.URL.RawQuery = q.Encode()

	// // processing the request
	response, err := client.Do(request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
	}

	// // reading the response
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
	}

	// // if all went well, sending back the response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(responseBody)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
	}
}

func getClientIP(r *http.Request) string {
	ip := r.RemoteAddr
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Got X-Forwarded-For
		ip = forwarded // If it's a single IP, then awesome!

		// If we got an array... grab the first IP
		ips := strings.Split(forwarded, ", ")
		if len(ips) > 1 {
			ip = ips[0]
		}
	}
	return ip
}

func parsePath(path string) string {
	if strings.HasSuffix(path, "/") { // removing the trailing slash
		path = strings.TrimSuffix(path, "/")
	}

	paths := strings.Split(path, "/")

	return "/" + paths[len(paths)-1] + "/"
}
