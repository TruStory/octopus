package truapi

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// HandleSpotlight proxies the request from the clients to the spotlight service
func (ta *TruAPI) HandleSpotlight(res http.ResponseWriter, req *http.Request) {

	// firing up the http client
	client := &http.Client{}

	err := req.ParseForm()
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
		return
	}
	storyID := req.FormValue("story_id")
	if storyID == "" {
		render.Error(res, req, "provide a valid story", http.StatusBadRequest)
		return
	}

	// preparing the request
	spotlightURL := strings.Replace("http://localhost:54448/story/STORY_ID/spotlight", "STORY_ID", storyID, -1)
	request, err := http.NewRequest("GET", spotlightURL, req.Body)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}
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
	res.Header().Set("Content-Type", "image/png")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(responseBody)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}
}
