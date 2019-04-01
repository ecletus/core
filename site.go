package core

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/moisespsena-go/aorm/assigners"

	"github.com/moisespsena-go/error-wrap"

	"errors"

	"github.com/ecletus/core/config"
	"github.com/ecletus/core/db"
	"github.com/ecletus/oss"
	"github.com/ecletus/oss/filesystem"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/xroute"
)

const DB_SYSTEM = "system"

var (
	StopSiteIteration = errors.New("stop site iteration")
	StopDBIteration   = errors.New("stop db iteration")
)

type SitesReaderInterface interface {
	Get(siteName string) SiteInterface
	GetOrError(siteName string) (SiteInterface, error)
	All() []SiteInterface
	Sorted() []SiteInterface
	Names() []string
	EachOrAll(siteName string, cb func(site SiteInterface) (err error)) error
	Each(cb func(site SiteInterface) (err error)) error
}

type SitesReader map[string]SiteInterface

func (r SitesReader) Get(siteName string) SiteInterface {
	s, _ := r[siteName]
	return s
}

func (r SitesReader) GetOrError(siteName string) (SiteInterface, error) {
	s, ok := r[siteName]
	if !ok {
		return nil, fmt.Errorf("Site %q does not exists.", siteName)
	}
	return s, nil
}

func (r SitesReader) All() (sites []SiteInterface) {
	for _, s := range r {
		sites = append(sites, s)
	}
	return
}

func (r SitesReader) Names() (names []string) {
	for k := range r {
		names = append(names, k)
	}
	return
}

func (r SitesReader) Sorted() []SiteInterface {
	sites := r.All()
	sort.Slice(sites, func(a, b int) bool {
		return sites[a].Name() < sites[b].Name()
	})
	return sites
}

func (r SitesReader) Each(cb func(site SiteInterface) (err error)) (err error) {
	for _, s := range r {
		if err = cb(s); err != nil {
			if err == StopSiteIteration {
				return nil
			}
			return errwrap.Wrap(err, "Site %q", s.Name())
		}
	}
	return nil
}

func (r SitesReader) EachOrAll(siteName string, cb func(site SiteInterface) (err error)) error {
	if siteName == "" || siteName == "*" {
		return r.Each(cb)
	}

	site, err := r.GetOrError(siteName)

	if err != nil {
		return err
	}

	return cb(site)
}

type RawDB struct {
	DB   *DB
	conn db.RawDBConnection
	lock sync.Mutex
}

func (r *RawDB) Open(ctx context.Context) (conn db.RawDBConnection, err error) {
	if conn, err = db.SystemRawFactories[r.DB.Config.Adapter](ctx, r.DB.Config); err == nil {
		err = conn.Open()
	}
	return
}

type RawDBConnectError struct {
	message string
}

func (r *RawDBConnectError) Error() string {
	return r.message
}

func (r *RawDB) Do(f func(con db.RawDBConnection)) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.conn == nil {
		var err error
		r.conn, err = r.Open(PrepareRawDBContext(r.DB))
		if err != nil {
			panic(&RawDBConnectError{fmt.Sprintf("github.com/ecletus/qor.site: Site %q: Failed to "+
				"open RAW connection of DB %q: %v", r.DB.Site.Name(), r.DB.Name, err)})
		}
	}
	r.conn.Do(f)
}

func (r *RawDB) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.conn != nil {
		err := r.conn.Close()
		if err == nil {
			r.conn = nil
		} else {
			panic(&RawDBConnectError{fmt.Sprintf("github.com/ecletus/qor.site: Site %q: Failed to "+
				"close RAW connection of DB %q: %v", r.DB.Site.Name(), r.DB.Name, err)})
		}
	}
}

type DB struct {
	initCallbacks []func(DB *DB)
	Site          SiteInterface
	Config        *config.DBConfig
	Name          string
	DB            *aorm.DB
	Raw           *RawDB
	open          func() (DB *aorm.DB, err error)
}

