package str

import "strings"

func TrimSpaceOfStrings(s ...string) []string {
	for i, v := range s {
		s[i] = strings.TrimSpace(v)
	}
	return s
}

func NotEmpties(s ...string) (r []string) {
	for _, s := range s {
		if s != "" {
			r = append(r, s)
		}
	}
	if len(r) == len(s) {
		return s
	}
	return
}
