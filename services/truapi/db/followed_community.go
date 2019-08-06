package db

import (
	"time"
)

// FollowedCommunity represents a community that an user follows.
type FollowedCommunity struct {
	ID             int64     `json:"id"`
	Address        string    `json:"address"`
	CommunityID    string    `json:"community_id"`
	FollowingSince time.Time `json:"following_since"`
	Timestamps
}

// follows tells whether a contains x.
func follows(followedCommunities []FollowedCommunity, community string) bool {
	for _, c := range followedCommunities {
		if community == c.CommunityID {
			return true
		}
	}
	return false
}

// FollowCommunities registers a community for a given user as followed.
func (c *Client) FollowCommunities(address string, communities []string) error {
	following, err := c.FollowedCommunities(address)
	if err != nil {
		return err
	}
	followedCommunities := make([]FollowedCommunity, 0)
	dateTime := time.Now()
	for _, c := range communities {
		if follows(following, c) {
			continue
		}
		followedCommunities = append(followedCommunities, FollowedCommunity{
			Address:        address,
			CommunityID:    c,
			FollowingSince: dateTime,
		})
	}
	if len(followedCommunities) == 0 {
		return nil
	}
	_, err = c.Model(&followedCommunities).Insert()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) FollowedCommunities(address string) ([]FollowedCommunity, error) {
	followedCommunities := make([]FollowedCommunity, 0)
	err := c.Model(&followedCommunities).Where("address = ?", address).Select()
	if err != nil {
		return nil, err
	}
	return followedCommunities, nil
}

func (c *Client) UnfollowCommunity(address, communityID string) error {
	following, err := c.FollowedCommunities(address)
	if err != nil {
		return err
	}
	if !follows(following, communityID) {
		return ErrNotFollowingCommunity
	}
	if len(following) <= 1 {
		return ErrFollowAtLeastOneCommunity
	}
	followedCommunity := &FollowedCommunity{}
	_, err = c.Model(followedCommunity).Where("address = ? ", address).
		Where("community_id = ?", communityID).
		Delete()
	return err
}
