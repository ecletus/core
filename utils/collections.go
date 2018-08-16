package utils

import (
	"strconv"
)

func Tuples(args... string) (r [][]string) {
	l := len(args)
	var i int
	for ;i < l; i = i+2 {
		r = append(r, []string{args[i], args[i+1]})
	}
	return
}

func TuplesIndex(args... string) (r [][]string) {
	for i, arg := range args {
		r = append(r, []string{strconv.Itoa(i), arg})
	}
	return
}

