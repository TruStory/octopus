package main

// StoryByIDQuery fetches a story by the given ID
const StoryByIDQuery = `
  query Story($storyId: ID!) {
	story(iD: $storyId) {
	  id
	  body
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

// StoryObject defines the schema of a story
type StoryObject struct {
	ID       int64          `json:"id"`
	Body     string         `json:"body"`
	Category CategoryObject `json:"category"`
	Creator  UserObject     `json:"creator"`
}

// StoryByIDResponse defines the JSON response
type StoryByIDResponse struct {
	Story StoryObject `json:"story"`
}
