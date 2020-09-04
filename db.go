package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	path_helpers "github.com/moisespsena-go/path-helpers"

	"github.com/ecletus/core/db/dbconfig"

	"github.com/ecletus/core/db"
	"github.com/moisespsena-go/aorm"
	errwrap "github.com/moisespsena-go/error-wrap"
)

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
	Site          *Site
	Config        *dbconfig.DBConfig
	Name          string
	DB            *aorm.DB
	//Raw           *RawDB
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
	if strings.HasPrefix(db.Config.Adapter, "sqlite") {
		if !path_helpers.IsExistingDir(filepath.Dir(db.Config.Name)) {
			if err := os.MkdirAll(filepath.Dir(db.Config.Name), path_helpers.MustResolveMode(filepath.Dir(db.Config.Name))); err != nil {
				return err
			}
		}
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