func (db *DB) InitCallback(cb ...func(DB *DB)) {
	db.initCallbacks = append(db.initCallbacks, cb...)
	if db.DB != nil {
		for _, cb := range cb {
			cb(db)
		}
	}
}

func (db *DB) Open() (err error) {
	if db.DB != nil {
		return fmt.Errorf("DB %q for site %q is open", db.Name, db.Site.Name())
	}
	db.DB, err = db.open()
	if err != nil {
		return errwrap.Wrap(err, "Open DB %q for site %q failed", db.Name, db.Site.Name())
	}
	for _, cb := range db.initCallbacks {
		cb(db)
	}
	return
}

func (db *DB) Close() (err error) {
	if db.DB != nil {
		err = db.DB.Close()
		if err == nil {
			db.DB = nil
		} else {
			err = errwrap.Wrap(err, "Close DB %q for site %q failed", db.Name, db.Site.Name())
		}
	}
	return
}

func (db *DB) ReOpen() (err error) {
	if err = db.Close(); err == nil {
		err = db.Open()
	}
	return
}

func DefaultPrepareRawDBContext(db *DB) context.Context {
	return nil
}

var PrepareRawDBContext = DefaultPrepareRawDBContext

type SiteInterface interface {
	xroute.Handler
	Config() *config.SiteConfig
	AdvancedConfig() config.OtherConfig
	GetDB(name string) *DB
	GetSystemDB() *DB
	EachDB(f func(db *DB) (err error)) (err error)
	StorageNames() *oss.Names
	SystemStorage() *filesystem.FileSystem
	MediaStorages() map[string]oss.NamedStorageInterface
	GetMediaStorage(name string) oss.NamedStorageInterface
	GetMediaStorageOrDefault(name string) oss.NamedStorageInterface
	GetDefaultMediaStorage() oss.NamedStorageInterface
	PrepareContext(ctx *Context) *Context
	Name() string
	Init() error
	InitOrPanic() SiteInterface
	NewContext() *Context
	PublicURL(path ...string) string
	PublicURLf(path ...interface{}) string
}

type Site struct {
	config         *config.SiteConfig
	dbs            map[string]*DB
	mediaStorages  map[string]oss.NamedStorageInterface
	systemStorage  *filesystem.FileSystem
	storageNames   *oss.Names
	Handler        xroute.ContextHandler
	contextFactory *ContextFactory
}

func NewSite(contextFactory *ContextFactory, config *config.SiteConfig) *Site {
	return &Site{
		contextFactory: contextFactory,
		config:         config,
		dbs:            make(map[string]*DB),
		mediaStorages:  make(map[string]oss.NamedStorageInterface),
		systemStorage:  filesystem.New(&filesystem.Config{RootDir: config.RootDir}),
		storageNames:   oss.NewNames(),
		Handler:        xroute.NewMux(),
	}
}

type SetupDB func(setup func(db *DB) error) (err error)

func (s *Site) Config() *config.SiteConfig {
	return s.config
}

func (s *Site) AdvancedConfig() config.OtherConfig {
	return s.config.OtherConfig
}

func (s *Site) Name() string {
	return s.config.Name
}

func (s *Site) Init() (err error) {
	s.mediaStorages["default"] = &oss.NamedStorage{s.systemStorage, "default"}

	for name, storageConfig := range s.config.MediaStorage {
		s.mediaStorages[name] = &oss.NamedStorage{storageConfig["@storage"].(oss.StorageInterface), name}
	}

	for name, dbConfig := range s.config.Db {
		func(dbConfig *config.DBConfig) {
			var d *DB
			d = &DB{
				Site:   s,
				Config: dbConfig,
				Name:   name,
				open: func() (DB *aorm.DB, err error) {
					if DB, err = db.SystemFactories.Factory(dbConfig); err == nil {
						DB = assigners.Assigners().ApplyToDB(DB)
						DB = DB.Inside(PREFIX+".site["+s.Name()+"]", "DB["+name+"]").Set(PREFIX+".site", s).Set(PREFIX+".db", d)
					}
					return
				},
			}
			if err := d.Open(); err != nil {
				panic(err)
			}
			d.Raw = &RawDB{DB: d}
			s.dbs[name] = d
		}(dbConfig)
	}

	return
}

