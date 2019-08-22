package truapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// CreateHighlightRequest creates a new highlight
type CreateHighlightRequest struct {
	HighlightableType string `json:"highlightable_type"`
	HighlightableID   int64  `json:"highlightable_id"`
	Text              string `json:"text"`
}

// HandleHighlights represents a highlight on th argument text
func (ta *TruAPI) HandleHighlights(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ta.createHighlight(w, r)
		return
	default:
		render.Response(w, r, "ok", 200)
	}
}

func (ta *TruAPI) createHighlight(w http.ResponseWriter, r *http.Request) {
	var request CreateHighlightRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	err = validateCreateHighlightRequest(request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	highlight := &db.Highlight{
		HighlightableType: request.HighlightableType,
		HighlightableID:   request.HighlightableID,
		Text:              request.Text,
	}
	err = ta.DBClient.Add(highlight)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	url, err := renderAndCacheHighlight(ta, highlight)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	err = ta.DBClient.AddImageURLToHighlight(highlight.ID, url)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	render.Response(w, r, highlight, 200)
}

func renderAndCacheHighlight(ta *TruAPI, highlight *db.Highlight) (string, error) {
	rendered, err := renderHighlight(ta, highlight)
	if err != nil {
		return "", err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-1"),
		Credentials: credentials.NewStaticCredentials(ta.APIContext.Config.AWS.AccessKey, ta.APIContext.Config.AWS.AccessSecret, ""),
	})
	if err != nil {
		return "", err
	}

	uploader := s3manager.NewUploader(session)
	uploaded, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("trustory"),
		Key:    aws.String(fmt.Sprintf("highlights/highlight-%d-%d.jpg", highlight.ID, time.Now().Unix())),
		Body:   rendered,
	})
	if err != nil {
		return "", err
	}

	return uploaded.Location, nil
}

func renderHighlight(ta *TruAPI, highlight *db.Highlight) (io.Reader, error) {
	// firing up the http client
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	spotlightURL := fmt.Sprintf("%s/api/v1/spotlight", ta.APIContext.Config.App.URL)
	request, err := http.NewRequest("GET", spotlightURL, nil)
	if err != nil {
		return nil, err
	}
	q := request.URL.Query()
	q.Add("highlight_id", strconv.FormatInt(highlight.ID, 10))
	request.URL.RawQuery = q.Encode()
	fmt.Println(request, spotlightURL)

	// processing the request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

func validateCreateHighlightRequest(request CreateHighlightRequest) error {
	if request.HighlightableType == "" {
		return errors.New("invalid highlightable type")
	}
	err := validateHighlightableType(request.HighlightableType)
	if err != nil {
		return err
	}

	if request.HighlightableID == 0 {
		return errors.New("invalid highlightable id")
	}

	if strings.TrimSpace(request.Text) == "" {
		return errors.New("empty highlight")
	}

	return nil
}

func validateHighlightableType(highlightableType string) error {
	valid := []string{"argument"}

	for _, validType := range valid {
		if validType == highlightableType {
			return nil
		}
	}

	return errors.New("invalid highlightable type")
}
