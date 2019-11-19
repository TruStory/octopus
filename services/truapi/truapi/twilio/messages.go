package twilio

import (
	"errors"
	"strings"
)

type MessageFactory struct{}

type Message struct {
	body string
	vars []string
}

var messages = map[string]string{
	"verification": "Your TruStory verification code is: ?",
}

func NewMessage(identifier string, vars ...string) (*Message, error) {
	message, ok := messages[identifier]
	if !ok {
		return nil, errors.New("no such message available")
	}
	return &Message{
		body: message,
		vars: vars,
	}, nil
}

func (m *Message) Get() string {
	replaced := m.body
	for _, v := range m.vars {
		replaced = strings.Replace(replaced, "?", v, 1)
	}
	return replaced
}
