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
			userProfile {
				avatarURL
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
				userProfile {
					avatarURL
					fullName
					username
				}
			}
			upvotedCount
    }
	}
`

// CommentsByClaimIDQuery fetches comments of a given claim
const CommentsByClaimIDQuery = `
  query ClaimQuery($claimId: ID!) {
	claim(id: $claimId) {
		id
		comments {
			id
			body
			creator {
				address
				userProfile {
					avatarURL
					fullName
					username
				}
			}
		}
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
	Address     string            `json:"address"`
	UserProfile UserProfileObject `json:"userProfile"`
}

// UserProfileObject defines the schema of the twitter profile
type UserProfileObject struct {
	AvatarURL string `json:"avatar_url"`
	FullName  string `json:"full_name"`
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
	Comments      []CommentObject `json:"comments"`
}

// ArgumentObject defines the schema of an argument
type ArgumentObject struct {
	ID           int64      `json:"id"`
	Body         string     `json:"body"`
	Summary      string     `json:"summary"`
	Creator      UserObject `json:"creator"`
	UpvotedCount int        `json:"upvotedCount"`
}

// CommentObject defines the schema of a comment
type CommentObject struct {
	ID      int64      `json:"id"`
	Body    string     `json:"body"`
	Creator UserObject `json:"creator"`
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

// CommentsByClaimIDResponse defines the JSON response
type CommentsByClaimIDResponse struct {
	Claim ClaimObject `json:"claim"`
}
