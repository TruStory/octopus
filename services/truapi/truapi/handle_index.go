package truapi

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
	"strconv"

	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/category"
	"github.com/TruStory/truchain/x/story"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stripmd "github.com/writeas/go-strip-markdown"
)

const (
	defaultDescription = "TruStory is a social network to debate claims with skin in the game"
	defaultImage       = "Image+from+iOS.jpg"
)

var (
	storyRegex         = regexp.MustCompile("/story/([0-9]+)$")
	claimRegex         = regexp.MustCompile("/claim/([0-9]+)$")
	argumentRegex      = regexp.MustCompile("/story/([0-9]+)/argument/([0-9]+)$")
	claimArgumentRegex = regexp.MustCompile("/claim/([0-9]+)/argument/([0-9]+)$")
	commentRegex       = regexp.MustCompile("/story/([0-9]+)/argument/([0-9]+)/comment/([0-9]+)$")
	claimCommentRegex  = regexp.MustCompile("/claim/([0-9]+)/argument/([0-9]+)/comment/([0-9]+)$")
)

// Tags defines the struct containing all the request Meta Tags for a page
type Tags struct {
	Title       string
	Description string
	Image       string
	URL         string
}

// CompileIndexFile replaces the placeholders for the social sharing
func CompileIndexFile(ta *TruAPI, index []byte, route string) string {

	// /story/xxx
	matches := storyRegex.FindStringSubmatch(route)
	if len(matches) == 2 {
		// replace placeholder with story details, where story id is in matches[1]
		storyID, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeStoryMetaTags(ta, route, storyID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	// /claim/xxx
	matches = claimRegex.FindStringSubmatch(route)
	if len(matches) == 2 {
		// replace placeholder with claim details, where claim id is in matches[1]
		claimID, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeClaimMetaTags(ta, route, uint64(claimID))
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	// /story/xxx/argument/xxx
	matches = argumentRegex.FindStringSubmatch(route)
	if len(matches) == 3 {
		// replace placeholder with story details, where story id is in matches[1]
		storyID, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		argumentID, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeArgumentMetaTags(ta, route, storyID, argumentID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	// /claim/xxx/argument/xxx
	matches = claimArgumentRegex.FindStringSubmatch(route)
	if len(matches) == 3 {
		// replace placeholder with claim details, where claim id is in matches[1]
		claimID, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		argumentID, err := strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeClaimArgumentMetaTags(ta, route, claimID, argumentID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	// /story/xxx/argument/xxx/comment/xxx
	matches = commentRegex.FindStringSubmatch(route)
	if len(matches) == 4 {
		// replace placeholder with story details, where story id is in matches[1]
		storyID, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		argumentID, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		commentID, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeCommentMetaTags(ta, route, storyID, argumentID, commentID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	// /claim/xxx/argument/xxx/comment/xxx
	matches = claimCommentRegex.FindStringSubmatch(route)
	if len(matches) == 4 {
		// replace placeholder with claim details, where claim id is in matches[1]
		claimID, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		argumentID, err := strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		commentID, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeClaimCommentMetaTags(ta, route, claimID, argumentID, commentID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	return compile(index, makeDefaultMetaTags(ta, route))
}

// compiles the index file with the variables
func compile(index []byte, tags Tags) string {
	compiled := bytes.Replace(index, []byte("$PLACEHOLDER__TITLE"), []byte(tags.Title), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__DESCRIPTION"), []byte(tags.Description), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__IMAGE"), []byte(tags.Image), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__URL"), []byte(tags.URL), -1)

	return string(compiled)
}

// makes the default meta tags
func makeDefaultMetaTags(ta *TruAPI, route string) Tags {
	return Tags{
		Title:       ta.APIContext.Config.App.Name,
		Description: defaultDescription,
		Image:       joinPath(ta.APIContext.Config.App.S3AssetsURL, defaultImage),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}
}

func joinPath(baseURL, route string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	u.Path = path.Join(u.Path, route)
	return u.String()

}

// meta tags for a story
func makeStoryMetaTags(ta *TruAPI, route string, storyID int64) (*Tags, error) {
	ctx := context.Background()

	storyObj := ta.storyResolver(ctx, story.QueryStoryByIDParams{ID: storyID})
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: storyObj.ID})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: storyObj.ID})
	backingTotalAmount := ta.backingPoolResolver(ctx, storyObj)
	challengeTotalAmount := ta.challengePoolResolver(ctx, storyObj)

	totalParticipants := len(backings) + len(challenges)
	totalParticipantsPlural := "s"
	if totalParticipants == 1 {
		totalParticipantsPlural = ""
	}
	totalStake := backingTotalAmount.Add(challengeTotalAmount).Amount.Quo(sdk.NewInt(app.Shanev))

	return &Tags{
		Title:       html.EscapeString(storyObj.Body),
		Description: fmt.Sprintf("%d participant%s, %s TruStake", totalParticipants, totalParticipantsPlural, totalStake),
		Image:       joinPath(ta.APIContext.Config.App.S3AssetsURL, defaultImage),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

// meta tags for a claim
func makeClaimMetaTags(ta *TruAPI, route string, claimID uint64) (*Tags, error) {
	ctx := context.Background()

	claimObj := ta.claimResolver(ctx, queryByClaimID{ID: claimID})
	participants := ta.claimParticipantsResolver(ctx, claimObj)
	totalStaked := sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
	arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: claimID})
	for _, argument := range arguments {
		stakes := ta.claimArgumentStakesResolver(ctx, argument)
		for _, stake := range stakes {
			totalStaked = totalStaked.Add(stake.Amount)
		}
	}

	totalParticipants := len(participants)
	totalParticipantsPlural := "s"
	if totalParticipants == 1 {
		totalParticipantsPlural = ""
	}

	return &Tags{
		Title:       html.EscapeString(claimObj.Body),
		Description: fmt.Sprintf("%d articipant%s, %s TruStake", totalParticipants, totalParticipantsPlural, totalStaked.Amount),
		Image:       fmt.Sprintf("%s/api/v1/spotlight?story_id=%v", ta.APIContext.Config.App.URL, claimID),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

func makeArgumentMetaTags(ta *TruAPI, route string, storyID int64, argumentID int64) (*Tags, error) {
	ctx := context.Background()
	storyObj := ta.storyResolver(ctx, story.QueryStoryByIDParams{ID: storyID})
	categoryObj := ta.categoryResolver(ctx, category.QueryCategoryByIDParams{ID: storyObj.CategoryID})
	argumentObj := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: argumentID})
	creatorObj, err := ta.DBClient.TwitterProfileByAddress(argumentObj.Creator.String())
	if err != nil {
		// if error, return default
		return nil, err
	}
	return &Tags{
		Title:       fmt.Sprintf("%s made an argument in %s", creatorObj.FullName, categoryObj.Title),
		Description: html.EscapeString(stripmd.Strip(argumentObj.Body)),
		Image:       joinPath(ta.APIContext.Config.App.S3AssetsURL, defaultImage),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

func makeClaimArgumentMetaTags(ta *TruAPI, route string, claimID uint64, argumentID uint64) (*Tags, error) {
	ctx := context.Background()
	argumentObj := ta.claimArgumentResolver(ctx, queryByArgumentID{ID: argumentID})
	creatorObj, err := ta.DBClient.TwitterProfileByAddress(argumentObj.Creator.String())
	if err != nil {
		// if error, return default
		return nil, err
	}
	return &Tags{
		Title:       fmt.Sprintf("%s made an argument", creatorObj.FullName),
		Description: html.EscapeString(stripmd.Strip(argumentObj.Summary)),
		Image:       joinPath(ta.APIContext.Config.App.S3AssetsURL, defaultImage),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

func makeCommentMetaTags(ta *TruAPI, route string, storyID int64, argumentID int64, commentID int64) (*Tags, error) {
	ctx := context.Background()
	storyObj := ta.storyResolver(ctx, story.QueryStoryByIDParams{ID: storyID})
	categoryObj := ta.categoryResolver(ctx, category.QueryCategoryByIDParams{ID: storyObj.CategoryID})
	argumentObj := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: argumentID})
	comments := ta.commentsResolver(ctx, argumentObj)
	commentObj := db.Comment{}
	for _, comment := range comments {
		if comment.ID == commentID {
			commentObj = comment
		}
	}
	creatorObj, err := ta.DBClient.TwitterProfileByAddress(commentObj.Creator)
	if err != nil {
		// if error, return default
		return nil, err
	}
	return &Tags{
		Title:       fmt.Sprintf("%s posted a comment in %s", creatorObj.FullName, categoryObj.Title),
		Description: html.EscapeString(stripmd.Strip(commentObj.Body)),
		Image:       joinPath(ta.APIContext.Config.App.S3AssetsURL, defaultImage),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

func makeClaimCommentMetaTags(ta *TruAPI, route string, claimID uint64, argumentID uint64, commentID int64) (*Tags, error) {
	ctx := context.Background()
	comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: claimID})
	commentObj := db.Comment{}
	for _, comment := range comments {
		if comment.ID == commentID {
			commentObj = comment
		}
	}
	creatorObj, err := ta.DBClient.TwitterProfileByAddress(commentObj.Creator)
	if err != nil {
		// if error, return default
		return nil, err
	}
	return &Tags{
		Title:       fmt.Sprintf("%s posted a comment", creatorObj.FullName),
		Description: html.EscapeString(stripmd.Strip(commentObj.Body)),
		Image:       joinPath(ta.APIContext.Config.App.S3AssetsURL, defaultImage),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}