func (site *Site) InitOrPanic() SiteInterface {
	err := site.Init()
	if err != nil {
		panic(fmt.Errorf("Init site %q failed: %v", site.Name(), err))
	}
	return site
}

func (site *Site) StorageNames() *oss.Names {
	return site.storageNames
}

func (site *Site) GetDB(name string) *DB {
	return site.dbs[name]
}

func (site *Site) GetSystemDB() *DB {
	return site.GetDB(DB_SYSTEM)
}

func (site *Site) EachDB(f func(db *DB) error) (err error) {
	for _, db := range site.dbs {
		if err = f(db); err != nil {
			if err == StopDBIteration {
				return nil
			}
			return
		}
	}
	return
}

func (site *Site) SystemStorage() *filesystem.FileSystem {
	return site.systemStorage
}

func (site *Site) MediaStorages() map[string]oss.NamedStorageInterface {
	return site.mediaStorages
}

func (site *Site) GetMediaStorage(name string) oss.NamedStorageInterface {
	return site.mediaStorages[name]
}

func (site *Site) GetMediaStorageOrDefault(name string) oss.NamedStorageInterface {
	s := site.GetMediaStorage(name)
	if s == nil {
		return site.GetDefaultMediaStorage()
	}
	return s
}

func (site *Site) GetDefaultMediaStorage() oss.NamedStorageInterface {
	return site.GetMediaStorage("default")
}

func (site *Site) PrepareContext(context *Context) *Context {
	context.AsTop()
	context.Site = site
	DB := site.GetSystemDB().DB
	if context.Request != nil {
		DB = DB.Inside("Req[" + context.Request.Method + " " + context.Request.RequestURI + "]")
	}
	context.SetDB(DB.Set(CONTEXT_KEY, context))
	if context.RouteContext == nil {
		context.RouteContext = xroute.NewRouteContext()
	}
	return context
}

func (site *Site) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if strings.HasPrefix(r.URL.Path, PATH_MEDIA) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, PATH_MEDIA)
		storage := site.GetDefaultMediaStorage()
		storage.ServeHTTP(w, r)
		return
	}
	prefix := httpu.PrefixR(r)
	r, context := site.contextFactory.NewContextFromRequestPair(w, r, prefix)
	context.StaticURL += "/static"
	site.PrepareContext(context)
	rctx.Data[CONTEXT_KEY] = context
	rctx.ChainRequestSetters[CONTEXT_KEY] = xroute.NewChainRequestSetter(func(chain *xroute.ChainHandler, r *http.Request) {
		chain.Context.Data[CONTEXT_KEY].(*Context).SetRequest(r)
	})
	site.Handler.ServeHTTPContext(w, r, rctx)
}

func (site *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	site.Handler.ServeHTTPContext(w, r, nil)
}

func (site *Site) NewContext() *Context {
	return site.PrepareContext(&Context{})
}

func (site *Site) PublicURL(p ...string) string {
	if len(p) > 0 {
		return strings.Join(append([]string{site.config.PublicURL}, p...), "/")
	}
	return site.config.PublicURL
}

func (site *Site) PublicURLf(p ...interface{}) string {
	parts := make([]string, len(p))
	for i, part := range p {
		if pt, ok := part.(string); ok {
			parts[i] = pt
		} else {
			parts[i] = fmt.Sprint(pt)
		}
	}
	return site.PublicURL(parts...)
}

func GetSiteFromDB(db *aorm.DB) SiteInterface {
	s, _ := db.Get(PREFIX + ".site")
	return s.(SiteInterface)
}

func GetSiteFromRequest(r *http.Request) SiteInterface {
	return ContextFromRequest(r).Site
}

func GetDBFromDB(db *aorm.DB) *DB {
	s, _ := db.Get(PREFIX + ".db")
	return s.(*DB)
}
