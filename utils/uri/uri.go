package uri

import "strings"

func Join(parts... string) string {
	l := len(parts)
	if l > 1 {
		prefix := strings.TrimSuffix(parts[0], "/")
		parts = parts[1:]
		l--
		l--
		var i int
		for ;i < l; i++ {
			parts[i] = strings.Trim(parts[i], "/")
		}
		parts[i] = strings.TrimPrefix(parts[i], "/")
		path := strings.Join(parts, "/")
		if path != "" {
			return prefix + "/" + path
		}
		return prefix
	} else {
		return parts[0]
	}
}

func Clean(path []string) (r []string) {
	for _, p := range path {
		if p != "" {
			r = append(r, p)
		}
	}
	return
}
