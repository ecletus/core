package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ecletus/oss"
	"github.com/ecletus/oss/factories"
	"github.com/moisespsena-go/stringvar"
	"github.com/moisespsena/go-assetfs"
	"github.com/moisespsena-go/error-wrap"
)

type OtherConfig map[string]interface{}

func NewOtherConfig(data ...map[string]interface{}) OtherConfig {
	oc := make(OtherConfig)
	for _, d := range data {
		for k, v := range d {
			oc.Set(k, v)
		}
	}
	return oc
}

func (oc OtherConfig) Set(key string, value interface{}) OtherConfig {
	if key == "" {
		panic(errors.New("OtherConfig.Set: Key is empty"))
	}
	parts := strings.Split(key, ".")
	end, parts := parts[len(parts)-1], parts[:len(parts)-1]
	o := oc
	for _, p := range parts {
		parent, ok := (o)[p]
		if !ok {
			parent = make(OtherConfig)
			(o)[p] = parent
		}
		o = parent.(OtherConfig)
	}
	(o)[end] = value
	return oc
}

func (oc OtherConfig) Merge(key string, values ...map[string]interface{}) OtherConfig {
	if key == "" {
		panic(errors.New("OtherConfig.Merge: Key is empty"))
	}
	for _, value := range values {
		for k, v := range value {
			if vmap, ok := v.(map[string]interface{}); ok {
				oc.GetMany(k, true).Merge(k, vmap)
			} else {
				(oc)[k] = v
			}
		}
	}
	return oc
}

func (oc OtherConfig) SetMany(key string, values ...map[string]interface{}) OtherConfig {
	oc.Set(key, nil)
	oc.GetMany(key, true).Merge(key, values...)
	return oc
}

func (oc OtherConfig) GetMany(key string, createIfNil ...bool) (value OtherConfig) {
	v, _ := oc.Get(key)

	if v == nil && len(createIfNil) == 1 && createIfNil[0] {
		value = make(OtherConfig)
		oc.Set(key, value)
		return value
	}

	return v.(OtherConfig)
}

func (oc OtherConfig) Get(key string) (value interface{}, ok bool) {
	if key == "" {
		panic(errors.New("OtherConfig.Merge: Key is empty"))
	}
	parts := strings.Split(key, ".")
	end, parts := parts[len(parts)-1], parts[:len(parts)-1]
	o := oc
	for _, p := range parts {
		parent, ok := (o)[p]
		if !ok {
			return nil, false
		}
		o = parent.(OtherConfig)
	}
	value, ok = (o)[end]
	return
}

func (oc OtherConfig) GetBool(key string, defaul ...bool) bool {
	v, _ := oc.Get(key)
	if v != nil {
		return v.(bool)
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return false
}

func (oc OtherConfig) GetString(key string, defaul ...string) string {
	v, _ := oc.Get(key)
	if v != nil {
		return v.(string)
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return ""
}

func (oc OtherConfig) GetInt(key string, defaul ...int) int {
	v, _ := oc.Get(key)
	if v != nil {
		return v.(int)
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return 0
}

func (oc OtherConfig) GetSlice(key string, defaul ...[]interface{}) (r []interface{}) {
	v, _ := oc.Get(key)
	if v != nil {
		return v.([]interface{})
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return
}

func (oc OtherConfig) GetStrings(key string, defaul ...[]string) (r []string) {
	v, _ := oc.Get(key)
	if v != nil {
		return v.([]string)
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return
}

func (oc OtherConfig) GetInterface(key string, defaul ...interface{}) interface{} {
	if v, ok := oc.Get(key); ok {
		return v
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return nil
}

func (oc OtherConfig) GetAssetFS(key string, defaul ...interface{}) assetfs.Interface {
	return oc.GetInterface(key, defaul...).(assetfs.Interface)
}

func (oc OtherConfig) On(key string, f func(ok bool, value interface{}) interface{}) interface{} {
	v, ok := oc.Get(key)
	return f(ok, v)
}

type SiteConfig struct {
	Name         string
	Title        string
	Domains      []string
	Db           map[string]*DBConfig
	MediaStorage map[string]map[string]interface{}
	RootDir      string
	SMTP         *SMTPConfig
	OtherConfig  OtherConfig
	PublicURL    string
}

func (s *SiteConfig) Prepare(siteName string, args *stringvar.StringVar) (err error) {
	if s.RootDir == "" {
		s.RootDir = "{ROOT}/system/sites/{SITE_NAME}"
	}

	args = args.Child("SITE_NAME", siteName)

	args.FormatPathPtr(&s.RootDir)
	args = args.Child("SITE_ROOT", s.RootDir)
	var storage oss.StorageInterface

	for mediaName, media := range s.MediaStorage {
		if storage, err = s.PrepareMediaStorage(siteName, mediaName, args); err != nil {
			return errwrap.Wrap(err, "Prepare Media Storage")
		}
		media["@storage"] = storage
	}

	for dbName, db := range s.Db {
		db.Prepare(siteName, dbName, args)
	}
	return nil
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type DBConfig struct {
	Name     string
	Adapter  string
	Host     string
	Port     int
	User     string
	Password string
	SSL      string
}

func (db *DBConfig) DSN() string {
	switch db.Adapter {
	case "mysql":
		return fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=True&loc=Local",
			db.User, db.Password, db.Host, db.Port, db.Name)
	case "postgres":
		ssl := db.SSL
		if ssl == "" {
			ssl = "disable"
		}
		return fmt.Sprintf("postgres://%v:%v@%v/%v?sslmode=%v",
			db.User, db.Password, db.Host, db.Name, ssl)
	case "sqlite", "sqlite3":
		return db.Name
	}
	return ""
}

func (s *SiteConfig) PrepareMediaStorage(siteName, mediaName string, vrs *stringvar.StringVar) (storage oss.StorageInterface, err error) {
	vrs = vrs.Child("MEDIA_NAME", mediaName)
	vrs.FormatPathPtr(&s.RootDir)
	fctx := factories.NewContext()
	fctx.Var = vrs
	cfg := s.MediaStorage[mediaName]
	typName := cfg["type"].(string)
	factory, ok := factories.Get(typName)
	if !ok {
		return nil, fmt.Errorf("Storage Factory %q does not exists", typName)
	}
	storage, err = factory.Factory(fctx, cfg)
	if err != nil {
		return nil, errwrap.Wrap(err, "Factory %q", typName)
	}
	return
}

func (d *DBConfig) Prepare(siteName, dbName string, args *stringvar.StringVar) {
	vrs := args.Child("DB_NAME", dbName)
	vrs.FormatPathPtr(&d.Name).
		FormatPtr(&d.Password, &d.User, &d.Host)
}
