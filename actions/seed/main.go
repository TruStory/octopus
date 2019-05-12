package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	db "github.com/TruStory/octopus/services/api/db"
	"github.com/icrowley/fake"
)

// Mockery is the service
type Mockery struct {
	httpClient           *http.Client
	dbClient             *db.Client
	apiEndpoint          string
	apiCall              func(method string, route string, body io.Reader) []byte
	authenticatedAPICall func(user User, method string, route string, body io.Reader) []byte
}

func (m *Mockery) mock() {
	// seed a mock user
	fmt.Println("[1] Mocking the user...")
	user := mockUser(m)

	// seed n stories by the user
	fmt.Println("[2] Mocking the stories...")
	mockStoryBy(m, user.Data, 5)

	// seed n notifications for the user
	fmt.Println("[3] Mocking the notifications...")
	mockNotificationsFor(m, user.Data, 5)
}

func main() {

	mockery := &Mockery{
		httpClient:  &http.Client{},
		dbClient:    db.NewDBClient(),
		apiEndpoint: mustEnv("SEED_API_ENDPOINT"),
	}

	mockery.authenticatedAPICall = func(user User, method string, route string, body io.Reader) []byte {
		// preparing the request
		request, err := makeHTTPRequest(mockery, method, route, body)
		if err != nil {
			log.Printf("Cannot prepare the http request (reason: %s)\n", err)
			return nil
		}

		request.Header.Add("Cookie", "tru-user="+user.AuthenticationCookie)

		// processing the request
		response, err := processHTTPRequest(mockery, request)
		if err != nil {
			log.Printf("Cannot process the http request (reason: %s)\n", err)
			return nil
		}

		return response
	}

	mockery.apiCall = func(method string, route string, body io.Reader) []byte {
		// preparing the request
		request, err := makeHTTPRequest(mockery, method, route, body)
		if err != nil {
			log.Printf("Cannot prepare the http request (reason: %s)\n", err)
			return nil
		}

		// processing the request
		response, err := processHTTPRequest(mockery, request)
		if err != nil {
			log.Printf("Cannot process the http request (reason: %s)\n", err)
			return nil
		}

		return response
	}

	mockery.mock()
}

func mockNotificationsFor(m *Mockery, user User, n int) {
	for i := 0; i < n; i++ {
		twitterProfileID, err := strconv.ParseInt(user.UserID, 10, 64)
		if err != nil {
			log.Printf("Cannot parse user id (reason: %s)\n", err)
			return
		}
		notificationEvent := &db.NotificationEvent{
			Address:          user.Address,
			TwitterProfileID: twitterProfileID,
			Read:             false,
			Timestamp:        time.Now(),
			Message:          fake.Sentence(),
			Type:             db.NotificationStoryAction,
			TypeID:           rand.Int63n(1000000),
		}

		_, err = m.dbClient.Model(notificationEvent).Returning("*").Insert()
		if err != nil {
			log.Printf("Cannot create the notification (reason: %s)\n", err)
			return
		}

		fmt.Printf("Added notification #%d.\n", i+1)
	}
}

func mockStoryBy(m *Mockery, user User, n int) {
	for i := 0; i < n; i++ {
		addStoryRequest := &AddStoryRequest{
			AccountNumber: 0,
			ChainID:       "test-chain-JHrAIn",
			Fee: Fee{
				Amount: []Amount{
					{
						Denomination: "trusteak",
						Amount:       "0",
					},
				},
				Gas: 100000,
			},
			Memo: "msg",
			Msgs: []SubmitStoryMsg{
				{
					Creator:    user.Address,
					Body:       fake.SentencesN(3),
					CategoryID: 1,
					StoryType:  0,
				},
			},
			Sequence: 0,
		}
		addStoryRequestJSON, _ := json.Marshal(addStoryRequest)
		addStoryRequestHex := strings.ToUpper(hex.EncodeToString(addStoryRequestJSON))

		reqBody := &UnsignedRequest{
			MsgTypes: []string{"SubmitStoryMsg"},
			Tx:       addStoryRequestHex,
			TxRaw:    string(addStoryRequestJSON),
		}
		reqBodyJSON, err := json.Marshal(reqBody)
		if err != nil {
			log.Printf("Cannot read the response body (reason %s)\n", err)
			return
		}

		response := m.authenticatedAPICall(user, "POST", "/unsigned", bytes.NewBuffer(reqBodyJSON))

		fmt.Printf("Response for adding story #%d: %s\n", i+1, response)
	}
}

func mockUser(m *Mockery) MockedRegisterResponse {

	response := m.apiCall("POST", "/mock_register", nil)

	var mockedUser MockedRegisterResponse
	err := json.Unmarshal(response, &mockedUser)
	if err != nil {
		panic(err)
	}

	return mockedUser
}

func makeHTTPRequest(m *Mockery, method string, route string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, m.apiEndpoint+route, body)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	return request, nil
}

func processHTTPRequest(m *Mockery, request *http.Request) ([]byte, error) {
	response, err := m.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}
