package dripper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TruStory/octopus/services/truapi/context"
)

// MailchimpAPIEndpoint is the base endpoint for the Mailchimp API
const MailchimpAPIEndpoint = "https://REGION.api.mailchimp.com/3.0"

// Workflow defines a drip campaign
type Workflow struct {
	ID      string
	EmailID string
	Tags    []string
	Dripper *Dripper
}

// Dripper is the drip campaign engine
type Dripper struct {
	Endpoint         string
	APIKey           string
	WorkflowRegistry map[string]*Workflow
}

// MailchimpError represents the error from the Mailchimp API
type MailchimpError struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// MailchimpWorkflow represents the mailchimp object for a workflow
type MailchimpWorkflow struct {
	ID         string             `json:"id"`
	Recipients WorkflowRecipients `json:"recipients"`
}

// WorkflowRecipients represents the mailchimp object for recipients
type WorkflowRecipients struct {
	ListID   string `json:"list_id"`
	ListName string `json:"list_name"`
}

// NewVanillaDripper creates a new instance of the Dripper
func NewVanillaDripper(key string) (*Dripper, error) {
	parts := strings.Split(key, "-") // api key --> key-region
	if len(parts) != 2 {
		return nil, errors.New("invalid api key provided. cannot initialize dripper")
	}
	dripper := &Dripper{
		Endpoint:         strings.Replace(MailchimpAPIEndpoint, "REGION", parts[1], -1),
		APIKey:           key,
		WorkflowRegistry: make(map[string]*Workflow),
	}

	return dripper, nil
}

// AddWorkflowToRegistry adds a workflow to the registry
func (dripper *Dripper) AddWorkflowToRegistry(name, workflowID, emailID string, tags []string) {
	dripper.WorkflowRegistry[name] = &Workflow{
		ID:      workflowID,
		EmailID: emailID,
		Tags:    tags,
	}
}

// NewDripper creates a fully configured dripper instance
func NewDripper(config context.Config) (*Dripper, error) {
	dripper, err := NewVanillaDripper(config.Dripper.Key)
	if err != nil {
		return nil, err
	}
	for _, workflow := range config.Dripper.Workflows {
		dripper.AddWorkflowToRegistry(workflow.Name, workflow.WorkflowID, workflow.EmailID, workflow.Tags)
	}

	return dripper, nil
}

// ToWorkflow returns the fully configured workflow object
func (dripper *Dripper) ToWorkflow(name string) *Workflow {
	workflow, ok := dripper.WorkflowRegistry[name]
	if !ok {
		return &Workflow{}
	}

	workflow.Dripper = dripper
	return workflow
}

// Subscribe subscribes an email to a workflow
func (workflow *Workflow) Subscribe(email string) error {
	// basic validation
	if workflow.ID == "" || workflow.EmailID == "" {
		return errors.New("invalid workflow provided")
	}

	err := workflow.addToAudience(email)
	if err != nil {
		return err
	}

	body := struct {
		EmailAddress string `json:"email_address"`
	}{
		EmailAddress: email,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := workflow.Dripper.makeMailchimpRequest(
		"POST",
		fmt.Sprintf("%s/automations/%s/emails/%s/queue", workflow.Dripper.Endpoint, workflow.ID, workflow.EmailID),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}
	response, err := getHTTPClient().Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode == 204 {
		return nil
	}

	// got an error
	var errorBody MailchimpError
	err = json.NewDecoder(response.Body).Decode(&errorBody)
	if err != nil {
		return err
	}

	return errors.New(errorBody.Detail)
}

func (workflow *Workflow) addToAudience(email string) error {
	recipients, err := workflow.getRecipients()
	if err != nil {
		return err
	}

	body := struct {
		EmailAddress string   `json:"email_address"`
		Status       string   `json:"status"`
		Tags         []string `json:"tags"`
	}{
		EmailAddress: email,
		Status:       "subscribed",
		Tags:         workflow.Tags,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := workflow.Dripper.makeMailchimpRequest(
		"POST",
		fmt.Sprintf("%s/lists/%s/members", workflow.Dripper.Endpoint, recipients.ListID),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}
	response, err := getHTTPClient().Do(request)
	if response.StatusCode == 200 {
		return nil
	}

	var errorBody MailchimpError
	err = json.NewDecoder(response.Body).Decode(&errorBody)
	if err != nil {
		return err
	}
	if errorBody.Title == "Member Exists" {
		// this error should fail the entire flow
		return nil
	}

	return errors.New(errorBody.Detail)
}

func (workflow *Workflow) getRecipients() (*WorkflowRecipients, error) {
	request, err := workflow.Dripper.makeMailchimpRequest(
		"GET",
		fmt.Sprintf("%s/automations/%s", workflow.Dripper.Endpoint, workflow.ID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	response, err := getHTTPClient().Do(request)

	var responseBody MailchimpWorkflow
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, err
	}

	return &responseBody.Recipients, nil
}

func (dripper *Dripper) makeMailchimpRequest(method, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth("mohitisawesome", dripper.APIKey)
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	return request, nil
}

func getHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}
