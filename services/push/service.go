package main

import (
	"net/http"

	"github.com/machinebox/graphql"

	"github.com/TruStory/octopus/services/truapi/db"
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
}
