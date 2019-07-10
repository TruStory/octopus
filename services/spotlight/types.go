package main

import (
	"net/url"
)

// ClaimByIDQuery fetches a claim by the given ID
const ClaimByIDQuery = `
  query ClaimQuery($claimId: ID!) {
	claim(id: $claimId) {
		id
		community {
      id
    	name
    }
    body
    creator {
      id
			twitterProfile {
				avatarURI
				fullName
				username
			}
    }
    source
		argumentCount
	}
}
`

// ArgumentByIDQuery fetches an argument by the given ID
const ArgumentByIDQuery = `
	query ArgumentQuery($argumentId: ID!) {
    claimArgument(id: $argumentId) {
			id
			summary
			body
			creator {
				address
				twitterProfile {
					avatarURI
					fullName
					username
				}
			}
			upvotedCount
    }
	}
`

// CommunityObject defines the schema of a category
type CommunityObject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserObject defines the schema of a user
type UserObject struct {
	Address        string               `json:"address"`
	TwitterProfile TwitterProfileObject `json:"twitterProfile"`
}

// TwitterProfileObject defines the schema of the twitter profile
type TwitterProfileObject struct {
	AvatarURI string `json:"avatarURI"`
	FullName  string `json:"fullName"`
	Username  string `json:"username"`
}

// ClaimObject defines the schema of a story
type ClaimObject struct {
	ID            int64           `json:"id"`
	Body          string          `json:"body"`
	Source        string          `json:"source"`
	Community     CommunityObject `json:"community"`
	Creator       UserObject      `json:"creator"`
	ArgumentCount int             `json:"argumentCount"`
}

// ArgumentObject defines the schema of an argument
type ArgumentObject struct {
	ID           int64      `json:"id"`
	Body         string     `json:"body"`
	Summary      string     `json:"summary"`
	Creator      UserObject `json:"creator"`
	UpvotedCount int        `json:"upvotedCount"`
}

// HasSource returns whether a story has a source or not
func (claim ClaimObject) HasSource() bool {
	return claim.Source != ""
}

// GetSource returns the hostname of the source
func (claim ClaimObject) GetSource() string {
	u, err := url.Parse(claim.Source)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// ClaimByIDResponse defines the JSON response
type ClaimByIDResponse struct {
	Claim ClaimObject `json:"claim"`
}

// ArgumentByIDResponse defines the JSON response
type ArgumentByIDResponse struct {
	ClaimArgument ArgumentObject `json:"claimArgument"`
}
