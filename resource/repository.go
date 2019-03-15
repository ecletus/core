package resource

import (
	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena/go-edis"
)

type Repository interface {
	FindOneLayout(key string, layout ...interface{}) (result interface{}, err error)
	FindManyLayout(layout ...interface{}) (result interface{}, err error)
	FindManyLayoutOrDefault(layout interface{}, defaul ...interface{}) (interface{}, error)
	FindManyBasic() (result []BasicValue, err error)
	FindOneBasic(key string) (result BasicValue, err error)
	FindOne(result interface{}, key ...string) (err error)
	FindMany(result interface{}) (err error)
	Create(record interface{}) error
	Update(record interface{}) error
	SaveOrCreate(recorde interface{}) error
	Delete(record interface{}) (err error)
}

type RepositoryClient interface {
	Repository
	edis.EventDispatcherInterface
	SetDB(DB *aorm.DB) RepositoryClient
	DB() *aorm.DB
	Resource() Resourcer
	Layout() LayoutInterface
	Parent() RepositoryClient
	MetaValues() *MetaValues
	SetMetaValues(metaValues *MetaValues) RepositoryClient
	Dispatchers() []edis.EventDispatcherInterface
	AppendDispatcher(dis ...edis.EventDispatcherInterface) RepositoryClient
	SetLayout(layout interface{}) RepositoryClient
	SetLayoutOrDefault(layout interface{}, defaul ...interface{}) RepositoryClient
	SetContext(ctx *core.Context) RepositoryClient
}
