package db

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"time"

	qorconfig "github.com/aghape/core/config"
	"github.com/moisespsena-go/aorm"
	_ "github.com/moisespsena-go/aorm/dialects/mysql"
	_ "github.com/moisespsena-go/aorm/dialects/postgres"
	_ "github.com/moisespsena-go/aorm/dialects/sqlite"
)

var FakeDB = &aorm.DB{}

type RawDBConnection interface {
	io.Closer
	Open() error
	Error() *bufio.Reader
	Out() *bufio.Reader
	In() io.Writer
	Do(func(c RawDBConnection))
}

type CmdDBConnection struct {
	cmd       *exec.Cmd
	open      func() (*exec.Cmd, error)
	in        io.Writer
	err       *bufio.Reader
	out       *bufio.Reader
	closer    func(c *CmdDBConnection) error
	running   bool
}

func NewCmdDBConnection(cmd *exec.Cmd, closer func(c *CmdDBConnection) error) *CmdDBConnection {
	return &CmdDBConnection{cmd: cmd, closer: closer}
}

func (c *CmdDBConnection) Open() (err error) {
	c.in, _ = c.cmd.StdinPipe()
	o, _ := c.cmd.StdoutPipe()
	c.out = bufio.NewReader(o)
	c.cmd.Stderr = os.Stderr
	if err = c.cmd.Start(); err != nil {
		return err
	}
	return nil
}

func (c *CmdDBConnection) Do(f func(c RawDBConnection)) {
	f(c)
}

func (c *CmdDBConnection) Error() *bufio.Reader {
	return c.err
}
func (c *CmdDBConnection) Out() *bufio.Reader {
	return c.out
}

func (c *CmdDBConnection) In() io.Writer {
	return c.in
}

func (c *CmdDBConnection) Close() error {
	defer func() {
		c.running = false
	}()
	if c.cmd == nil || c.cmd.Process == nil {
		c.cmd = nil
		return nil
	}
	if c.cmd.ProcessState == nil {
		err := c.closer(c)
		if err != nil {
			return err
		} else {
			<-time.After(time.Second)
			if c.cmd.ProcessState == nil {
				return c.cmd.Process.Kill()
			}
		}
	}
	if c.cmd.ProcessState.Exited() {
		if c.cmd.ProcessState.Success() {
			return nil
		}
		return errors.New(c.cmd.ProcessState.String())
	}
	return nil
}

type Factory func(config *qorconfig.DBConfig) (db *aorm.DB, err error)
type RawFactory func(ctx context.Context, config *qorconfig.DBConfig) (db RawDBConnection, err error)
type Factories map[string]Factory
type RawFactories map[string]RawFactory

func (f Factories) Register(adapterName string, factory Factory) {
	f[adapterName] = factory
}

func (f Factories) Factory(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
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
	"mysql":    MySQLfacotry,
	"postgres": PostgresFactory,
	"sqlite":   Sqlite3Factory,
	"sqlite3":  Sqlite3Factory,
}

var SystemRawFactories = RawFactories{
	"postgres": PostgreSQLRawFactory,
	"sqlite":   Sqlite3RawFactory,
	"sqlite3":  Sqlite3RawFactory,
}

type FieldCacher struct {
	data sync.Map
}

func (fc *FieldCacher) Get(model interface{}, fieldName string) *aorm.Field {
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
	return fi.(*aorm.Field)
}

var FieldCache = &FieldCacher{}
