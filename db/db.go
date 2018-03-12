package db

import (
	"fmt"
	"os"
	"errors"
	"github.com/jinzhu/gorm"
	"sync"
	"reflect"
	qorconfig "github.com/qor/qor/config"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var FakeDB = &gorm.DB{}

type Factory func(config *qorconfig.DBConfig) (db *gorm.DB, err error)
type Factories map[string]Factory

func (f Factories) Register(adapterName string, factory Factory) {
	f[adapterName] = factory
}

func (f Factories) Factory(config *qorconfig.DBConfig) (db *gorm.DB, err error) {
	if fc, ok := f[config.Adapter]; ok {
		db, err = fc(config)
		if err != nil {
			return nil, err
		}
		if os.Getenv("DEBUG") != "" {
			db.LogMode(true)
		}
		return
	} else {
		return nil, errors.New("not supported database adapter: " + config.Adapter)
	}
}

var SystemFactories = Factories{
	"mysql": func(config *qorconfig.DBConfig) (db *gorm.DB, err error) {
		return gorm.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=True&loc=Local",
			config.User, config.Password, config.Host, config.Port, config.Name))
	},
	"postgres": func(config *qorconfig.DBConfig) (db *gorm.DB, err error) {
		return gorm.Open("postgres", fmt.Sprintf("postgres://%v:%v@%v/%v?sslmode=disable",
			config.User, config.Password, config.Host, config.Name))
	},
	"sqlite": func(config *qorconfig.DBConfig) (db *gorm.DB, err error) {
		return gorm.Open("sqlite3", fmt.Sprintf("%v/%v", os.TempDir(), config.Name))
	},
}


type FieldCacher struct {
	data sync.Map
}

func (fc *FieldCacher) Get(model interface{}, fieldName string) *gorm.Field {
	typ := reflect.Indirect(reflect.ValueOf(model)).Type()
	mi, ok := fc.data.Load(typ)
	if !ok {
		mi = &sync.Map{}
		fc.data.Store(typ, mi)
	}
	m := mi.(*sync.Map)
	fi, ok := m.Load(fieldName)
	if !ok {
		fi, ok = FakeDB.NewScope(model).FieldByName(fieldName)
		if !ok {
			panic(fmt.Errorf("Invalid field %v.%v.%v", typ.PkgPath(), typ.Name(), fieldName))
		}
		m.Store(fieldName, fi)
	}
	return fi.(*gorm.Field)
}

var FieldCache = &FieldCacher{}
