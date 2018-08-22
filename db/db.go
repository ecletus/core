package db

import (
	"fmt"
	"os"
	"errors"
	"github.com/moisespsena-go/aorm"
	"sync"
	"reflect"
	"context"
	"os/exec"
	"time"
	"io"
	"bufio"
	qorconfig "github.com/aghape/core/config"
	_ "github.com/moisespsena-go/aorm/dialects/mysql"
	_ "github.com/moisespsena-go/aorm/dialects/postgres"
	_ "github.com/moisespsena-go/aorm/dialects/sqlite"
)

var FakeDB = &aorm.DB{}

type RawDBConnection interface {
	io.Closer
	Error() *bufio.Reader
	Out() *bufio.Reader
	In() io.Writer
}

type CmdDBConnection struct {
	Cmd *exec.Cmd
	in  io.Writer
	err *bufio.Reader
	out *bufio.Reader
}

func NewCmdDBConnection(cmd *exec.Cmd) *CmdDBConnection {
	in, _ := cmd.StdinPipe()
	o, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	return &CmdDBConnection{cmd, in, nil, bufio.NewReader(o)}
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
	if c.Cmd.ProcessState == nil {
		_, err := c.In().Write([]byte("\\q\\n"))
		if err != nil {
			return err
		} else {
			<-time.After(time.Second)
			return c.Cmd.Process.Kill()
		}
	}
	if c.Cmd.ProcessState.Exited() {
		if c.Cmd.ProcessState.Success() {
			return nil
		}
		return errors.New(c.Cmd.ProcessState.String())
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
	"mysql": func(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
		return aorm.Open("mysql", config.DSN())
	},
	"postgres": func(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
		return aorm.Open("postgres", config.DSN())
	},
	"sqlite": func(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
		return aorm.Open("sqlite3", config.DSN())
	},
}

var SystemRawFactories = RawFactories{
	"postgres": func(ctx context.Context, config *qorconfig.DBConfig) (db RawDBConnection, err error) {
		var cmd *exec.Cmd
		if ctx == nil {
			cmd = exec.Command("psql")
		} else {
			cmd = exec.CommandContext(ctx, "psql")
		}
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("PGUSER=%v", config.User),
			fmt.Sprintf("PGPASS=%v", config.Password),
			fmt.Sprintf("PGDATABASE=%v", config.Name))
		if config.Host != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("PGHOST=%v", config.Host))
		}
		if config.Port != 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("PGPORT=%v", config.Port))
		}

		con := NewCmdDBConnection(cmd)
		err = cmd.Start()
		if err != nil {
			return nil, err
		}
		return con, nil
	},
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
