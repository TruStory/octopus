package main

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

// Argument defines the schema of either the backings or the challenges
type Argument struct {
	Creator UserObject `json:"creator"`
}

// StoryObject defines the schema of a story
type StoryObject struct {
	ID         int64          `json:"id"`
	Body       string         `json:"body"`
	Source     string         `json:"source"`
	Category   CategoryObject `json:"category"`
	Creator    UserObject     `json:"creator"`
	Backings   []Argument     `json:"backings"`
	Challenges []Argument     `json:"challenges"`
}

// GetArgumentCount returns the total count of backings + challenges
func (story StoryObject) GetArgumentCount() int {
	return len(story.Backings) + len(story.Challenges)
}

// HasSource returns whether a story has a source or not
func (story StoryObject) HasSource() bool {
	return story.Source != ""
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
