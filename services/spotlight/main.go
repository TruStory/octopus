package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/itskingori/go-wkhtml/wkhtmltox"
	"github.com/joho/godotenv"
	"github.com/machinebox/graphql"
)

type service struct {
	port          string
	router        *mux.Router
	storyTemplate *template.Template
	graphqlClient *graphql.Client
}

func (s *service) run() {
	s.router.Handle("/story/{id:[0-9]+}/render-spotlight", renderSpotlightHandler(s))
	s.router.Handle("/story/{id:[0-9]+}/spotlight", spotlightHandler(s))
	http.Handle("/", s.router)
	err := http.ListenAndServe(":"+s.port, nil)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file", err)
	}

	spotlight := &service{
		port:          getEnv("PORT", "54448"),
		router:        mux.NewRouter(),
		storyTemplate: template.Must(template.ParseFiles("story.html")),
		graphqlClient: graphql.NewClient(mustEnv("SPOTLIGHT_GRAPHQL_ENDPOINT")),
	}

	spotlight.run()
}

func renderSpotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyID, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		data := getStory(s, storyID)
		err = s.storyTemplate.Execute(w, data)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
	}

	return http.HandlerFunc(fn)
}

func spotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyID := vars["id"]
		ifs := make(wkhtmltox.ImageFlagSet)
		renderURL := fmt.Sprintf("http://localhost:%s/story/%s/render-spotlight", s.port, storyID)
		imageName := fmt.Sprintf("./storage/story-%s.png", storyID)
		_, err := ifs.Generate(renderURL, imageName)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		http.ServeFile(w, r, imageName)
	}

	return http.HandlerFunc(fn)
}

func getStory(s *service, storyID int64) StoryByIDResponse {
	graphqlReq := graphql.NewRequest(StoryByIDQuery)

	graphqlReq.Var("storyId", storyID)
	var graphqlRes StoryByIDResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		log.Fatal(err)
		panic(err)
	}

	return graphqlRes
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}
