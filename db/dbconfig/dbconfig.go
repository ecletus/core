package dbconfig

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/moisespsena-go/stringvar"
)

type SSHConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	KeyFile  string `mapstructure:"key_file"`
}

type SSHTunnelConfig struct {
	Enabled                    bool
	InheritDisabled            bool `mapstructure:"inherit_disabled" yaml:"inherit_disabled"`
	Name, Host, User, Password string
	Port                       uint16
	LocalPort                  uint16    `mapstructure:"local_port" yaml:"local_port"`
	SSH                        SSHConfig `mapstructure:"ssh" yaml:"ssh"`
}

type DBConfig struct {
	Name           string          `mapstructure:"name"`
	Adapter        string          `mapstructure:"adapter"`
	Host           string          `mapstructure:"host"`
	Port           uint16          `mapstructure:"port"`
	User           string          `mapstructure:"user"`
	Password       string          `mapstructure:"password"`
	SSL            string          `mapstructure:"ssl"`
	Args           url.Values      `mapstructure:"args"`
	SSHTunnel      SSHTunnelConfig `mapstructure:"ssh_tunnel"`
	CommitDisabled bool            `mapstructure:"commit_disabled"`
}

func (this DBConfig) DSN() string {
	switch this.Adapter {
	case "mysql":
		var args = make(url.Values)
		args.Set("parseTime", "true")
		for k, v := range this.Args {
			args.Set(k, v[0])
		}
		return fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?"+args.Encode(),
			this.User, this.Password, this.Host, this.Port, this.Name)
	case "postgres":
		var args = make(url.Values)
		for k, v := range this.Args {
			args.Set(k, v[0])
		}
		if this.SSL == "" {
			args.Set("sslmode", "disable")
		}
		args.Set("binary_parameters", "yes")
		var hostPort = this.Host
		if this.Port != 0 {
			hostPort += fmt.Sprintf(":%d", this.Port)
		}
		dsn := fmt.Sprintf("postgres://%v:%v@%v/%v",
			this.User, this.Password, hostPort, this.Name)
		if len(args) > 0 {
			return dsn + "?" + args.Encode()
		}
		return dsn
	case "sqlite", "sqlite3":
		if len(this.Args) > 0 {
			return this.Name + "?" + this.Args.Encode()
		}
		return this.Name
	}
	return ""
}

func (this *DBConfig) Prepare(siteName, dbName string, args *stringvar.StringVar) {
	if strings.HasPrefix(this.Adapter, "sqlite") && this.Name == "" {
		this.Name = "{{.SITE_ROOT}}/db/{{.DB_NAME}}.db"
	}
	vrs := args.Child("DB_NAME", dbName)
	vrs.FormatPathPtr(&this.Name).
		FormatPtr(&this.Password, &this.User, &this.Host)
	if this.Args == nil {
		this.Args = url.Values{}
	}

	switch this.Adapter {
	case "postgres":
		if this.Args.Get("application_name") == "" {
			this.Args.Set("application_name", filepath.Base(os.Args[0])+"@"+siteName)
		}
	}
}

const DB_SYSTEM = "system"

func Args(values url.Values, append bool, del ...string) string {
	for _, name := range del {
		values.Del(name)
	}
	v := values.Encode()
	if append && v != "" {
		return "&" + v
	}
	return ""
}
