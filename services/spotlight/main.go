package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	s.router.Handle("/story/{id:[0-9]+}/svg-spotlight", spotlightSVG(s))
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
	templatePath := getEnv("SPOTLIGHT_HTML_TEMPLATE", "claim.html")
	spotlight := &service{
		port:          getEnv("PORT", "54448"),
		storagePath:   getEnv("SPOTLIGHT_STORAGE_PATH", "./storage"),
		router:        mux.NewRouter(),
		storyTemplate: template.Must(template.ParseFiles(templatePath)),
		graphqlClient: graphql.NewClient(mustEnv("SPOTLIGHT_GRAPHQL_ENDPOINT")),
	}

	spotlight.run()
}

func spotlightSVG(s *service) http.Handler {
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

		filePath := filepath.Join("./", "claim.svg")
		rawPreview, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}

		compiledPreview := compilePreview(rawPreview, data.Story)
		w.Header().Add("Content-Type", "image/svg+xml")
		_, err = fmt.Fprint(w, compiledPreview)
		if err != nil {
			http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func compilePreview(raw []byte, story StoryObject) string {
	// BODY
	bodyLines := wordWrap(story.Body)
	// make sure to have 4 lines atleast
	if len(bodyLines) < 4 {
		for i := len(bodyLines); i < 4; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) >= 4 {
		bodyLines[3] += "..." // ellipsis if the entire claim couldn't be contained in this preview
	}
	compiled := bytes.Replace(raw, []byte("$PLACEHOLDER__CLAIM_LINE_1"), []byte(bodyLines[0]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CLAIM_LINE_2"), []byte(bodyLines[1]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CLAIM_LINE_3"), []byte(bodyLines[2]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CLAIM_LINE_4"), []byte(bodyLines[3]), -1)

	// ARGUMENT COUNT
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__ARGUMENT_COUNT"), []byte(strconv.Itoa(story.GetArgumentCount())), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte(story.Creator.TwitterProfile.FullName), -1)

	// SOURCE
	if story.HasSource() {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte(story.Source), -1)
	} else {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte("—"), -1)
	}

	return string(compiled)
}

func wordWrap(body string) []string {
	defaultWordsPerLine := 7
	lines := make([]string, 0)

	if strings.TrimSpace(body) == "" {
		lines = append(lines, body)
		return lines
	}

	// convert string to slice
	words := strings.Fields(body)
	wordsPerLine := defaultWordsPerLine
	for len(words) >= 1 {
		candidate := strings.Join(words[:wordsPerLine], " ")
		for len(candidate) > 40 {
			wordsPerLine--
			candidate = strings.Join(words[:wordsPerLine], " ")
		}

		// add words into a line
		lines = append(lines, candidate)

		// remove the added words
		words = words[wordsPerLine:]

		// for the last few words
		if len(words) < wordsPerLine {
			wordsPerLine = len(words)
		} else {
			wordsPerLine = defaultWordsPerLine
		}
	}

	return lines
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

func spotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyID := vars["id"]
		log.Printf("serving spotlight for storyId : [%s]", storyID)

		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetCacheDir(filepath.Join(s.storagePath, "web-cache"))

		renderURL := fmt.Sprintf("http://localhost:%s/story/%s/svg-spotlight", s.port, storyID)
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
