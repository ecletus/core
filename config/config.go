package config

import (
	"github.com/moisespsena/go-i18n-modular/i18nmod"
)

// Config qor config struct
type Config struct {
	Translator *i18nmod.Translator
}

func NewConfig() *Config {
	return &Config{Translator: &i18nmod.Translator{}}
}
