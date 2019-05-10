package main

import (
	"net/http"

	"github.com/machinebox/graphql"

	db "github.com/TruStory/octopus/services/api/db"
	"github.com/sirupsen/logrus"
)

type service struct {
	db        *db.Client
	apnsTopic string
	log       logrus.FieldLogger
	// gorush
	httpClient        *http.Client
	gorushHTTPAddress string
	// graphql
	graphqlClient *graphql.Client

	// argumentMentions
	argumentMentionsCh chan argumentMention
}
