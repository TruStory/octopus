package dripper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	Title  string `json:"title"`
	Detail string `json:"detail"`
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

	// adding a dummy workflow

	return dripper, nil
}

// AddWorkflowToRegistry adds a workflow to the registry
func (dripper *Dripper) AddWorkflowToRegistry(name, workflowID, emailID string) {
	dripper.WorkflowRegistry[name] = &Workflow{
		ID:      workflowID,
		EmailID: emailID,
	}
}

// NewDripper creates a fully configured dripper instance
func NewDripper(config context.Config) (*Dripper, error) {
	dripper, err := NewVanillaDripper(config.Dripper.Key)
	if err != nil {
		return nil, err
	}
	for _, workflow := range config.Dripper.Workflows {
		dripper.AddWorkflowToRegistry(workflow.Name, workflow.WorkflowID, workflow.EmailID)
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

	request, err := workflow.makeHTTPRequest(email)
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

func getHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}

func (workflow *Workflow) makeHTTPRequest(email string) (*http.Request, error) {
	body := struct {
		EmailAddress string `json:"email_address"`
	}{
		EmailAddress: email,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/automations/%s/emails/%s/queue", workflow.Dripper.Endpoint, workflow.ID, workflow.EmailID), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth("mohitisawesome", workflow.Dripper.APIKey)
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	return request, nil
}
