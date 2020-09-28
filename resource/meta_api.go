package resource

import (
	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

const (
	SiblingsRequirementCheckDisabledOnTrue SiblingsRequirementCheckDisabled = iota + 1
	SiblingsRequirementCheckDisabledOnFalse
)

type SiblingsRequirementCheckDisabled uint8

func (this SiblingsRequirementCheckDisabled) OnTrue() bool {
	return this == SiblingsRequirementCheckDisabledOnTrue
}

func (this SiblingsRequirementCheckDisabled) OnFalse() bool {
	return this == SiblingsRequirementCheckDisabledOnFalse
}

type MetaScanner interface {
	MetaScan(value interface{}) error
}

// Metaor interface
type Metaor interface {
	core.Permissioner
	GetName() string
	GetFieldName() string
	GetFieldStruct() *aorm.StructField
	GetSetter() func(resource interface{}, metaValue *MetaValue, context *core.Context) error
	GetFormattedValuer() func(recorde interface{}, context *core.Context) interface{}
	GetValuer() func(recorde interface{}, context *core.Context) interface{}
	GetContextResourcer() func(meta Metaor, context *core.Context) Resourcer
	GetResource() Resourcer
	GetBaseResource() Resourcer
	GetMetas() []Metaor
	GetContextMetas(recorde interface{}, context *core.Context) []Metaor
	GetContextResource(context *core.Context) Resourcer
	IsInline() bool
	IsRequired() bool
	IsZero(recorde, value interface{}) bool
	GetLabelC(ctx *core.Context) string
	Validators() []func(record interface{}, values *MetaValue, ctx *core.Context) (err error)
	GetRecordLabelC(ctx *core.Context, record interface{}) string
	Proxier() bool
	IsAlone() bool
	IsSiblingsRequirementCheckDisabled() SiblingsRequirementCheckDisabled
	Record(record interface{}) interface{}
}

// ConfigureMetaBeforeInitializeInterface if a struct's field's type implemented this interface, it will be called when initializing a meta
type ConfigureMetaBeforeInitializeInterface interface {
	ConfigureQorMetaBeforeInitialize(Metaor)
}

// ConfigureMetaInterface if a struct's field's type implemented this interface, it will be called after configed
type ConfigureMetaInterface interface {
	ConfigureQorMeta(Metaor)
}

// MetaConfigInterface meta configuration interface
type MetaConfigInterface interface {
	ConfigureMetaInterface
}
