package qor

import (
	"github.com/moisespsena-go/aorm"
	_ "github.com/moisespsena-go/aorm/dialects/postgres"
)

var FakeDB = aorm.FakeDB("postgres")
