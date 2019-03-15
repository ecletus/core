package db

import (
	qorconfig "github.com/ecletus/core/config"
	"github.com/moisespsena-go/aorm"
)

func MySQLfacotry(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
	return aorm.Open("mysql", config.DSN())
}

func PostgresFactory(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
	return aorm.Open("postgres", config.DSN())
}

func Sqlite3Factory(config *qorconfig.DBConfig) (db *aorm.DB, err error) {
	return aorm.Open("sqlite3", config.DSN())
}
