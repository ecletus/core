package qor

import (
	"fmt"
	"strings"
	"net/http"
	"github.com/qor/oss"
	"github.com/qor/oss/filesystem"
	"github.com/qor/oss/ftp"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/qor/qor/db"
	"github.com/qor/qor/config"
)

type SitesReaderInterface interface {
	Get(siteName string) SiteInterface
	GetOrError(siteName string) (SiteInterface, error)
	All() []SiteInterface
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

type DB struct {
	Site SiteInterface
	Name string
	DB   *gorm.DB
}

type SiteInterface interface {
	http.Handler
	Config() *config.SiteConfig
	SetDB(name string, db *gorm.DB)
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
	Name()string
	Init() error
	InitOrPanic() SiteInterface
	NewContext() *Context
	PublicURL(path... string) string
	PublicURLf(path... interface{}) string
}

type Site struct {
	config        *config.SiteConfig
	dbs           map[string]*DB
	mediaStorages map[string]oss.StorageInterface
	systemStorage *filesystem.FileSystem
	storageNames  *oss.Names
	Handler        http.Handler
}

func NewSite(config *config.SiteConfig) *Site {
	return &Site{
		config:        config,
		dbs:           make(map[string]*DB),
		mediaStorages: make(map[string]oss.StorageInterface),
		systemStorage: filesystem.New(config.RootDir),
		storageNames:  oss.NewNames(),
		Handler: chi.NewRouter(),
	}
}

type SetupDB func(setup func(db *DB) error) (err error)

func (s *Site) Config() *config.SiteConfig {
	return s.config
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

	var db_ *gorm.DB

	for name, dbConfig := range s.config.Db {
		db_, err = db.SystemFactories.Factory(dbConfig)
		if err != nil {
			return fmt.Errorf("Init DB %q fail: %v", name, err)
		}
		s.dbs[name] = &DB{s, name, db_.Set("qor:site", s)}
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

func (site *Site) SetDB(name string, db *gorm.DB) {
	site.dbs[name] = &DB{site, name, db.Set("qor:site", site)}
}

func (site *Site) GetDB(name string) *DB {
	return site.dbs[name]
}

func (site *Site) GetSystemDB() *DB {
	return site.GetDB("system")
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
	context.Site = site
	context.DB = site.GetSystemDB().DB.Set(CONTEXT_KEY, context)
	context.Top()
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

func (site *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	site.Handler.ServeHTTP(w, r)
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

func GetSiteFromDB(db *gorm.DB) SiteInterface {
	s, _ := db.Get("qor:site")
	return s.(SiteInterface)
}

func GetSiteFromRequest(r *http.Request) SiteInterface {
	return ContextFromRequest(r).Site
}


