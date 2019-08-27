package spotlight

import (
	"bytes"
	"context"
	"encoding/base64"
	"html"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-pg/pg"

	"github.com/TruStory/octopus/services/truapi/db"

	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/machinebox/graphql"
	stripmd "github.com/writeas/go-strip-markdown"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
)

var regexMention = regexp.MustCompile("(cosmos|tru)([a-z0-9]{4})[a-z0-9]{31}([a-z0-9]{4})")

const (
	WORDS_PER_LINE_CLAIM     = 7
	WORDS_PER_LINE_ARGUMENT  = 7
	WORDS_PER_LINE_COMMENT   = 7
	WORDS_PER_LINE_HIGHLIGHT = 10

	MAX_CHARS_PER_LINE = 40

	BODY_LINES_CLAIM     = 3
	BODY_LINES_ARGUMENT  = 3
	BODY_LINES_COMMENT   = 3
	BODY_LINES_HIGHLIGHT = 4
)

type Service struct {
	port          string
	router        *mux.Router
	graphqlClient *graphql.Client
	dbClient      *db.Client
	jpeg          bool
}

func NewService(port, endpoint string, jpeg bool, config truCtx.Config) *Service {
	return &Service{
		port:          port,
		router:        mux.NewRouter(),
		graphqlClient: graphql.NewClient(endpoint),
		dbClient:      db.NewDBClient(config),
		jpeg:          jpeg,
	}
}
func (s *Service) Run() {
	s.router.Handle("/claim/{id:[0-9]+}/spotlight", renderClaim(s))
	s.router.Handle("/argument/{id:[0-9]+}/spotlight", renderArgument(s))
	s.router.Handle("/claim/{claimID:[0-9]+}/comment/{id:[0-9]+}/spotlight", renderComment(s))
	s.router.Handle("/highlight/{id:[0-9]+}/spotlight", renderHighlight(s))
	http.Handle("/", s.router)
	err := http.ListenAndServe(":"+s.port, nil)
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

func render(preview string, w http.ResponseWriter, jpegEnabled bool) {
	cmd := exec.Command("rsvg-convert", "-f", "png", "--width", "1920", "--height", "1080")
	contentType := "image/png"
	if jpegEnabled {
		contentType = "image/jpeg"
	}
	w.Header().Add("Content-Type", contentType)
	cmd.Stdin = strings.NewReader(preview)
	buf := new(bytes.Buffer)
	cmd.Stdout = buf

	err := cmd.Run()
	if err != nil {
		http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
		return
	}
	if !jpegEnabled {
		_, err := io.Copy(w, buf)
		if err != nil {
			http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
		}
		return
	}

	pngImage, err := png.Decode(buf)
	if err != nil {
		http.Error(w, "PNG can't be decoded", http.StatusInternalServerError)
		return
	}

	if err := jpeg.Encode(w, pngImage, nil); err != nil {
		http.Error(w, "Can't encode to JPEG", http.StatusInternalServerError)
		return
	}
}

func renderClaim(s *Service) http.Handler {
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
		rawPreview, err := box.Find("claim.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}
		compiledPreview := compileClaimPreview(rawPreview, data.Claim)
		render(compiledPreview, w, s.jpeg)
	}
	return http.HandlerFunc(fn)
}

func renderHighlight(s *Service) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		highlightID, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid highlight ID passed.", http.StatusBadRequest)
			return
		}
		highlight, err := getHighlight(s, highlightID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		if highlight == nil {
			log.Println("Invalid highlight ID passed.")
			http.Error(w, "Invalid highlight ID passed.", http.StatusInternalServerError)
			return
		}
		if highlight.HighlightableType != "argument" {
			// when more highlightable types come in, this will become a loop. until then, efficiency.
			log.Println("invalid highlightable type")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		argument, err := getArgument(s, highlight.HighlightableID)
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}

		box := packr.New("Templates", "./templates")
		rawPreview, err := box.Find("highlight.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}
		compiledPreview, err := compileHighlightPreview(rawPreview, highlight, argument.ClaimArgument)
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}
		render(compiledPreview, w, s.jpeg)
	}
	return http.HandlerFunc(fn)
}

