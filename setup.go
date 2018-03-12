package qor

import (
	"errors"
	"os"

	"github.com/gorilla/sessions"
	"github.com/gorilla/securecookie"
)

type CookiStoreFactory func(context *Context, options *sessions.Options, codecs *CookieCodec) *sessions.CookieStore

type CookieCodec struct {
	Codecs []securecookie.Codec
}

type SetupOptions struct {
	Home               string
	Root               string
	Prefix             string
	Production         bool
	CookieStoreFactory CookiStoreFactory
	CookieMaxAge       int
	CookieCodecs       []securecookie.Codec
}

type SetupConfig struct {
	home               string
	root               string
	prefix             string
	production         bool
	cookieStoreFactory CookiStoreFactory
	cookieMaxAge       int
	cookieCodecs       []securecookie.Codec
}

func (c *SetupConfig) IsProduction() bool {
	return c.production
}

func (c *SetupConfig) IsDev() bool {
	return !c.production
}

func (c *SetupConfig) Home() string {
	return c.home
}

func (c *SetupConfig) Root() string {
	return c.root
}

func (c *SetupConfig) Prefix() string {
	return c.prefix
}

func CONFIG() *SetupConfig {
	return setupConfig
}

func SetupCheck() *SetupConfig {
	if setupConfig == nil {
		panic(errors.New("qor is not initialized."))
	}
	return setupConfig
}

var setupConfig *SetupConfig

func Setup(options SetupOptions) {
	if setupConfig != nil {
		panic("qor has be initialized.")
	}

	if options.Root == "" {
		options.Root = os.Getenv("ROOT")
	}

	if options.Home == "" {
		options.Home = os.Getenv("HOME")
	}

	setupConfig = &SetupConfig{options.Home, options.Root, options.Prefix,
		options.Production, options.CookieStoreFactory, options.CookieMaxAge,
		options.CookieCodecs}

	if setupConfig.cookieMaxAge == 0 {
		setupConfig.cookieMaxAge = 86400 * 30
	}

	if len(setupConfig.cookieCodecs) == 0 {
		setupConfig.cookieCodecs = securecookie.CodecsFromPairs([]byte("secret"))
	}

	if setupConfig.cookieStoreFactory == nil {
		setupConfig.cookieStoreFactory = func(context *Context, options *sessions.Options, codecs *CookieCodec) *sessions.CookieStore {
			if options == nil {
				options = &sessions.Options{}
			}
			if options.Path == "" {
				options.Path = context.GetTop().Prefix
			}
			if options.MaxAge == 0 {
				options.MaxAge = setupConfig.cookieMaxAge
			}

			cc := setupConfig.cookieCodecs

			if codecs != nil {
				cc = codecs.Codecs
			}

			cs := &sessions.CookieStore{
				Codecs:  cc,
				Options: options,
			}

			cs.MaxAge(cs.Options.MaxAge)
			return cs
		}
	}
}

func IfDev(f func()) {
	if SetupCheck(); setupConfig.IsDev() {
		f()
	}
}

func IfProd(f func()) {
	if SetupCheck(); setupConfig.IsProduction() {
		f()
	}
}

func NewCookieStore(context *Context, options *sessions.Options, codecs *CookieCodec) *sessions.CookieStore {
	return SetupCheck().cookieStoreFactory(context, options, codecs)
}
