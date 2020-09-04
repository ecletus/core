package utils

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func ParseFormKey(key string) (result []interface{}, err error) {
	var (
		buf          bytes.Buffer
		cur          string
		indexStarted bool
		c            byte
		i            int
	)
	for i = 0; i < len(key); i++ {
		c = key[i]
		switch c {
		case '[':
			if indexStarted {
				return nil, fmt.Errorf("malformed key: un expected char key[%d] = '['", i)
			}
			indexStarted = true
			cur = buf.String()
			if cur != "" {
				result = append(result, cur)
				buf.Reset()
			}
		case ']':
			if !indexStarted {
				return nil, fmt.Errorf("malformed key: un expected key[%d] = ']'", i)
			}
			indexStarted = false
			cur = buf.String()
			if cur == "" {
				result = append(result, -1)
			} else {
				buf.Reset()
				var isInt = true
				for _, r := range cur {
					if !('0' <= r && r <= '9') {
						isInt = false
						break
					}
				}
				if isInt {
					var i int
					if i, err = strconv.Atoi(strings.TrimLeft(cur, "0")); err != nil {
						return nil, errors.Wrapf(err, "malformed index `%s`", cur)
					}
					result = append(result, i)
				} else {
					result = append(result, cur)
				}
			}
		case '.':
			if indexStarted {
				buf.WriteByte(c)
			} else if i == 0 || key[i-1] != ']' {
				if cur = buf.String(); cur == "" {
					return nil, fmt.Errorf("malformed key: un expected %d's key[%d] = '.'", i)
				}
				result = append(result, cur)
				buf.Reset()
			}
		default:
			buf.WriteByte(c)
		}
	}
	cur = buf.String()
	if indexStarted {
		return nil, fmt.Errorf("malformed key: unclosed index name started at key[%d]", i-len(cur)-1)
	}
	if cur != "" {
		result = append(result, cur)
	}
	return
}