func renderArgument(s *Service) http.Handler {
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
		render(compiledPreview, w, s.jpeg)
	}

	return http.HandlerFunc(fn)
}

func renderComment(s *Service) http.Handler {
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
		render(compiledPreview, w, s.jpeg)
	}

	return http.HandlerFunc(fn)
}

func compileClaimPreview(raw []byte, claim ClaimObject) string {
	// BODY
	bodyLines := wordWrap(claim.Body, WORDS_PER_LINE_CLAIM)
	// make sure to have minimum lines atleast
	if len(bodyLines) < BODY_LINES_CLAIM {
		for i := len(bodyLines); i < BODY_LINES_CLAIM; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) > BODY_LINES_CLAIM {
		bodyLines[BODY_LINES_CLAIM-1] += "..." // ellipsis if the entire body couldn't be contained in this preview
	}
	compiled := bytes.Replace(raw, []byte("$PLACEHOLDER__BODY_LINE_1"), []byte(bodyLines[0]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_2"), []byte(bodyLines[1]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_3"), []byte(bodyLines[2]), -1)

	// ARGUMENT COUNT
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__ARGUMENT_COUNT"), []byte(strconv.Itoa(claim.ArgumentCount)), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+claim.Creator.UserProfile.Username), -1)

	// SOURCE
	if claim.HasSource() {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte(claim.GetSource()), -1)
	} else {
		compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__SOURCE"), []byte("—"), -1)
	}

	return string(compiled)
}

func compileHighlightPreview(raw []byte, highlight *db.Highlight, argument ArgumentObject) (string, error) {
	// BODY
	bodyLines := wordWrap(highlight.Text, WORDS_PER_LINE_HIGHLIGHT)
	// make sure to have minimum lines atleast
	if len(bodyLines) < BODY_LINES_HIGHLIGHT {
		for i := len(bodyLines); i < BODY_LINES_HIGHLIGHT; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) > BODY_LINES_HIGHLIGHT {
		bodyLines[BODY_LINES_HIGHLIGHT-1] += "..." // ellipsis if the entire body couldn't be contained in this preview
	}

	// base64-ing the avatar
	// we need to fetch the image and convert it into base64 so that we can embed it in the SVG template.
	avatarURL := strings.Replace(argument.Creator.UserProfile.AvatarURL, "_bigger", "_200x200", 1)
	avatarResponse, err := http.Get(avatarURL)
	if err != nil {
		return "", err
	}
	defer avatarResponse.Body.Close()

	avatar, err := ioutil.ReadAll(avatarResponse.Body)
	if err != nil {
		return "", err
	}
	avatarBase64 := base64.StdEncoding.EncodeToString(avatar)

	// compiling the template
	var compiled bytes.Buffer
	tmpl, err := template.New("highlight").Parse(string(raw))
	if err != nil {
		return "", err
	}

	vars := struct {
		BodyLines    []string
		Highlight    *db.Highlight
		Argument     ArgumentObject
		AvatarType   string
		AvatarBase64 string
	}{
		BodyLines:    bodyLines,
		Highlight:    highlight,
		Argument:     argument,
		AvatarType:   avatarResponse.Header.Get("Content-Type"),
		AvatarBase64: avatarBase64,
	}

	err = tmpl.Execute(&compiled, vars)
	if err != nil {
		return "", err
	}

	return compiled.String(), nil
}

