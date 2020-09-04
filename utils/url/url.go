package url

import (
	"strings"
	"fmt"
	"path"
	nurl "net/url"
)

// PatchURL updates the query part of the request url.
//     PatchURL("google.com","key","value") => "google.com?key=value"
//     PatchURL("google.com?key=value","key","value") => "google.com"
//     PatchURL("google.com", "~a[]","b") => "google.com?a[]=b"
//     PatchURL("google.com?a[]=b","~a[]","c") => "google.com?a[]=b&a[]=c"
//     PatchURL("google.com?a[]=b&a[]=c","~a[]","c") => "google.com?a[]=b"
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

		if key[0] == '~' {
			key = key[1:]
			if values, ok := query[key]; ok {
				var (
					newValues []string
					has bool
				)
				for _, v := range values {
					if v == value {
						has = true
					} else {
						newValues = append(newValues, v)
					}
				}
				if !has {
					newValues = append(newValues, value)
				}
				if len(newValues) == 0 {
					query.Del(key)
				} else {
					query[key] = newValues
				}
			} else {
				query.Set(key, value)
			}
		} else if value == "" {
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

func MustJoinURL(originalURL string, paths ...interface{}) (joinedURL string) {
	joinedURL, _ = JoinURL(originalURL, paths...)
	return
}
