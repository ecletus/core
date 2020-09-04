package db

import (
	"context"
	"fmt"
	"github.com/ecletus/core/db/dbconfig"
	"os"
	"os/exec"
)

func PostgreSQLRawFactory(ctx context.Context, config *dbconfig.DBConfig) (db RawDBConnection, err error) {
	var cmd *exec.Cmd
	if ctx == nil {
		cmd = exec.Command("psql")
	} else {
		cmd = exec.CommandContext(ctx, "psql")
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("PGUSER=%v", config.User),
		fmt.Sprintf("PGPASSWORD=%v", config.Password),
		fmt.Sprintf("PGDATABASE=%v", config.Name))
	if config.Host != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGHOST=%v", config.Host))
	}
	if config.Port != 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPORT=%v", config.Port))
	}

	con := NewCmdDBConnection(cmd, func(c *CmdDBConnection) (err error) {
		_, err = c.In().Write([]byte("\\q\\n"))
		return
	})
	return con, nil
}

func Sqlite3RawFactory(ctx context.Context, config *dbconfig.DBConfig) (db RawDBConnection, err error) {
	var cmd *exec.Cmd
	if ctx == nil {
		cmd = exec.Command("sqlite3", config.Name)
	} else {
		cmd = exec.CommandContext(ctx, "sqlite3", config.Name)
	}

	con := NewCmdDBConnection(cmd, func(c *CmdDBConnection) (err error) {
		_, err = c.In().Write([]byte("\n.quit\n"))
		return
	})
	return con, nil
}
