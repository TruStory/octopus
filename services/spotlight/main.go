package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gobuffalo/packr/v2"

	"github.com/gorilla/mux"
	"github.com/itskingori/go-wkhtml/wkhtmltox"
	"github.com/machinebox/graphql"
)

type service struct {
	port          string
	storagePath   string
	router        *mux.Router
	graphqlClient *graphql.Client
}

func (s *service) run() {
	s.router.Handle("/story/{id:[0-9]+}/render-spotlight", renderStorySpotlight(s))
	s.router.Handle("/argument/{id:[0-9]+}/render-spotlight", renderArgumentSpotlight(s))
	s.router.Handle("/story/{id:[0-9]+}/spotlight", storySpotlightHandler(s))
	s.router.Handle("/argument/{id:[0-9]+}/spotlight", argumentSpotlightHandler(s))
	http.Handle("/", s.router)
	err := http.ListenAndServe(":"+s.port, nil)
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

func main() {
	spotlight := &service{
		port:          getEnv("PORT", "54448"),
		storagePath:   getEnv("SPOTLIGHT_STORAGE_PATH", "./storage"),
		router:        mux.NewRouter(),
		graphqlClient: graphql.NewClient(mustEnv("SPOTLIGHT_GRAPHQL_ENDPOINT")),
	}

	spotlight.run()
}

func renderStorySpotlight(s *service) http.Handler {
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

		box := packr.New("Templates", "./templates")
		rawPreview, err := box.Find("claim-v2.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}

		compiledPreview := compileStoryPreview(rawPreview, data.Story)
		w.Header().Add("Content-Type", "image/svg+xml")
		_, err = fmt.Fprint(w, compiledPreview)
		if err != nil {
			http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func renderArgumentSpotlight(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		argumentID, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid argument ID passed.", http.StatusBadRequest)
			return
		}
		data, err := getArgument(s, argumentID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		box := packr.New("Templates", "./templates")
		rawPreview, err := box.Find("argument.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}

		compiledPreview := compileArgumentPreview(rawPreview, data.ClaimArgument)
		w.Header().Add("Content-Type", "image/svg+xml")
		_, err = fmt.Fprint(w, compiledPreview)
		if err != nil {
			http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func compileStoryPreview(raw []byte, story StoryObject) string {
	// BODY
	bodyLines := wordWrap(story.Body)
	// make sure to have 3 lines atleast
	if len(bodyLines) < 3 {
		for i := len(bodyLines); i < 3; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) > 3 {
		bodyLines[2] += "..." // ellipsis if the entire body couldn't be contained in this preview
	}
	compiled := bytes.Replace(raw, []byte("$PLACEHOLDER__BODY_LINE_1"), []byte(bodyLines[0]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_2"), []byte(bodyLines[1]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_3"), []byte(bodyLines[2]), -1)

	// ARGUMENT COUNT
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__ARGUMENT_COUNT"), []byte(strconv.Itoa(story.GetArgumentCount())), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte(story.Creator.TwitterProfile.FullName), -1)

	// SOURCE
	if story.HasSource() {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte(story.GetSource()), -1)
	} else {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte("—"), -1)
	}

	return string(compiled)
}

func compileArgumentPreview(raw []byte, argument ArgumentObject) string {
	// BODY
	bodyLines := wordWrap(argument.Summary)
	// make sure to have 4 lines atleast
	if len(bodyLines) < 4 {
		for i := len(bodyLines); i < 4; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) > 4 {
		bodyLines[3] += "..." // ellipsis if the entire body couldn't be contained in this preview
	}
	compiled := bytes.Replace(raw, []byte("$PLACEHOLDER__BODY_LINE_1"), []byte(bodyLines[0]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_2"), []byte(bodyLines[1]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_3"), []byte(bodyLines[2]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_4"), []byte(bodyLines[3]), -1)

	// AGREE COUNT
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__AGREE_COUNT"), []byte(strconv.Itoa(argument.UpvoteCount)), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte(argument.Creator.TwitterProfile.FullName), -1)

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

	if len(words) < wordsPerLine {
		lines = append(lines, strings.Join(words, " "))
		return lines
	}

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

func storySpotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		storyID := vars["id"]
		log.Printf("serving spotlight for storyId : [%s]", storyID)

		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetCacheDir(filepath.Join(s.storagePath, "web-cache"))
		ifs.SetFormat("jpeg")

		renderURL := fmt.Sprintf("http://localhost:%s/story/%s/render-spotlight", s.port, storyID)
		imageName := fmt.Sprintf("story-%s.jpeg", storyID)
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

func argumentSpotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		argumentID := vars["id"]
		log.Printf("serving spotlight for argumentId : [%s]", argumentID)

		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetCacheDir(filepath.Join(s.storagePath, "web-cache"))
		ifs.SetFormat("jpeg")

		renderURL := fmt.Sprintf("http://localhost:%s/argument/%s/render-spotlight", s.port, argumentID)
		imageName := fmt.Sprintf("argument-%s.jpeg", argumentID)
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

func getArgument(s *service, argumentID int64) (ArgumentByIDResponse, error) {
	graphqlReq := graphql.NewRequest(ArgumentByIDQuery)

	graphqlReq.Var("argumentId", argumentID)
	var graphqlRes ArgumentByIDResponse
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
