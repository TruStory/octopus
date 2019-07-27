package db

import "fmt"

// CachedFeed represents a cached feed
type CachedFeed struct {
	ID   string `json:"id"`
	Feed string `json:"string"`
}

// CachedFeedByID returns a cached feed
func (c *Client) CachedFeedByID(key string) (string, error) {
	cachedFeed := new(CachedFeed)
	err := c.Model(cachedFeed).Where("id = ?", key).Limit(1).Select()
	if err != nil {
		return "[]", err
	}

	return cachedFeed.Feed, nil
}

// AddCachedFeed caches a feed
func (c *Client) AddCachedFeed(cachedFeed *CachedFeed) error {
	_, err := c.Model(cachedFeed).OnConflict("(id) DO UPDATE").
		Set("feed = ?", cachedFeed.Feed).
		Insert()
	return err
}

// DeleteCachedFeeds clears the cached_feeds table
func (c *Client) DeleteCachedFeeds() error {
	t := &CachedFeed{}
	_, err := c.Model(t).Where("feed != ''").Delete()
	fmt.Println(err)
	return err
}
