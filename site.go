package core

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/ecletus/roles"
	"github.com/moisespsena-go/logging"
	"golang.org/x/text/message"

	"github.com/moisespsena-go/maps"

	"github.com/ecletus/core/db/dbconfig"

	"github.com/moisespsena-go/getters"

	"github.com/ecletus/core/site_config"

	"github.com/ecletus/oss"
	"github.com/ecletus/oss/filesystem"
	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/xroute"

	"github.com/ecletus/core/db"
	"github.com/moisespsena-go/aorm"
)

type SiteConfigType uint8

var DefaultLang = "pt"

const (
	SiteConfigBasic SiteConfigType = iota
	SiteConfigContextFactory

	SiteInitOptionsKey = "core:site_init_options"
)

type SiteInitOptions struct {
	DBAutoConnectDisabled bool
}

type Site struct {
	name                    string
	timeLocation            *time.Location
	configGetter            ConfigGetter
	configSetter            ConfigSetter
	basicConfig             site_config.Config
	Dbs                     map[string]*DB
	mediaStorages           map[string]oss.NamedStorageInterface
	systemStorage           *filesystem.FileSystem
	storageNames            *oss.Names
	handler                 xroute.ContextHandler
	contextFactory          *ContextFactory
	Data                    maps.Map
	Middlewares             xroute.Middlewares
	postInitCallbacks       []func()
	handlerChangedCallbacks []func(oldh, newh xroute.ContextHandler)
	onDestroyCallbacks      []func()
	initialized,
	registered bool
	Log                    logging.Logger
	PermissionModeProvider PermissionModeProvider
	role                   *roles.Role
}

func (this *Site) ConfigSetter() ConfigSetter {
	return this.configSetter
}

func (this *Site) SetConfigSetter(configSetter ConfigSetter) {
	this.configSetter = configSetter
}

func NewSite(name string, basicConfig site_config.Config, configGetter getters.InterfaceGetter, cf *ContextFactory) *Site {
	getter := MapstructureRawGetter2InterfaceGetter(basicConfig.Raw.Getter(), func(key, value, dest interface{}, err error) {
		log.Errorf("site %q: decode config %q => '%s' into %T failed: %s", name, key, dest, err.Error())
	})
	s := &Site{
		name:           name,
		configGetter:   getters.MultipleGetter{configGetter, getter},
		contextFactory: cf,
		basicConfig:    basicConfig,
		Dbs:            make(map[string]*DB),
		mediaStorages:  make(map[string]oss.NamedStorageInterface),
		systemStorage:  filesystem.New(&filesystem.Config{RootDir: basicConfig.RootDir}),
		storageNames:   oss.NewNames(),
		handler:        xroute.NewMux(),
	}
	s.PermissionModeProvider = &SitePermissionModeProvider{
		Site: s,
	}
	if basicConfig.TimeLocation != "" {
		var err error
		if s.timeLocation, err = time.LoadLocation(basicConfig.TimeLocation); err != nil {
			log.Warnf("locate time location %q from site %q failed: %s", basicConfig.TimeLocation, name, err)
			s.timeLocation = time.Local
		}
	} else {
		s.timeLocation = time.Local
	}

	return s
}

func (this *Site) PostInit(f ...func()) {
	if this.initialized {
		for _, f := range f {
			f()
		}
		return
	}
	this.postInitCallbacks = append(this.postInitCallbacks, f...)
}

func (this *Site) OnDestroy(f ...func()) {
	this.onDestroyCallbacks = append(this.onDestroyCallbacks, f...)
}

func (this *Site) IsRegistered() bool {
	return this.registered
}

func (this *Site) TimeLocation() *time.Location {
	return this.timeLocation
}

func (this *Site) Handler() xroute.ContextHandler {
	return this.handler
}

func (this *Site) SetHandler(handler xroute.ContextHandler) {
	if old := this.handler; old != handler {
		this.handler = handler
		for _, f := range this.handlerChangedCallbacks {
			f(old, handler)
		}
	}
}

func (this *Site) HandlerChanged(f ...func(oldh, newh xroute.ContextHandler)) {
	this.handlerChangedCallbacks = append(this.handlerChangedCallbacks, f...)
}

