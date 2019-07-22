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

	stripmd "github.com/writeas/go-strip-markdown"

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
	s.router.Handle("/claim/{id:[0-9]+}/render-spotlight", renderClaimSpotlight(s))
	s.router.Handle("/argument/{id:[0-9]+}/render-spotlight", renderArgumentSpotlight(s))
	s.router.Handle("/claim/{claimID:[0-9]+}/comment/{id:[0-9]+}/render-spotlight", renderCommentSpotlight(s))
	s.router.Handle("/claim/{id:[0-9]+}/spotlight", claimSpotlightHandler(s))
	s.router.Handle("/argument/{id:[0-9]+}/spotlight", argumentSpotlightHandler(s))
	s.router.Handle("/claim/{claimID:[0-9]+}/comment/{id:[0-9]+}/spotlight", commentSpotlightHandler(s))
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

func renderClaimSpotlight(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		claimID, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid claim ID passed.", http.StatusBadRequest)
			return
		}
		data, err := getClaim(s, claimID)
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

		compiledPreview := compileClaimPreview(rawPreview, data.Claim)
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
		rawPreview, err := box.Find("argument-v2.svg")
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

func renderCommentSpotlight(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		claimID, err := strconv.ParseInt(vars["claimID"], 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid claim ID passed.", http.StatusBadRequest)
			return
		}
		commentID, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid comment ID passed.", http.StatusBadRequest)
			return
		}
		comment, err := getComment(s, claimID, commentID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		box := packr.New("Templates", "./templates")
		rawPreview, err := box.Find("comment.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}

		compiledPreview := compileCommentPreview(rawPreview, comment)
		w.Header().Add("Content-Type", "image/svg+xml")
		_, err = fmt.Fprint(w, compiledPreview)
		if err != nil {
			http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func compileClaimPreview(raw []byte, claim ClaimObject) string {
	// BODY
	bodyLines := wordWrap(claim.Body)
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
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__ARGUMENT_COUNT"), []byte(strconv.Itoa(claim.ArgumentCount)), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+claim.Creator.TwitterProfile.Username), -1)

	// SOURCE
	if claim.HasSource() {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte(claim.GetSource()), -1)
	} else {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte("—"), -1)
	}

	return string(compiled)
}

func compileArgumentPreview(raw []byte, argument ArgumentObject) string {
	// BODY
	bodyLines := wordWrap(argument.Summary)
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

	// AGREE COUNT
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__AGREE_COUNT"), []byte(strconv.Itoa(argument.UpvotedCount)), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+argument.Creator.TwitterProfile.Username), -1)

	return string(compiled)
}

func compileCommentPreview(raw []byte, comment CommentObject) string {
	// BODY
	bodyLines := wordWrap(comment.Body)
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

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+comment.Creator.TwitterProfile.Username), -1)

	return string(compiled)
}

func wordWrap(body string) []string {
	body = stripmd.Strip(body)
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
		wordsPerLine = len(words)
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
		if len(words) < defaultWordsPerLine {
			wordsPerLine = len(words)
		} else {
			wordsPerLine = defaultWordsPerLine
		}
	}

	return lines
}

func claimSpotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		claimID := vars["id"]
		log.Printf("serving spotlight for claimID : [%s]", claimID)

		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetCacheDir(filepath.Join(s.storagePath, "web-cache"))
		ifs.SetFormat("jpeg")

		renderURL := fmt.Sprintf("http://localhost:%s/claim/%s/render-spotlight", s.port, claimID)
		imageName := fmt.Sprintf("claim-%s.jpeg", claimID)
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

func commentSpotlightHandler(s *service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		claimID := vars["claimID"]
		commentID := vars["id"]
		log.Printf("serving spotlight for commentID : [%s]", commentID)

		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetCacheDir(filepath.Join(s.storagePath, "web-cache"))
		ifs.SetFormat("jpeg")

		renderURL := fmt.Sprintf("http://localhost:%s/claim/%s/comment/%s/render-spotlight", s.port, claimID, commentID)
		imageName := fmt.Sprintf("comment-%s.jpeg", claimID)
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

func getClaim(s *service, claimID int64) (ClaimByIDResponse, error) {
	graphqlReq := graphql.NewRequest(ClaimByIDQuery)

	graphqlReq.Var("claimId", claimID)
	var graphqlRes ClaimByIDResponse
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

func getComment(s *service, claimID int64, commentID int64) (CommentObject, error) {
	graphqlReq := graphql.NewRequest(CommentsByClaimIDQuery)

	graphqlReq.Var("claimId", claimID)
	var graphqlRes CommentsByClaimIDResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return CommentObject{}, err
	}

	for _, comment := range graphqlRes.Claim.Comments {
		if comment.ID == commentID {
			return comment, nil
		}
	}

	return CommentObject{}, nil
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
