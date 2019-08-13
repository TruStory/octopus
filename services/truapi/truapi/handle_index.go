package truapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
	"strconv"

	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/truchain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stripmd "github.com/writeas/go-strip-markdown"
)

const (
	defaultDescription = "TruStory is a social network to debate claims with skin in the game"
	previewDirectory   = "communities/previews" // full url format: S3_URL/communities/previews/PREVIEW.jpeg
)

var (
	claimRegex         = regexp.MustCompile("/claim/([0-9]+)/?$")
	claimArgumentRegex = regexp.MustCompile("/claim/([0-9]+)/argument/([0-9]+)/?$")
	claimCommentRegex  = regexp.MustCompile("/claim/([0-9]+)/comment/([0-9]+)/?$")
	communityRegex     = regexp.MustCompile("/community/([^/]+)")
)

// Tags defines the struct containing all the request Meta Tags for a page
type Tags struct {
	Title       string
	Description string
	Image       string
	URL         string
}

// CompileIndexFile replaces placeholders in index.html file with dynamic values
func CompileIndexFile(ta *TruAPI, index []byte, route string) string {
	indexWithMetaTags := renderMetaTags(ta, index, route)

	mixpanelToken := ta.APIContext.Config.App.MixpanelToken
	compiled := bytes.Replace(indexWithMetaTags, []byte("$PLACEHOLDER__MIXPANEL_TOKEN"), []byte(mixpanelToken), -1)
	return string(compiled)
}

// renderMetaTags replaces <meta> placeholders in index.html file with dynamic values
func renderMetaTags(ta *TruAPI, index []byte, route string) []byte {
	// /claim/xxx
	matches := claimRegex.FindStringSubmatch(route)
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

	// /claim/xxx/comment/xxx
	matches = claimCommentRegex.FindStringSubmatch(route)
	if len(matches) == 3 {
		// replace placeholder with claim details, where claim id is in matches[1]
		claimID, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		commentID, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			// if error, return the default tags
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		metaTags, err := makeClaimCommentMetaTags(ta, route, claimID, commentID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}
		return compile(index, *metaTags)
	}

	matches = communityRegex.FindStringSubmatch(route)
	if len(matches) == 2 {
		// replace placeholder with community details
		communityID := matches[1]

		metaTags, err := makeCommunityMetaTags(ta, route, communityID)
		if err != nil {
			return compile(index, makeDefaultMetaTags(ta, route))
		}

		return compile(index, *metaTags)
	}

	return compile(index, makeDefaultMetaTags(ta, route))
}

// compiles the index file with the variables
func compile(index []byte, tags Tags) []byte {
	compiled := bytes.Replace(index, []byte("$PLACEHOLDER__TITLE"), []byte(tags.Title), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__DESCRIPTION"), []byte(tags.Description), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__IMAGE"), []byte(tags.Image), -1)
	compiled = bytes.Replace(compiled, []byte("$PLACEHOLDER__URL"), []byte(tags.URL), -1)

	return compiled
}

// makes the default meta tags
func makeDefaultMetaTags(ta *TruAPI, route string) Tags {
	previewsDirectory := joinPath(ta.APIContext.Config.App.S3AssetsURL, previewDirectory)
	return Tags{
		Title:       ta.APIContext.Config.App.Name,
		Description: defaultDescription,
		Image:       joinPath(previewsDirectory, "feed.jpg"),
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

	// HACK: video debate thumbnail
	image := fmt.Sprintf("%s/api/v1/spotlight?claim_id=%v", ta.APIContext.Config.App.URL, claimID)
	if claimID == 824 {
		image = "https://s3-us-west-1.amazonaws.com/trustory/images/22405a4351507698.jpg"
	} else if claimID == 893 {
		image = "https://s3-us-west-1.amazonaws.com/trustory/images/407a1236454a9a59.jpg"
	}

	return &Tags{
		Title:       html.EscapeString(claimObj.Body),
		Description: fmt.Sprintf("%d participant%s, %s %s", totalParticipants, totalParticipantsPlural, totalStaked.Amount.Quo(sdk.NewInt(app.Shanev)), db.CoinDisplayName),
		Image:       image,
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

func makeClaimArgumentMetaTags(ta *TruAPI, route string, claimID uint64, argumentID uint64) (*Tags, error) {
	ctx := context.Background()
	argumentObj := ta.claimArgumentResolver(ctx, queryByArgumentID{ID: argumentID})
	creatorObj, err := ta.DBClient.UserByAddress(argumentObj.Creator.String())
	if creatorObj == nil || err != nil {
		// if error, return default
		return nil, err
	}
	return &Tags{
		Title:       fmt.Sprintf("%s made an argument", "@"+creatorObj.Username),
		Description: html.EscapeString(stripmd.Strip(argumentObj.Summary)),
		Image:       fmt.Sprintf("%s/api/v1/spotlight?argument_id=%v", ta.APIContext.Config.App.URL, argumentID),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

func makeClaimCommentMetaTags(ta *TruAPI, route string, claimID uint64, commentID int64) (*Tags, error) {
	ctx := context.Background()
	comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: claimID})
	commentObj := db.Comment{}
	for _, comment := range comments {
		if comment.ID == commentID {
			commentObj = comment
		}
	}
	creatorObj, err := ta.DBClient.UserByAddress(commentObj.Creator)
	if creatorObj == nil || err != nil {
		// if error, return default
		return nil, err
	}
	return &Tags{
		Title:       fmt.Sprintf("%s posted a comment", "@"+creatorObj.Username),
		Description: html.EscapeString(stripmd.Strip(commentObj.Body)),
		Image:       fmt.Sprintf("%s/api/v1/spotlight?claim_id=%v&comment_id=%v", ta.APIContext.Config.App.URL, claimID, commentID),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}

// makes the community meta tags
func makeCommunityMetaTags(ta *TruAPI, route string, communityID string) (*Tags, error) {
	ctx := context.Background()
	community := ta.communityResolver(ctx, queryByCommunityID{CommunityID: communityID})
	if community == nil {
		return nil, errors.New("Community not found")
	}
	previewsDirectory := joinPath(ta.APIContext.Config.App.S3AssetsURL, previewDirectory)

	return &Tags{
		Title:       fmt.Sprintf("%s Community on %s", community.Name, ta.APIContext.Config.App.Name),
		Description: community.Description,
		Image:       joinPath(previewsDirectory, fmt.Sprintf("%s.jpg", communityID)),
		URL:         joinPath(ta.APIContext.Config.App.URL, route),
	}, nil
}
