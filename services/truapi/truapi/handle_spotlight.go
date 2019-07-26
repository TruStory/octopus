package truapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// HandleSpotlight proxies the request from the clients to the spotlight service
func (ta *TruAPI) HandleSpotlight(res http.ResponseWriter, req *http.Request) {

	// firing up the http client
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	err := req.ParseForm()
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
		return
	}
	claimID := req.FormValue("claim_id")
	argumentID := req.FormValue("argument_id")
	commentID := req.FormValue("comment_id")
	if claimID == "" && argumentID == "" && commentID == "" {
		render.Error(res, req, "provide a valid claim or argument or comment", http.StatusBadRequest)
		return
	}

	// preparing the request
	spotlightURL := ""
	if claimID != "" && commentID != "" {
		spotlightURL = strings.Replace("http://localhost:54448/claim/CLAIM_ID/comment/COMMENT_ID/spotlight", "CLAIM_ID", claimID, -1)
		spotlightURL = strings.Replace(spotlightURL, "COMMENT_ID", commentID, -1)
	} else if claimID != "" {
		spotlightURL = strings.Replace("http://localhost:54448/claim/CLAIM_ID/spotlight", "CLAIM_ID", claimID, -1)
	} else if argumentID != "" {
		spotlightURL = strings.Replace("http://localhost:54448/argument/ARGUMENT_ID/spotlight", "ARGUMENT_ID", argumentID, -1)
	}
	request, err := http.NewRequest("GET", spotlightURL, req.Body)
	if err != nil {
		fmt.Println("error creating request ", err.Error())
		render.Error(res, req, err.Error(), http.StatusBadRequest)
		return
	}
	// processing the request
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("error requesting spotlight: ", err.Error())
		render.Error(res, req, err.Error(), http.StatusBadRequest)
		return
	}

	// reading the response
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("error processing spotlight response ", err.Error())
		render.Error(res, req, err.Error(), http.StatusBadRequest)
		return
	}

	// if all went well, sending back the response
	res.Header().Set("Content-Type", "image/jpeg")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(responseBody)
	if err != nil {
		render.Error(res, req, err.Error(), http.StatusBadRequest)
	}
}
