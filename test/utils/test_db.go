package utils

import (
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/moisespsena-go/aorm"
	_ "github.com/lib/pq"
)

// TestDB initialize a db for testing
func TestDB() *aorm.DB {
	var db *aorm.DB
	var err error
	var dbuser, dbpwd, dbname = "qor", "qor", "qor_test"

	if os.Getenv("DB_USER") != "" {
		dbuser = os.Getenv("DB_USER")
	}

	if os.Getenv("DB_PWD") != "" {
		dbpwd = os.Getenv("DB_PWD")
	}

	if os.Getenv("TEST_DB") == "postgres" {
		db, err = aorm.Open("postgres", fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=disable", dbuser, dbpwd, dbname))
	} else {
		// CREATE USER 'qor'@'localhost' IDENTIFIED BY 'qor';
		// CREATE DATABASE qor_test;
		// GRANT ALL ON qor_test.* TO 'qor'@'localhost';
		db, err = aorm.Open("mysql", fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=True&loc=Local", dbuser, dbpwd, dbname))
	}

	if err != nil {
		panic(err)
	}

	if os.Getenv("DEBUG") != "" {
		db.LogMode(true)
	}

	return db
}
