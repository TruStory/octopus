package spotlight

import (
	"bytes"
	"context"
	"html"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	stripmd "github.com/writeas/go-strip-markdown"

	"github.com/gobuffalo/packr/v2"

	"github.com/gorilla/mux"
	"github.com/machinebox/graphql"
)

var regexMention = regexp.MustCompile("(cosmos|tru)([a-z0-9]{4})[a-z0-9]{31}([a-z0-9]{4})")

type Service struct {
	port          string
	router        *mux.Router
	graphqlClient *graphql.Client
}

func NewService(port, endpoint string) *Service {
	return &Service{
		port:          port,
		router:        mux.NewRouter(),
		graphqlClient: graphql.NewClient(endpoint),
	}
}
func (s *Service) Run() {
	s.router.Handle("/claim/{id:[0-9]+}/spotlight", renderClaim(s))
	s.router.Handle("/argument/{id:[0-9]+}/spotlight", renderArgument(s))
	s.router.Handle("/claim/{claimID:[0-9]+}/comment/{id:[0-9]+}/spotlight", renderComment(s))
	http.Handle("/", s.router)
	err := http.ListenAndServe(":"+s.port, nil)
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

func render(preview string, w http.ResponseWriter) {
	cmd := exec.Command("rsvg-convert", "-f", "png", "--width", "1920", "--height", "1080")
	w.Header().Add("Content-Type", "image/jpg")
	cmd.Stdin = strings.NewReader(preview)
	buf := new(bytes.Buffer)
	cmd.Stdout = buf

	err := cmd.Run()
	if err != nil {
		http.Error(w, "URL Preview cannot be generated", http.StatusInternalServerError)
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
		rawPreview, err := box.Find("claim-v2.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}
		compiledPreview := compileClaimPreview(rawPreview, data.Claim)
		render(compiledPreview, w)
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
		rawPreview, err := box.Find("argument-v2.svg")
		if err != nil {
			log.Println(err)
			http.Error(w, "URL Preview error", http.StatusInternalServerError)
			return
		}

		compiledPreview := compileArgumentPreview(rawPreview, data.ClaimArgument)
		render(compiledPreview, w)
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
		render(compiledPreview, w)
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
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+claim.Creator.UserProfile.Username), -1)

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
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+argument.Creator.UserProfile.Username), -1)

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
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__CREATOR"), []byte("@"+comment.Creator.UserProfile.Username), -1)

	return string(compiled)
}

func wordWrap(body string) []string {
	body = stripmd.Strip(html.EscapeString(body))
	body = regexMention.ReplaceAllString(body, "$1$2...$3") // converts @cosmos1xqc5gsesg5m4jv252ce9g4jgfev52s68an2ss9 into @cosmos1xqc...2ss9
	defaultWordsPerLine := 7
	lines := make([]string, 0)
	if strings.TrimSpace(body) == "" {
		lines = append(lines, body)
		return lines
	}

	// convert string to slice
	words := strings.Fields(body)
	wordsPerLine := defaultWordsPerLine
	maxCharsPerLine := 40

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
