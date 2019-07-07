package main

import (
	"net/url"
)

// StoryByIDQuery fetches a story by the given ID
const StoryByIDQuery = `
  query Story($storyId: ID!) {
	story(iD: $storyId) {
		id
		body
		source
	  category {
			title
			id
	  }
	  creator {
			address
			twitterProfile {
				avatarURI
				fullName
				username
			}
		}
		backings {
			argument {
				creator {
					address
					twitterProfile {
						avatarURI
						fullName
						username
					}
				}
			}
      creator {
        address
				twitterProfile {
					avatarURI
					fullName
					username
				}
      }
    }
    challenges {
			argument {
				creator {
					address
					twitterProfile {
						avatarURI
						fullName
						username
					}
				}
			}
      creator {
        address
				twitterProfile {
					avatarURI
					fullName
					username
				}
      }
    }
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

// CategoryObject defines the schema of a category
type CategoryObject struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
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

// Argument defines the schema of the argument
type Argument struct {
	Creator UserObject `json:"creator"`
}

// StoryAction defines the schema of either the backings or the challenges
type StoryAction struct {
	Argument Argument   `json:"argument"`
	Creator  UserObject `json:"creator"`
}

// StoryObject defines the schema of a story
type StoryObject struct {
	ID         int64          `json:"id"`
	Body       string         `json:"body"`
	Source     string         `json:"source"`
	Category   CategoryObject `json:"category"`
	Creator    UserObject     `json:"creator"`
	Backings   []StoryAction  `json:"backings"`
	Challenges []StoryAction  `json:"challenges"`
}

// ArgumentObject defines the schema of an argument
type ArgumentObject struct {
	ID          int64      `json:"id"`
	Body        string     `json:"body"`
	Summary     string     `json:"summary"`
	Creator     UserObject `json:"creator"`
	UpvoteCount int        `json:"upvoteCount"`
}

// GetArgumentCount returns the total count of backings + challenges
func (story StoryObject) GetArgumentCount() int {
	count := 0

	for _, backing := range story.Backings {
		if backing.Creator.Address == backing.Argument.Creator.Address {
			count++
		}
	}

	for _, challenge := range story.Challenges {
		if challenge.Creator.Address == challenge.Argument.Creator.Address {
			count++
		}
	}
	return count
}

// HasSource returns whether a story has a source or not
func (story StoryObject) HasSource() bool {
	return story.Source != ""
}

// GetSource returns the hostname of the source
func (story StoryObject) GetSource() string {
	u, err := url.Parse(story.Source)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// GetTopParticipants returns the top participants of a story
func (story StoryObject) GetTopParticipants() []UserObject {
	limit := 3
	var participants []UserObject

	participants = append(participants, story.Creator)
	for i := 0; len(participants) < limit && i < len(story.Backings); i++ {
		participants = append(participants, story.Backings[i].Creator)
	}
	for i := 0; len(participants) < limit && i < len(story.Challenges); i++ {
		participants = append(participants, story.Challenges[i].Creator)
	}

	return participants
}

// StoryByIDResponse defines the JSON response
type StoryByIDResponse struct {
	Story StoryObject `json:"story"`
}

// ArgumentByIDResponse defines the JSON response
type ArgumentByIDResponse struct {
	ClaimArgument ArgumentObject `json:"claimArgument"`
}
