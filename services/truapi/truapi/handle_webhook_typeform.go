package truapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
)

// TypeformPayload represents the payload the webhook receives
type TypeformPayload struct {
	EventID      string       `json:"event_id"`
	EventType    string       `json:"event_type"`
	FormResponse FormResponse `json:"form_response"`
}

// FormResponse represents the form response withint the payload
type FormResponse struct {
	FormID      string         `json:"form_id"`
	Token       string         `json:"token"`
	SubmittedAt time.Time      `json:"submitted_at"`
	Definition  FormDefinition `json:"definition"`
	Answers     []Answer       `json:"answers"`
}

// FormDefinition defines the form
type FormDefinition struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Fields []FormField `json:"fields"`
}

// FormField represents an individual form field
type FormField struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	Type  string `json:"type"`
	Ref   string `json:"ref"`
}

// Answer represents the response by the user
type Answer struct {
	Type    string         `json:"type"`
	Field   FormField      `json:"field"`
	Text    string         `json:"text,omitempty"`
	Email   string         `json:"email,omitempty"`
	Boolean bool           `json:"boolean,omitempty"`
	Number  int64          `json:"number,omitempty"`
	Choice  *AnswerChoice  `json:"choice,omitempty"`
	Choices *AnswerChoices `json:"choices,omitempty"`
	Date    *Date          `json:"date,omitempty"`
	URL     string         `json:"url,omitempty"`
}

// AnswerChoice represents the choice response
type AnswerChoice struct {
	Label string `json:"label"`
}

// AnswerChoices represents the choices response
type AnswerChoices struct {
	Label []string `json:"labels"`
}

// Date represents the date response
type Date struct {
	time.Time
}

// UnmarshalJSON implements custom logic to unmarshal dates
func (d *Date) UnmarshalJSON(buf []byte) error {
	parsed, err := time.Parse("2006-01-02", strings.Trim(string(buf), `"`))
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

// HandleTypeformWebhook handles the webhook payloads from the typeform service
func (ta *TruAPI) HandleTypeformWebhook(r *http.Request) chttp.Response {
	var payload TypeformPayload
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	err = validateTypeformPayload(ta, r, reqBody)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnauthorized, err)
	}
	err = json.Unmarshal(reqBody, &payload)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	firstName, lastName, username, email := getSignupRequestDetailsFromPayload(payload)
	token, err := generateRandomString(32)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusInternalServerError, err)
	}
	user := &db.User{
		FirstName: firstName,
		LastName:  lastName,
		Username:  username,
		Email:     email,
		Token:     token,
	}

	err = ta.DBClient.AddUser(user)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	response, err := json.Marshal(user)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	return chttp.SimpleResponse(200, response)
}

func generateRandomString(length int) (string, error) {
	random := make([]byte, 16)
	_, err := rand.Read(random)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(random), nil
	// bytes := make([]byte, length)
	// for i := 0; i < length; i++ {
	// 	bytes[i] = byte(97 + rand.Intn(25)) // a=97 and z=97+25
	// }
	// return string(bytes)
}

func validateTypeformPayload(ta *TruAPI, request *http.Request, payload []byte) error {
	mac := hmac.New(sha256.New, []byte(ta.APIContext.Config.Typeform.PayloadSecret))
	_, err := mac.Write(payload)
	if err != nil {
		return errors.New("payload could not be authorised")
	}
	hash := mac.Sum(nil)
	hash64 := base64.StdEncoding.EncodeToString(hash)

	// Typeform-Signature: sha256=the-hmac-sha256-hash-of-the-request-here
	if request.Header.Get("Typeform-Signature") != "sha256="+hash64 {
		return errors.New("payload could not be authorised")
	}

	return nil
}

func getSignupRequestDetailsFromPayload(payload TypeformPayload) (string, string, string, string) {
	firstName := getAnswerForField(payload, "first_name").Text
	lastName := getAnswerForField(payload, "last_name").Text
	username := getAnswerForField(payload, "username").Text
	email := getAnswerForField(payload, "email").Email

	return firstName, lastName, username, email
}

func getAnswerForField(payload TypeformPayload, field string) *Answer {
	for _, answer := range payload.FormResponse.Answers {
		if answer.Field.Ref == field {
			return &answer
		}
	}

	return nil
}
