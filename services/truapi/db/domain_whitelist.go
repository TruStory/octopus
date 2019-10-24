package db

import "fmt"

// DomainWhitelist represets a whitelisted domain
type DomainWhitelist struct {
	ID     int64  `json:"id"`
	Domain string `json:"domain"`
	Timestamps
}

func (c *Client) IsDomainWhitelisted(domain string) (bool, error) {
	fmt.Println("checking domain", domain)
	count, err := c.DB.Model((*DomainWhitelist)(nil)).
		Where("domain = ?", domain).Count()

	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}