func (this *Site) GetConfig(key interface{}) (value interface{}, ok bool) {
	return this.configGetter.Get(key)
}

func (this *Site) GetConfigInterface(key, dest interface{}) (ok bool) {
	return this.configGetter.GetInterface(key, dest)
}

func (this *Site) MustConfig(key interface{}) (value interface{}) {
	value, _ = this.configGetter.Get(key)
	return
}

func (this *Site) SetConfig(key string, value interface{}) (err error) {
	return this.configSetter.Set(key, value)
}

type SetupDB func(setup func(db *DB) error) (err error)

func (this *Site) Role() *roles.Role {
	return this.role
}

func (this *Site) BasicConfig() *site_config.Config {
	return &this.basicConfig
}

func (this *Site) Config() SiteConfig {
	return &struct {
		getters.InterfaceGetter
		ConfigSetter
	}{this.configGetter, this.configSetter}
}

func (this *Site) Name() string {
	return this.name
}

func (this *Site) Title() string {
	return this.basicConfig.Title
}

func (this *Site) Init(opts *SiteInitOptions) (err error) {
	this.mediaStorages["default"] = &oss.NamedStorage{this.systemStorage, "default"}
	if this.basicConfig.MediaStorage != nil {
		for name, storageConfig := range this.basicConfig.MediaStorage {
			this.mediaStorages[name] = &oss.NamedStorage{storageConfig["@storage"].(oss.StorageInterface), name}
		}
	}

	if this.basicConfig.Db != nil && !opts.DBAutoConnectDisabled {
		for name, dbConfig := range this.basicConfig.Db {
			func(dbConfig *dbconfig.DBConfig) {
				var d *DB
				d = &DB{
					Site:   this,
					Config: dbConfig,
					Name:   name,
					open: func(ctx context.Context) (DB *aorm.DB, err error) {
						if DB, err = db.SystemFactories.Factory(ctx, dbConfig); err == nil {
							DB = DB.Inside(PREFIX+".site["+this.Name()+"]", "DB["+name+"]").Set(PREFIX+".site", this).Set(PREFIX+".db", d)
						}
						if dbConfig.DryRun {
							DB = DB.Opt(aorm.OptDryCommit())
						}
						DB.Location = this.timeLocation
						return
					},
				}

				if err := d.Open(context.Background()); err != nil {
					panic(err)
				}
				// d.Raw = &RawDB{DB: d}
				this.Dbs[name] = d
			}(dbConfig)
		}
	}

	this.role = &roles.Role{}
	this.role.Register(roles.Global.Descriptors().Intersection(this.PermissionModeProvider.Provides().Strings())...)

	for _, f := range this.postInitCallbacks {
		f()
	}

	this.postInitCallbacks = nil
	this.initialized = true
	return
}

func (this *Site) InitOrPanic(opts *SiteInitOptions) *Site {
	err := this.Init(opts)
	if err != nil {
		this.Log.Errorf("Init this %q failed: %v", this.Name(), err)
	}
	return this
}

func (this *Site) StorageNames() *oss.Names {
	return this.storageNames
}

func (this *Site) GetDB(name string) *DB {
	return this.Dbs[name]
}

func (this *Site) GetDBOrSystem(name string) *DB {
	db, ok := this.Dbs[name]
	if !ok {
		return this.GetSystemDB()
	}
	return db
}

func (this *Site) GetSystemDB() *DB {
	return this.GetDB(dbconfig.DB_SYSTEM)
}

func (this *Site) EachDB(f func(db *DB) error) (err error) {
	for _, db := range this.Dbs {
		if err = f(db); err != nil {
			if err == StopDBIteration {
				return nil
			}
			return
		}
	}
	return
}

func (this *Site) SystemStorage() *filesystem.FileSystem {
	return this.systemStorage
}

func (this *Site) MediaStorages() map[string]oss.NamedStorageInterface {
	return this.mediaStorages
}

func (this *Site) GetMediaStorage(name string) oss.NamedStorageInterface {
	return this.mediaStorages[name]
}

func (this *Site) GetMediaStorageOrDefault(name string) oss.NamedStorageInterface {
	s := this.GetMediaStorage(name)
	if s == nil {
		return this.GetDefaultMediaStorage()
	}
	return s
}

