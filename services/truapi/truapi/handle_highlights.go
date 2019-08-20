package truapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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

	render.Response(w, r, highlight, 200)
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
