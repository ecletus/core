package site_config

import (
	"fmt"

	"github.com/ecletus/core/db/dbconfig"

	"github.com/moisespsena-go/maps"

	"github.com/ecletus/oss"
	"github.com/ecletus/oss/factories"
	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/stringvar"
)

type Config struct {
	Title        string
	Db           map[string]*dbconfig.DBConfig
	MediaStorage map[string]map[string]interface{} `mapstructure:"media_storage"`
	RootDir      string                            `mapstructure:"root_dir"`
	PublicURL    string                            `mapstructure:"public_url"`
	StaticURL    string                            `mapstructure:"static_url"`
	Locale       string                            `mapstructure:"locale"`
	TimeLocation string                            `mapstructure:"time_location"`
	Raw          maps.MapSI
}

func (this *Config) Prepare(mainDBConfig map[string]*dbconfig.DBConfig, siteName string, args *stringvar.StringVar) (err error) {
	if this.Db == nil {
		this.Db = map[string]*dbconfig.DBConfig{}
	}
	if _, ok := this.Db[dbconfig.DB_SYSTEM]; !ok {
		this.Db[dbconfig.DB_SYSTEM] = &dbconfig.DBConfig{}
	}
	if mainDBConfig != nil {
		for dbName, defaultCfg := range mainDBConfig {
			if db := this.Db[dbName]; db != nil {
				if db.Adapter == "" {
					db.Adapter = defaultCfg.Adapter
				}
			}
		}
	}

	if _, ok := this.Db[dbconfig.DB_SYSTEM]; !ok {
		this.Db[dbconfig.DB_SYSTEM] = &dbconfig.DBConfig{}
	}

	if this.RootDir == "" {
		this.RootDir = "{{.DATA_DIR}}/sites/{{.SITE_NAME}}"
	}

	args = args.Child("SITE_NAME", siteName)
	args.FormatPathPtr(&this.RootDir)
	args.FormatPtr(&this.PublicURL, &this.StaticURL)
	args = args.Child("SITE_ROOT", this.RootDir)
	var storage oss.StorageInterface

	for mediaName, media := range this.MediaStorage {
		if storage, err = this.PrepareMediaStorage(siteName, mediaName, args); err != nil {
			return errwrap.Wrap(err, "Prepare Media Storage")
		}
		media["@storage"] = storage
	}

	for dbName, db := range this.Db {
		db.Prepare(siteName, dbName, args)
	}
	return nil
}

func (this *Config) PrepareMediaStorage(siteName, mediaName string, vrs *stringvar.StringVar) (storage oss.StorageInterface, err error) {
	vrs = vrs.Child("MEDIA_NAME", mediaName)
	vrs.FormatPathPtr(&this.RootDir)
	fctx := factories.NewContext()
	fctx.Var = vrs
	cfg := this.MediaStorage[mediaName]
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
