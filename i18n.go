package core

import (
	"os"
	"strings"
)

var (
	DefaultLocale string
)

func init() {
	lang := os.Getenv("LANG")

	if len(lang) >= 5 {
		DefaultLocale = strings.Replace(strings.Split(lang, ".")[0], "_", "-", 1)
	}
}

type i18nGroup struct {
	Prev  *i18nGroup
	Value string
}
