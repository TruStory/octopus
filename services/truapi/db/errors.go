package db

import "errors"

// Errors for db module.
var (
	ErrInvalidAddress            = errors.New("invalid address")
	ErrFollowAtLeastOneCommunity = errors.New("should follow at least one community")
	ErrNotFollowingCommunity     = errors.New("user doesn't follow community")
)
