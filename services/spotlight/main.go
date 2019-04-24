package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/itskingori/go-wkhtml/wkhtmltox"
	"github.com/machinebox/graphql"
)

type service struct {
	port          string
	storagePath   string
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
		log.Println(err)
		panic(err)
	}
}

func main() {
	templatePath := getEnv("SPOTLIGHT_HTML_TEMPLATE", "story.html")
	spotlight := &service{
		port:          getEnv("PORT", "54448"),
		storagePath:   getEnv("SPOTLIGHT_STORAGE_PATH", "./storage"),
		router:        mux.NewRouter(),
		storyTemplate: template.Must(template.ParseFiles(templatePath)),
		graphqlClient: graphql.NewClient(mustEnv("SPOTLIGHT_GRAPHQL_ENDPOINT")),
	}

	spotlight.run()
}

func renderSpotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyID, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid story ID passed.", http.StatusBadRequest)
			return
		}
		data, err := getStory(s, storyID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		err = s.storyTemplate.Execute(w, data)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func spotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyID := vars["id"]
		log.Printf("serving spotlight for storyId : [%s]", storyID)

		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetCacheDir(filepath.Join(s.storagePath, "web-cache"))

		renderURL := fmt.Sprintf("http://localhost:%s/story/%s/render-spotlight", s.port, storyID)
		imageName := fmt.Sprintf("story-%s.png", storyID)
		filePath := filepath.Join(s.storagePath, imageName)

		_, err := ifs.Generate(renderURL, filePath)
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		http.ServeFile(w, r, filePath)
	}

	return http.HandlerFunc(fn)
}

func getStory(s *service, storyID int64) (StoryByIDResponse, error) {
	graphqlReq := graphql.NewRequest(StoryByIDQuery)

	graphqlReq.Var("storyId", storyID)
	var graphqlRes StoryByIDResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return graphqlRes, err
	}

	return graphqlRes, nil
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
