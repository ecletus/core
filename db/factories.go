package db

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/elliotchance/sshtunnel"
	"golang.org/x/crypto/ssh"

	"github.com/ecletus/core/db/dbconfig"
	"github.com/moisespsena-go/aorm"
)

func MySQLfacotry(config *dbconfig.DBConfig) (db *aorm.DB, err error) {
	return aorm.Open("mysql", config.DSN())
}

func PostgresFactory(config *dbconfig.DBConfig) (db *aorm.DB, err error) {
	if config.SSHTunnel.Enabled {
		cfg := config.SSHTunnel
		if !cfg.InheritDisabled {
			if cfg.Host == "" {
				cfg.Host = config.Host
			}
			if cfg.User == "" {
				cfg.User = config.User
			}
			if cfg.Password == "" {
				cfg.Password = config.Password
			}
			if cfg.Name == "" {
				cfg.Name = config.Name
			}
			if cfg.Port == 0 {
				cfg.Port = config.Port
			}
		}
		host, port := tunnel(config.SSHTunnel)
		config.Port = port
		config.Host = host
		config.User = cfg.User
		config.Password = cfg.Password
		config.Name = cfg.Name
	}
	return aorm.Open("postgres", config.DSN())
}

func Sqlite3Factory(config *dbconfig.DBConfig) (db *aorm.DB, err error) {
	return aorm.Open("sqlite3", config.DSN())
}

func tunnel(config dbconfig.SSHTunnelConfig) (host string, port uint16) {
	var hostAddr = config.SSH.Host
	if config.SSH.Port != 0 {
		hostAddr += ":" + fmt.Sprint(config.SSH.Port)
	}
	if config.Host == "" {
		config.Host = "127.0.0.1"
	}

	var client = &ssh.ClientConfig{
		User: config.SSH.User,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Always accept key.
			return nil
		},
	}

	if config.SSH.User == "" {
		if u, _ := user.Current(); u != nil {
			config.SSH.User = u.Username
		}
	}

	if config.SSH.Password != "" {
		client.Auth = append(client.Auth, ssh.Password(config.SSH.Password))
	} else {
		client.Auth = append(client.Auth, keyFile(config.SSH.KeyFile))
	}

	// Setup the tunnel, but do not yet start it yet.
	tunnel := sshtunnel.NewSSHTunnel(
		// User and host of tunnel server, it will default to port 22
		// if not specified.
		config.SSH.User+"@"+hostAddr,

		ssh.Password(config.SSH.Password), // 2. password

		// The destination host and port of the actual server.
		fmt.Sprintf("127.0.0.1:%d", config.Port),

		// The local port you want to bind the remote port to.
		// Specifying "0" will lead to a random port.
		fmt.Sprint(config.LocalPort),
	)

	tunnel.Config = client
	tunnel.Log = log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds)

	// Start the server in the background. You will need to wait a
	// small amount of time for it to bind to the localhost port
	// before you can start sending connections.
	go tunnel.Start()
	time.Sleep(100 * time.Millisecond)

	// NewSSHTunnel will bind to a random port so that you can have
	// multiple SSH tunnels available. The port is available through:
	//   tunnel.Local.Port

	// You can use any normal Go code to connect to the destination server
	// through localhost. You may need to use 127.0.0.1 for some libraries.
	//
	// Here is an example of connecting to a PostgreSQL server:
	return "127.0.0.1", uint16(tunnel.Local.Port)
}

func keyFile(file string) ssh.AuthMethod {
	if file == "" {
		usr, _ := user.Current()
		file = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
	}
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}
