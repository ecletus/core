package resource

import (
	"github.com/moisespsena-go/edis"

	"github.com/ecletus/core"
	"github.com/go-aorm/aorm"
)

// Resourcer interface
type Resourcer interface {
	edis.EventDispatcherInterface
	Struct
	core.Permissioner
	GetID() string
	FullID() string
	GetResource() *Resource
	GetPrimaryFields() []*aorm.StructField
	GetMetas([]string) []Metaor
	GetContextMetas(*core.Context) []Metaor
	NewStruct(site ...*core.Site) interface{}
	GetPathLevel() int
	SetParent(parent Resourcer, rel *ParentRelationship)
	GetParentResource() Resourcer
	GetParentRelation() *ParentRelationship
	IsSingleton() bool
	ToParam() string
	ParamIDPattern() string
	ParamIDName() string
	BasicValue(ctx *core.Context, recorde interface{}) BasicValuer
	BasicDescriptableValue(ctx *core.Context, record interface{}) BasicDescriptableValuer
	Crud(ctx *core.Context) *CRUD
	CrudDB(db *aorm.DB) *CRUD
	Layout(name string, layout LayoutInterface)
	GetLayoutOrDefault(name string) LayoutInterface
	GetLayout(name string, defaul ...string) LayoutInterface
	HasKey() bool
	ContextSetup(ctx *core.Context) *core.Context
	GetKey(value interface{}) aorm.ID
	ParseID(s string) (ID aorm.ID, err error)
	SetID(record interface{}, id aorm.ID)
	PrimaryValues(id aorm.ID) (args []interface{})
	GetModelStruct() *aorm.ModelStruct
	DefaultDenyMode() bool
	DefaultPrimaryKeyOrder() aorm.Order
	SetDefaultPrimaryKeyOrder(val aorm.Order)
}
