package url

import (
	"strings"
	"fmt"
	"path"
	nurl "net/url"
)

// PatchURL updates the query part of the request url.
//     PatchURL("google.com","key","value") => "google.com?key=value"
func PatchURL(originalURL string, params ...interface{}) (patchedURL string, err error) {
	url, err := nurl.Parse(originalURL)
	if err != nil {
		return
	}

	query := url.Query()
	for i := 0; i < len(params)/2; i++ {
		// Check if params is key&value pair
		key := fmt.Sprintf("%v", params[i*2])
		value := fmt.Sprintf("%v", params[i*2+1])

		if value == "" {
			query.Del(key)
		} else {
			query.Set(key, value)
		}
	}

	url.RawQuery = query.Encode()
	patchedURL = url.String()
	return
}

// JoinURL updates the path part of the request url.
//     JoinURL("google.com", "admin") => "google.com/admin"
//     JoinURL("google.com?q=keyword", "admin") => "google.com/admin?q=keyword"
func JoinURL(originalURL string, paths ...interface{}) (joinedURL string, err error) {
	u, err := nurl.Parse(originalURL)
	if err != nil {
		return
	}

	var urlPaths = []string{u.Path}
	for _, p := range paths {
		urlPaths = append(urlPaths, fmt.Sprint(p))
	}

	if strings.HasSuffix(strings.Join(urlPaths, ""), "/") {
		u.Path = path.Join(urlPaths...) + "/"
	} else {
		u.Path = path.Join(urlPaths...)
	}

	joinedURL = u.String()
	return
}
