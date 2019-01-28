package models

// model of the url object structure
type Url struct {
	Name      	string     	`bson:"image_name" json:"image_name"`
	Content    	string      `bson:"content_type" json:"content_type"`
	}