func compileArgumentPreview(raw []byte, argument ArgumentObject) string {
	// BODY
	bodyLines := wordWrap(argument.Summary, WORDS_PER_LINE_ARGUMENT)
	// make sure to have minimum lines atleast
	if len(bodyLines) < BODY_LINES_ARGUMENT {
		for i := len(bodyLines); i < BODY_LINES_ARGUMENT; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) > BODY_LINES_ARGUMENT {
		bodyLines[BODY_LINES_ARGUMENT-1] += "..." // ellipsis if the entire body couldn't be contained in this preview
	}
	compiled := bytes.Replace(raw, []byte("$PLACEHOLDER__BODY_LINE_1"), []byte(bodyLines[0]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_2"), []byte(bodyLines[1]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_3"), []byte(bodyLines[2]), -1)

	// AGREE COUNT
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__AGREE_COUNT"), []byte(strconv.Itoa(argument.UpvotedCount)), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+argument.Creator.UserProfile.Username), -1)

	return string(compiled)
}

func compileCommentPreview(raw []byte, comment CommentObject) string {
	// BODY
	bodyLines := wordWrap(comment.Body, WORDS_PER_LINE_COMMENT)
	// make sure to have minimum lines atleast
	if len(bodyLines) < BODY_LINES_COMMENT {
		for i := len(bodyLines); i < BODY_LINES_COMMENT; i++ {
			bodyLines = append(bodyLines, "")
		}
	} else if len(bodyLines) > BODY_LINES_COMMENT {
		bodyLines[BODY_LINES_COMMENT-1] += "..." // ellipsis if the entire body couldn't be contained in this preview
	}
	compiled := bytes.Replace(raw, []byte("$PLACEHOLDER__BODY_LINE_1"), []byte(bodyLines[0]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_2"), []byte(bodyLines[1]), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__BODY_LINE_3"), []byte(bodyLines[2]), -1)

	// CREATED BY
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+comment.Creator.UserProfile.Username), -1)

	return string(compiled)
}

func wordWrap(body string, defaultWordsPerLine int) []string {
	body = stripmd.Strip(html.EscapeString(body))
	body = regexMention.ReplaceAllString(body, "$1$2...$3") // converts @cosmos1xqc5gsesg5m4jv252ce9g4jgfev52s68an2ss9 into @cosmos1xqc...2ss9
	lines := make([]string, 0)
	if strings.TrimSpace(body) == "" {
		lines = append(lines, body)
		return lines
	}

	// convert string to slice
	words := strings.Fields(body)
	wordsPerLine := defaultWordsPerLine
	maxCharsPerLine := MAX_CHARS_PER_LINE

	if len(words) < wordsPerLine {
		wordsPerLine = len(words)
	}

	for len(words) >= 1 {
		candidate := strings.Join(words[:wordsPerLine], " ")
		for len(candidate) > maxCharsPerLine {
			if len(words[0]) >= maxCharsPerLine {
				// if the first word (it'll always be the first word because it'd have been the last word that was omitted by the previous line)
				// itself is more than what a line can accomodate, we'll shorten it by taking only a few characters out of it.
				words[0] = words[0][:20] + "..." // take first few chars
			}
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

func getClaim(s *Service, claimID int64) (ClaimByIDResponse, error) {
	graphqlReq := graphql.NewRequest(ClaimByIDQuery)

	graphqlReq.Var("claimId", claimID)
	var graphqlRes ClaimByIDResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return graphqlRes, err
	}

	return graphqlRes, nil
}

func getHighlight(s *Service, highlightID int64) (*db.Highlight, error) {
	highlight := &db.Highlight{ID: highlightID}
	err := s.dbClient.Find(highlight)
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return highlight, nil
}

func getArgument(s *Service, argumentID int64) (ArgumentByIDResponse, error) {
	graphqlReq := graphql.NewRequest(ArgumentByIDQuery)

	graphqlReq.Var("argumentId", argumentID)
	var graphqlRes ArgumentByIDResponse
	ctx := context.Background()
	if err := s.graphqlClient.Run(ctx, graphqlReq, &graphqlRes); err != nil {
		return graphqlRes, err
	}

	return graphqlRes, nil
}

func getComment(s *Service, claimID int64, commentID int64) (CommentObject, error) {
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
