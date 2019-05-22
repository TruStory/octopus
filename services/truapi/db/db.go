package db

import (
	"fmt"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/go-pg/pg"
)

// Client is a Postgres client.
// It wraps a pool of Postgres DB connections.
type Client struct {
	*pg.DB
	config truCtx.Config
}

// NewDBClient creates a Postgres client
func NewDBClient(config truCtx.Config) *Client {
	db := pg.Connect(&pg.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Database.Host, config.Database.Port),
		User:     config.Database.User,
		Password: config.Database.Pass,
		Database: config.Database.Name,
		PoolSize: config.Database.Pool,
	})

	return &Client{db, config}
}

// GenericMutations write to the database
type GenericMutations interface {
	Add(model ...interface{}) error
	UpdateModel(model interface{}) error
	Remove(model interface{}) error
}

// Add adds any number of models as a database rows
func (c *Client) Add(model ...interface{}) error {
	return c.Insert(model...)
}

// UpdateModel updates a model
func (c *Client) UpdateModel(model interface{}) error {
	return c.Update(model)
}

// Remove deletes a models from a table
func (c *Client) Remove(model interface{}) error {
	return c.Delete(model)
}

// GenericQueries are generic reads for models
type GenericQueries interface {
	Count(model interface{}) (int, error)
	Find(model interface{}) error
	FindAll(models interface{}) error
}

// Count returns the count of the model
func (c *Client) Count(model interface{}) (count int, err error) {
	count, err = c.Model(model).Count()

	return
}

// Find selects a single model by primary key
func (c *Client) Find(model interface{}) error {
	return c.Select(model)
}

// FindAll selects all models
func (c *Client) FindAll(models interface{}) error {
	return c.Model(models).Select()
}
