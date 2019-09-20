package truapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// HandlePush proxies the request from the clients to the push service
func (ta *TruAPI) HandlePush(res http.ResponseWriter, req *http.Request) {

	// firing up the http client
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// preparing the request
	request, err := http.NewRequest(http.MethodPost, ta.APIContext.Config.Push.EndpointURL+parsePath(req.URL.Path), req.Body)
	fmt.Println(request)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	// processing the request
	response, err := client.Do(request)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}

	// reading the response
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}

	// if all went well, sending back the response
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(responseBody)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}
}

func parsePath(path string) string {
	if strings.HasSuffix(path, "/") { // removing the trailing slash
		path = strings.TrimSuffix(path, "/")
	}

	paths := strings.Split(path, "/")

	return "/" + paths[len(paths)-1]
}