func (this *Site) GetDefaultMediaStorage() oss.NamedStorageInterface {
	return this.GetMediaStorage("default")
}

func (this *Site) PrepareContext(ctx *Context) *Context {
	ctx.AsTop()
	ctx.Site = this
	DB := this.GetSystemDB().DB
	if ctx.Request == nil {
		var err error
		ctx.Request, err = http.NewRequest(http.MethodGet, this.basicConfig.PublicURL, nil)
		if err != nil {
			log.Errorf("Site %q: prepare context: new request failed: %v", err.Error())
		}
		ctx.Request.RequestURI = ctx.Request.URL.RequestURI()
		ctx.Prefix = ctx.Request.URL.Path
		ctx.Request.URL = &url.URL{Path: "/"}
		if ctx.StaticURL == "" {
			ctx.StaticURL = this.basicConfig.StaticURL
		}
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), CONTEXT_KEY, ctx))
		ctx.OriginalURL, _ = url.Parse(this.basicConfig.PublicURL)
	} else {
		DB = DB.Inside("Req[" + ctx.Request.Method + " " + ctx.Request.RequestURI + "]")
		if ctx.RouteContext == nil {
			ctx.RouteContext = xroute.NewRouteContext()
		}
		if ctx.StaticURL == "" || ctx.StaticURL[0] == '/' {
			ctx.StaticURL += "/static"
		}
	}

	if ctx.Lang == "" {
		lang, _ := ctx.Request.Cookie("lang")
		accept := ctx.Request.Header.Get("Accept-Language")

		fallback := this.basicConfig.Lang
		if fallback == "" {
			fallback = DefaultLang
		}
		tag := message.MatchLanguage(lang.String(), accept, fallback)
		ctx.LangTag = &tag
	}

	ctx.Role = this.role.Copy()
	ctx.SetRequestTime(time.Now())
	ctx.SetDB(DB.Set(CONTEXT_KEY, ctx))
	return ctx
}

func (this *Site) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if strings.HasPrefix(r.URL.Path, PATH_MEDIA) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, PATH_MEDIA)
		storage := this.GetDefaultMediaStorage()
		storage.ServeHTTP(w, r)
		return
	}
	prefix := httpu.PrefixR(r)
	r, context := this.contextFactory.NewContextFromRequestPair(w, r, prefix)
	this.PrepareContext(context)
	rctx.Data[CONTEXT_KEY] = context
	rctx.ChainRequestSetters[CONTEXT_KEY] = xroute.NewChainRequestSetter(func(chain *xroute.ChainHandler, r *http.Request) {
		chain.Context.Data[CONTEXT_KEY].(*Context).SetRequest(r)
	})
	this.handler.ServeHTTPContext(w, r, rctx)
}

func (this *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	this.handler.ServeHTTPContext(w, r, nil)
}

func (this *Site) NewContext() *Context {
	return this.PrepareContext(NewContext())
}

func (this *Site) PublicURL(p ...string) string {
	if len(p) > 0 {
		return strings.Join(append([]string{this.basicConfig.PublicURL}, strings.TrimPrefix(path.Join(p...), "/")), "/")
	}
	return this.basicConfig.PublicURL
}

func (this *Site) PublicURLf(p ...interface{}) string {
	parts := make([]string, len(p))
	for i, part := range p {
		if pt, ok := part.(string); ok {
			parts[i] = pt
		} else {
			parts[i] = fmt.Sprint(pt)
		}
	}
	return this.PublicURL(parts...)
}

func (this *Site) TextRender(v string) (s string, err error) {
	var tmpl *template.Template
	if tmpl, err = template.New("<no name>").Parse(v); err != nil {
		return
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, this)
	if err == nil {
		s = out.String()
	}
	return
}

func GetSiteFromDB(db *aorm.DB) *Site {
	s, _ := db.Get(PREFIX + ".site")
	return s.(*Site)
}

func GetSiteFromRequest(r *http.Request) *Site {
	if ctx := ContextFromRequest(r); ctx != nil {
		return ctx.Site
	}
	return nil
}

func GetDBFromDB(db *aorm.DB) *DB {
	s, _ := db.Get(PREFIX + ".db")
	return s.(*DB)
}
