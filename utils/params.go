package utils

import (
	"path"
	"regexp"
	"strings"
)

func isAlpha(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch == '-' || ch == '!'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}

func matchPart(b byte) func(byte) bool {
	return func(c byte) bool {
		return c != b && c != '/'
	}
}

func match(s string, f func(byte) bool, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) {
		j++
	}
	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

type PathValue struct {
	Index int
	Value string
}

type PathValues struct {
	Keys   []string
	Values []*PathValue
	Map    map[string][]*PathValue
	Size   int
}

func (p *PathValues) Get(key string) *PathValue {
	values, ok := p.Map[key]
	if ok {
		return values[len(values)-1]
	}
	return nil
}

func (p *PathValues) GetString(key string) (value string, ok bool) {
	v := p.Get(key)
	if v != nil {
		value, ok = v.Value, true
	}
	return
}

func (p *PathValues) Add(key, value string) {
	_, ok := p.Map[key]
	v := &PathValue{len(p.Values), value}
	p.Values = append(p.Values, v)
	p.Size++
	if ok {
		p.Map[key] = append(p.Map[key], v)
	} else {
		p.Map[key] = []*PathValue{v}
		p.Keys = append(p.Keys, key)
	}
}

func (p *PathValues) Dict() map[string][]string {
	m := make(map[string][]string)
	for k, items := range p.Map {
		var data []string
		for _, v := range items {
			data = append(data, v.Value)
		}
		m[k] = data
	}
	return m
}

// ParamsMatch match string by param
func ParamsMatch(source string, pth string) (*PathValues, string, bool) {
	var (
		i, j int
		p    = &PathValues{Map: make(map[string][]*PathValue)}
		ext  = path.Ext(pth)
	)

	pth = strings.TrimSuffix(pth, ext)

	if ext != "" {
		p.Add(":format", strings.TrimPrefix(ext, "."))
	}

	for i < len(pth) {
		switch {
		case j >= len(source):

			if source != "/" && len(source) > 0 && source[len(source)-1] == '/' {
				return p, pth[:i], true
			}

			if source == "" && pth == "/" {
				return p, pth, true
			}
			return p, pth[:i], false
		case source[j] == ':':
			var name, val string
			var nextc byte

			name, nextc, j = match(source, isAlnum, j+1)
			val, _, i = match(pth, matchPart(nextc), i)

			if (j < len(source)) && source[j] == '[' {
				var index int
				if idx := strings.Index(source[j:], "]/"); idx > 0 {
					index = idx
				} else if source[len(source)-1] == ']' {
					index = len(source) - j - 1
				}

				if index > 0 {
					match := strings.TrimSuffix(strings.TrimPrefix(source[j:j+index+1], "["), "]")
					if reg, err := regexp.Compile("^" + match + "$"); err == nil && reg.MatchString(val) {
						j = j + index + 1
					} else {
						return nil, "", false
					}
				}
			}

			p.Add(":"+name, val)
		case pth[i] == source[j]:
			i++
			j++
		default:
			return nil, "", false
		}
	}

	if j != len(source) {
		if (len(source) == j+1) && source[j] == '/' {
			return p, pth, true
		}

		return nil, "", false
	}
	return p, pth, true
}
