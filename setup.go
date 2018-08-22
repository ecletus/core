package core

import (
	"os"
	"path/filepath"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/aghape/core/contextdata"
)

type CookiStoreFactory func(context *Context, options *sessions.Options, codecs *CookieCodec) *sessions.CookieStore

type CookieCodec struct {
	Codecs []securecookie.Codec
}

type SetupOptions struct {
	Home               string
	Root               string
	TempDir            string
	Prefix             string
	Production         bool
	CookieStoreFactory CookiStoreFactory
	CookieMaxAge       int
	CookieCodecs       []securecookie.Codec
}

type SetupConfig struct {
	home               string
	root               string
	tempDir            string
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

func (c *SetupConfig) TempDir() string {
	return c.tempDir
}

func (c *SetupConfig) Prefix() string {
	return c.prefix
}
func (c *SetupConfig) CookieStoreFactory() CookiStoreFactory {
	return c.cookieStoreFactory
}

func Setup(options SetupOptions) *SetupConfig {
	if options.Root == "" {
		options.Root = os.Getenv("ROOT")
	}

	if options.TempDir == "" {
		options.TempDir = os.Getenv("TEMP_DIR")
		if options.TempDir == "" {
			options.TempDir = filepath.Join(options.Root, "tmp")
		}
	}

	if options.Home == "" {
		options.Home = os.Getenv("HOME")
	}

	setupConfig := &SetupConfig{options.Home, options.Root, options.TempDir, options.Prefix,
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
				options.Path = context.Top().Prefix
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
	return setupConfig
}

func (s *SetupConfig) IfDev(f func() error) error {
	if s.IsDev() {
		return f()
	}
	return nil
}

func (s *SetupConfig) IfProd(f func() error) error {
	if s.IsProduction() {
		return f()
	}
	return nil
}

const SETUP_CONFIG = "qor:qor.setupConfig"

func NewCookieStore(context *Context, options *sessions.Options, codecs *CookieCodec) *sessions.CookieStore {
	config := context.SetupConfig()
	return config.cookieStoreFactory(context, options, codecs)
}

func (c *Context) SetupConfig() *SetupConfig {
	return c.Data().Get(SETUP_CONFIG).(*SetupConfig)
}

func (c *Context) SetSetupConfig(s *SetupConfig) *contextdata.ContextData {
	return c.Data().Set(SETUP_CONFIG, s)
}
