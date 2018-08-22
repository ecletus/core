package core

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena/go-route"
	"github.com/aghape/oss"
	"github.com/aghape/oss/filesystem"
	"github.com/aghape/oss/ftp"
	"github.com/aghape/core/config"
	"github.com/aghape/core/db"
)

const DB_SYSTEM = "system"

type SitesReaderInterface interface {
	Get(siteName string) SiteInterface
	GetOrError(siteName string) (SiteInterface, error)
	All() []SiteInterface
	Sorted() []SiteInterface
	Names() []string
	EachOrAll(siteName string, cb func(site SiteInterface) (cont bool, err error)) error
	Each(cb func(site SiteInterface) (cont bool, err error)) error
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

func (r SitesReader) Each(cb func(site SiteInterface) (cont bool, err error)) error {
	for _, s := range r {
		cont, err := cb(s)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}
	return nil
}

func (r SitesReader) EachOrAll(siteName string, cb func(site SiteInterface) (cont bool, err error)) error {
	if siteName == "" || siteName == "*" {
		return r.Each(cb)
	}

	site, err := r.GetOrError(siteName)

	if err != nil {
		return err
	}

	_, err = cb(site)
	return err
}

type RawDB struct {
	DB   *DB
	conn db.RawDBConnection
	lock sync.Mutex
}

func (r *RawDB) Open(ctx context.Context) (conn db.RawDBConnection, err error) {
	return db.SystemRawFactories[r.DB.Config.Adapter](ctx, r.DB.Config)
}

type RawDBConnectError struct {
	message string
}

func (r *RawDBConnectError) Error() string {
	return r.message
}

func (r *RawDB) With(f func(con db.RawDBConnection)) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.conn == nil {
		var err error
		r.conn, err = r.Open(PrepareRawDBContext(r.DB))
		if err != nil {
			panic(&RawDBConnectError{fmt.Sprintf("github.com/aghape/qor.site: Site %q: Failed to "+
				"open RAW connection of DB %q: %v", r.DB.Site.Name(), r.DB.Name, err)})
		}
	}
	f(r.conn)
}

func (r *RawDB) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.conn != nil {
		err := r.conn.Close()
		if err != nil {
			panic(&RawDBConnectError{fmt.Sprintf("github.com/aghape/qor.site: Site %q: Failed to "+
				"close RAW connection of DB %q: %v", r.DB.Site.Name(), r.DB.Name, err)})
		}
	}
}

type DB struct {
	Site   SiteInterface
	Config *config.DBConfig
	Name   string
	DB     *aorm.DB
	Raw    *RawDB
}

func DefaultPrepareRawDBContext(db *DB) context.Context {
	return nil
}

var PrepareRawDBContext = DefaultPrepareRawDBContext

type SiteInterface interface {
	route.Handler
	Config() *config.SiteConfig
	AdvancedConfig() config.OtherConfig
	SetDB(name string, db *aorm.DB)
	GetDB(name string) *DB
	GetSystemDB() *DB
	EachDB(f func(db *DB) bool) bool
	SetupDB(setup func(db *DB) error) (err error)
	StorageNames() *oss.Names
	SystemStorage() *filesystem.FileSystem
	MediaStorages() map[string]oss.StorageInterface
	GetMediaStorage(name string) oss.StorageInterface
	GetMediaStorageOrDefault(name string) oss.StorageInterface
	GetDefaultMediaStorage() oss.StorageInterface
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
	mediaStorages  map[string]oss.StorageInterface
	systemStorage  *filesystem.FileSystem
	storageNames   *oss.Names
	Handler        route.ContextHandler
	contextFactory *ContextFactory
}

func NewSite(contextFactory *ContextFactory, config *config.SiteConfig) *Site {
	return &Site{
		contextFactory: contextFactory,
		config:         config,
		dbs:            make(map[string]*DB),
		mediaStorages:  make(map[string]oss.StorageInterface),
		systemStorage:  filesystem.New(config.RootDir),
		storageNames:   oss.NewNames(),
		Handler:        route.NewMux(),
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
	s.mediaStorages["default"] = s.systemStorage

	for name, storageConfig := range s.config.MediaStorage {
		if storageConfig.Ftp != nil {
			s.mediaStorages[name], err = ftp.New(*storageConfig.Ftp)
			if err != nil {
				return fmt.Errorf("Configure FTP media storage %q fail: %v", name, err)
			}
		} else {
			if storageConfig.RootDir == s.systemStorage.Base {
				s.mediaStorages[name] = s.systemStorage
			} else {
				s.mediaStorages[name] = filesystem.New(storageConfig.RootDir)
			}
		}
	}

	var db_ *aorm.DB

	for name, dbConfig := range s.config.Db {
		db_, err = db.SystemFactories.Factory(dbConfig)
		if err != nil {
			return fmt.Errorf("Init DB %q fail: %v", name, err)
		}
		d := &DB{s, dbConfig, name, db_.Inside(PREFIX+".site["+s.Name()+"]", "DB["+name+"]").Set(PREFIX+".site", s), nil}
		d.Raw = &RawDB{DB: d}
		s.dbs[name] = d
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

func (site *Site) SetDB(name string, db *aorm.DB) {
	site.dbs[name] = &DB{site, nil, name, db.Set(PREFIX+".site", site), nil}
}

func (site *Site) GetDB(name string) *DB {
	return site.dbs[name]
}

func (site *Site) GetSystemDB() *DB {
	return site.GetDB(DB_SYSTEM)
}

func (site *Site) EachDB(f func(db *DB) bool) bool {
	for _, db := range site.dbs {
		if !f(db) {
			return false
		}
	}
	return true
}

func (site *Site) SystemStorage() *filesystem.FileSystem {
	return site.systemStorage
}

func (site *Site) MediaStorages() map[string]oss.StorageInterface {
	return site.mediaStorages
}

func (site *Site) GetMediaStorage(name string) oss.StorageInterface {
	return site.mediaStorages[name]
}

func (site *Site) GetMediaStorageOrDefault(name string) oss.StorageInterface {
	s := site.GetMediaStorage(name)
	if s == nil {
		return site.GetDefaultMediaStorage()
	}
	return s
}

func (site *Site) GetDefaultMediaStorage() oss.StorageInterface {
	return site.GetMediaStorage("default")
}

func (site *Site) PrepareContext(context *Context) *Context {
	context.AsTop()
	context.Site = site
	DB := site.GetSystemDB().DB
	if context.Request != nil {
		DB = DB.Inside("Req["+context.Request.Method+" "+context.Request.RequestURI+"]")
	}
	context.SetDB(DB.Set(CONTEXT_KEY, context))
	if context.RouteContext == nil {
		context.RouteContext = route.NewRouteContext()
	}
	return context
}

func (site *Site) SetupDB(setup func(db *DB) error) (err error) {
	site.EachDB(func(db *DB) bool {
		err = setup(db)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func (site *Site) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext) {
	prefix, r := site.contextFactory.GetCleanSkipPrefixFromRequest(r)
	r, context := site.contextFactory.NewContextFromRequestPair(w, r, prefix)
	context.StaticURL += "/static"
	site.PrepareContext(context)
	rctx.Data[CONTEXT_KEY] = context
	rctx.ChainRequestSetters[CONTEXT_KEY] = route.NewChainRequestSetter(func(chain *route.ChainHandler, r *http.Request) {
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
