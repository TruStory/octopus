package messages

import (
	"net/url"
	"path"
)

func joinPath(baseURL, route string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	u.Path = path.Join(u.Path, route)
	return u.String()
}