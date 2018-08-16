package config

import (
	"fmt"
	"errors"
	"strings"
	"github.com/aghape/aghape/utils/str"
	"github.com/aghape/oss/ftp"
	"github.com/moisespsena/go-assetfs"
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

func (oc OtherConfig) GetBool(key string, defaul ... bool) bool {
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

func (oc OtherConfig) GetInt(key string, defaul ... int) int {
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
	MediaStorage map[string]*MediaStorageConfig
	RootDir      string
	SMTP         *SMTPConfig
	OtherConfig  OtherConfig
	PublicURL    string
}

func (s *SiteConfig) Prepare(siteName string, args *Args) {
	if s.RootDir == "" {
		s.RootDir = "{ROOT}/system/sites/{SITE_NAME}"
	}

	args = args.Merge(map[string]string{
		"SITE_NAME": siteName,
	})

	args.Vars().FormatPathPtr(&s.RootDir)
	args = args.Merge(map[string]string{
		"SITE_ROOT": s.RootDir,
	})

	for mediaName, media := range s.MediaStorage {
		media.Prepare(siteName, mediaName, args)
	}

	for dbName, db := range s.Db {
		db.Prepare(siteName, dbName, args)
	}
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
	case "sqlite":
		return db.Name
	}
	return ""
}

type MediaStorageConfig struct {
	RootDir string
	Ftp     *ftp.Config
}

type Args struct {
	Home string
	Root string
	data map[string]string
}

func NewArgs(home, root string) *Args {
	if root == "" {
		root = "./public"
	}
	return &Args{home, root, nil}
}

func (a *Args) Merge(merges ...map[string]string) *Args {
	r := &Args{a.Home, a.Root, make(map[string]string)}
	for k, v := range a.data {
		r.data[k] = v
	}

	for _, m := range merges {
		for k, v := range m {
			r.data[k] = v
		}
	}
	return r
}

func (v *Args) Vars(merges ...map[string]string) (vrs *str.Vars) {
	vrs = &str.Vars{Data: map[string]string{
		"HOME": v.Home,
		"ROOT": v.Root,
	}}

	return vrs.Merge(v.data).Merge(merges...)
}

func (m *MediaStorageConfig) Prepare(siteName, mediaName string, args *Args) {
	if m.RootDir == "" {
		m.RootDir = "{SITE_ROOT}/media"
	}

	vrs := args.Vars(map[string]string{
		"MEDIA_NAME": mediaName,
	})

	vrs.FormatPathPtr(&m.RootDir)

	if m.Ftp != nil {
		vrs.FormatPathPtr(&m.Ftp.RootDir).
			FormatPtr(&m.Ftp.Endpoint, &m.Ftp.User, &m.Ftp.Password)

		for i, host := range m.Ftp.Hosts {
			vrs.FormatPtr(&host)
			m.Ftp.Hosts[i] = host
		}
	}
}

func (d *DBConfig) Prepare(siteName, dbName string, args *Args) {
	vrs := args.Vars(map[string]string{
		"DB_NAME": dbName,
	})
	vrs.FormatPathPtr(&d.Name).
		FormatPtr(&d.Password, &d.User, &d.Host)
}
