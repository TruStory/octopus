package regex

import (
	"regexp"
)

// RegexValidEmail for valid email
var RegexValidEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// RegexValidUsername for valid username
var RegexValidUsername = regexp.MustCompile("^[a-zA-Z0-9_]{1,28}$")

// Some helper methods based on the above regex

// IsValidEmail returns whether an email matches the valid email regex or not
func IsValidEmail(email string) bool {
	return RegexValidEmail.MatchString(email)
}

// IsValidUsername returns whether an username matches the valid username regex or not
func IsValidUsername(username string) bool {
	return RegexValidUsername.MatchString(username)
}
