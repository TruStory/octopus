package graphql

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	thunder "github.com/samsarahq/thunder/graphql"
	"github.com/samsarahq/thunder/graphql/introspection"
	builder "github.com/samsarahq/thunder/graphql/schemabuilder"
	"github.com/spf13/viper"
	"github.com/tendermint/tmlibs/cli"
)

// Request represents the JSON body of a GraphQL query request
type Request struct {
	Query     string                 `json:"query"`     // The GraphQL query string
	Variables map[string]interface{} `json:"variables"` // Variable values for the query
}

// Client holds a GraphQL schema / execution context
type Client struct {
	pendingSchema *builder.Schema
	queries       *builder.Object
	mutations     *builder.Object
	Schema        *thunder.Schema
	Built         bool
}

// NewGraphQLClient returns a GraphQL client with an empty, unbuilt schema
func NewGraphQLClient() *Client {
	schema := builder.NewSchema()
	client := Client{pendingSchema: schema, queries: schema.Query(), mutations: schema.Mutation(), Schema: nil, Built: false}
	return &client
}

func (c *Client) Handler() http.Handler {
	if !c.Built {
		c.BuildSchema()
	}
	return thunder.HTTPHandler(c.Schema)
}

// RegisterQueryResolver adds a top-level resolver to find the first batch of entities in a GraphQL query
func (c *Client) RegisterQueryResolver(name string, fn interface{}) {
	c.queries.FieldFunc(name, fn, builder.Expensive)
}

// RegisterPaginatedQueryResolver adds a top-level resolver to find the first paginated batch of entities in a GraphQL query
func (c *Client) RegisterPaginatedQueryResolver(name string, fn interface{}) {
	c.queries.FieldFunc(name, fn, builder.Paginated, builder.Expensive)
}

// RegisterPaginatedQueryResolverWithFilter adds a top-level resolver to find the first paginated batch of entities in a GraphQL query filtered by content
func (c *Client) RegisterPaginatedQueryResolverWithFilter(name string, fn interface{}, filter map[string]interface{}) {
	options := []builder.FieldFuncOption{builder.Paginated, builder.Expensive}

	for k, i := range filter {
		options = append(options, builder.FilterField(k, i))
	}
	c.queries.FieldFunc(name, fn, options...)
}

// RegisterMutation registers a mutation
func (c *Client) RegisterMutation(name string, fn interface{}) {
	c.mutations.FieldFunc(name, fn, builder.Expensive)
}

// RegisterObjectResolver adds a set of field resolvers for objects of the given type that are returned by top-level resolvers
func (c *Client) RegisterObjectResolver(name string, objPrototype interface{}, fields map[string]interface{}) {
	obj := c.pendingSchema.Object(name, objPrototype)
	for fieldName, fn := range fields {
		obj.FieldFunc(fieldName, fn, builder.Expensive)
	}
}

// RegisterPaginatedObjectResolver adds a set of paginated field resolvers for objects of the given type that are returned by top-level resolvers
func (c *Client) RegisterPaginatedObjectResolver(name, key string, objPrototype interface{}, fields map[string]interface{}) {
	obj := c.pendingSchema.Object(name, objPrototype)
	obj.Key(key)

	for fieldName, fn := range fields {
		obj.FieldFunc(fieldName, fn, builder.Expensive)
	}
}

// BuildSchema builds the GraphQL schema from the given resolvers and
func (c *Client) BuildSchema() {
	builtSchema := c.pendingSchema.MustBuild()
	introspection.AddIntrospectionToSchema(builtSchema)
	c.Schema = builtSchema
	c.Built = true
}

// GenerateSchema writes the GraphQL schema to a file
func (c *Client) GenerateSchema() {
	valueJSON, err := introspection.ComputeSchemaJSON(*c.pendingSchema)
	if err != nil {
		panic(err)
	}

	rootdir := viper.GetString(cli.HomeFlag)
	if rootdir == "" {
		rootdir = os.ExpandEnv("$HOME/.truchaind")
	}

	path := filepath.Join(rootdir, "graphql-schema.json")
	err = ioutil.WriteFile(path, valueJSON, 0644)
	if err != nil {
		panic(err)
	}
}
