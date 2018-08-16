package qor

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var FakeDB = gorm.FakeDB("postgres")
