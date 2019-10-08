package truapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	HighlightedURL    string `json:"highlighted_url"`
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

	highlightableType, highlightableID, err := parseHighlightableFromRequest(request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	highlight := &db.Highlight{
		HighlightableType: highlightableType,
		HighlightableID:   highlightableID,
		Text:              request.Text,
	}
	err = ta.DBClient.Add(highlight)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	go renderAndCacheHighlight(ta, highlight)

	render.Response(w, r, highlight, 200)
}

func renderAndCacheHighlight(ta *TruAPI, highlight *db.Highlight) {
	rendered, err := renderHighlight(ta, highlight)
	if err != nil {
		log.Println(err)
		return
	}

	url, err := cacheHighlight(ta, highlight, rendered)
	if err != nil {
		log.Println(err)
		return
	}

	err = ta.DBClient.AddImageURLToHighlight(highlight.ID, url)
	if err != nil {
		log.Println(err)
		return
	}
}

func renderHighlight(ta *TruAPI, highlight *db.Highlight) (io.Reader, error) {
	// firing up the http client
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	spotlightURL := fmt.Sprintf("%s/highlight/%d/spotlight", ta.APIContext.Config.Spotlight.URL, highlight.ID)
	request, err := http.NewRequest("GET", spotlightURL, nil)
	if err != nil {
		return nil, err
	}
	q := request.URL.Query()
	q.Add("highlight_id", strconv.FormatInt(highlight.ID, 10))
	request.URL.RawQuery = q.Encode()

	// processing the request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

func cacheHighlight(ta *TruAPI, highlight *db.Highlight, rendered io.Reader) (string, error) {
	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(ta.APIContext.Config.AWS.S3Region),
		Credentials: credentials.NewStaticCredentials(ta.APIContext.Config.AWS.AccessKey, ta.APIContext.Config.AWS.AccessSecret, ""),
	})
	if err != nil {
		return "", err
	}

	uploader := s3manager.NewUploader(session)
	uploaded, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(ta.APIContext.Config.AWS.S3Bucket),
		Key:    aws.String(fmt.Sprintf("highlights/highlight-%d-%d.jpg", highlight.ID, time.Now().Unix())),
		Body:   rendered,
	})
	if err != nil {
		return "", err
	}

	return uploaded.Location, nil
}

func validateCreateHighlightRequest(request CreateHighlightRequest) error {
	// if highlighted url is provided
	if request.HighlightedURL != "" {
		return nil
	}

	// if highlightable is provided
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
	valid := []string{"argument", "comment"}

	for _, validType := range valid {
		if validType == highlightableType {
			return nil
		}
	}

	return errors.New("invalid highlightable type")
}

func parseHighlightableFromRequest(request CreateHighlightRequest) (string, int64, error) {
	// if highlighted url is passed
	if request.HighlightedURL != "" {
		// if argument
		matches := claimArgumentRegex.FindStringSubmatch(request.HighlightedURL)
		if len(matches) == REGEX_MATCHES_CLAIM_ARGUMENT {
			highlightableType := "argument"
			highlightableID, err := strconv.ParseInt(matches[2], 10, 64)
			if err != nil {
				return "", 0, err
			}

			return highlightableType, highlightableID, nil
		}

		// if comment
		matches = claimCommentRegex.FindStringSubmatch(request.HighlightedURL)
		if len(matches) == REGEX_MATCHES_CLAIM_COMMENT {
			highlightableType := "comment"
			highlightableID, err := strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				return "", 0, err
			}

			return highlightableType, highlightableID, nil
		}

		// if argument comment
		matches = argumentCommentRegex.FindStringSubmatch(request.HighlightedURL)
		if len(matches) == REGEX_MATCHES_ARGUMENT_COMMENT {
			highlightableType := "comment"
			highlightableID, err := strconv.ParseInt(matches[3], 10, 64)
			if err != nil {
				return "", 0, err
			}

			return highlightableType, highlightableID, nil
		}

		return "", 0, errors.New("highlighted url not supported yet")
	}

	// if highlightable is passed
	return request.HighlightableType, request.HighlightableID, nil
}
