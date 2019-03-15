package resource

import (
	"reflect"
	"strings"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
	"github.com/ecletus/roles"
	"github.com/moisespsena-go/aorm"
)

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
	GetMetas() []Metaor
	GetContextMetas(recorde interface{}, context *core.Context) []Metaor
	GetContextResource(context *core.Context) Resourcer
	IsInline() bool
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

// MetaConfig base meta config struct
type MetaConfig struct {
}

// ConfigureQorMeta implement the MetaConfigInterface
func (MetaConfig) ConfigureQorMeta(Metaor) {
}

type MetaName struct {
	Name        string
	EncodedName string
}

// GetName get meta's name
func (meta *MetaName) GetName() string {
	return meta.Name
}

// GetEncodedName get meta's encodedName
func (meta *MetaName) GetEncodedName() string {
	return meta.EncodedName
}

// GetEncodedName get meta's encodedName
func (meta *MetaName) GetEncodedNameOrDefault() string {
	if meta.EncodedName != "" {
		return meta.EncodedName
	}
	return meta.Name
}

// Meta meta struct definition
type Meta struct {
	*MetaName
	Alias            *MetaName
	FieldName        string
	FieldStruct      *aorm.StructField
	ContextResourcer func(meta Metaor, context *core.Context) Resourcer
	Setter           func(resource interface{}, metaValue *MetaValue, context *core.Context) error
	Valuer           func(interface{}, *core.Context) interface{}
	FormattedValuer  func(interface{}, *core.Context) interface{}
	Config           MetaConfigInterface
	BaseResource     Resourcer
	Resource         Resourcer
	Permission       *roles.Permission
	Help             string
	HelpLong         string
	SaveID           bool
	Inline           bool
}

func (meta *Meta) Namer() *MetaName {
	if meta.Alias != nil {
		return meta.Alias
	}
	return meta.MetaName
}

func (meta *Meta) IsInline() bool {
	return meta.Inline
}

// GetBaseResource get base resource from meta
func (meta *Meta) GetBaseResource() Resourcer {
	return meta.BaseResource
}

// GetFieldStruct get aorm field struct
func (meta *Meta) GetFieldStruct() *aorm.StructField {
	return meta.FieldStruct
}

// GetContextResource get resource from meta
func (meta *Meta) GetContextResourcer() func(meta Metaor, context *core.Context) Resourcer {
	return meta.ContextResourcer
}

func (meta *Meta) GetContextResource(context *core.Context) Resourcer {
	if meta.ContextResourcer != nil {
		return meta.ContextResourcer(meta, context)
	}
	return meta.Resource
}

func (meta *Meta) GetContextMetas(recort interface{}, context *core.Context) (metas []Metaor) {
	return meta.GetContextResource(context).GetMetas([]string{})
}

func (meta *Meta) GetMetas() (metas []Metaor) {
	return
}

func (meta *Meta) GetResource() Resourcer {
	return meta.Resource
}

// GetFieldName get meta's field name
func (meta *Meta) GetFieldName() string {
	return meta.FieldName
}

// SetFieldName set meta's field name
func (meta *Meta) SetFieldName(name string) {
	meta.FieldName = name
}

// GetSetter get setter from meta
func (meta Meta) GetSetter() func(resource interface{}, metaValue *MetaValue, context *core.Context) error {
	return meta.Setter
}

// SetSetter set setter to meta
func (meta *Meta) SetSetter(fc func(resource interface{}, metaValue *MetaValue, context *core.Context) error) {
	meta.Setter = fc
}

// GetValuer get valuer from meta
func (meta *Meta) GetValuer() func(interface{}, *core.Context) interface{} {
	return meta.Valuer
}

// SetValuer set valuer for meta
func (meta *Meta) SetValuer(fc func(interface{}, *core.Context) interface{}) {
	meta.Valuer = fc
}

// GetFormattedValuer get formatted valuer from meta
func (meta *Meta) GetFormattedValuer() func(interface{}, *core.Context) interface{} {
	if meta.FormattedValuer != nil {
		return meta.FormattedValuer
	}
	return meta.Valuer
}

// SetFormattedValuer set formatted valuer for meta
func (meta *Meta) SetFormattedValuer(fc func(interface{}, *core.Context) interface{}) {
	meta.FormattedValuer = fc
}

// HasPermission check has permission or not
func (meta *Meta) HasPermissionE(mode roles.PermissionMode, context *core.Context) (ok bool, err error) {
	if meta.Permission == nil {
		return true, roles.ErrDefaultPermission
	}
	var roles_ = []interface{}{}
	for _, role := range context.Roles {
		roles_ = append(roles_, role)
	}
	return roles.HasPermissionDefaultE(true, meta.Permission, mode, roles_...)
}

// SetPermission set permission for meta
func (meta *Meta) SetPermission(permission *roles.Permission) {
	meta.Permission = permission
}

// PreInitialize when will be run before initialize, used to fill some basic necessary information
func (meta *Meta) PreInitialize() error {
	if meta.Name == "" {
		utils.ExitWithMsg("Meta should have name: %v", reflect.TypeOf(meta))
	} else if meta.FieldName == "" {
		meta.FieldName = meta.Name
	}

	if meta.Name == "RegionID" {
		println()
	}

	// parseNestedField used to handle case like Profile.Name
	var parseNestedField = func(value reflect.Value, name string) (reflect.Value, string) {
		fields := strings.Split(name, ".")
		value = reflect.Indirect(value)
		for _, field := range fields[:len(fields)-1] {
			value = value.FieldByName(field)
		}

		return value, fields[len(fields)-1]
	}

	var getField = func(fields []*aorm.StructField, name string) *aorm.StructField {
		for _, field := range fields {
			if field.Name == name || field.DBName == name {
				return field
			}
		}
		return nil
	}

	var nestedField = strings.Contains(meta.FieldName, ".")
	var scope = meta.BaseResource.GetResource().FakeScope
	if nestedField {
		subModel, name := parseNestedField(reflect.ValueOf(meta.BaseResource.GetResource().Value), meta.FieldName)
		meta.FieldStruct = getField(scope.New(subModel.Interface()).GetStructFields(), name)
	} else {
		meta.FieldStruct = getField(scope.GetStructFields(), meta.FieldName)
	}
	return nil
}

// Initialize initialize meta, will set valuer, setter if haven't configure it
func (meta *Meta) Initialize() error {
	if meta.Valuer == nil {
		setupValuer(meta, meta.FieldName, meta.GetBaseResource().NewStruct())
	}

	if meta.Valuer == nil {
		utils.ExitWithMsg("Meta %v is not supported for resource %v, no `Valuer` configured for it", meta.FieldName, reflect.TypeOf(meta.BaseResource.GetResource().Value))
	}

	if meta.Setter == nil {
		setupSetter(meta, meta.FieldName, meta.GetBaseResource().NewStruct())
	}
	return nil
}

func getNestedModel(value interface{}, fieldName string, context *core.Context) interface{} {
	model := reflect.Indirect(reflect.ValueOf(value))
	fields := strings.Split(fieldName, ".")
	for _, field := range fields[:len(fields)-1] {
		if model.CanAddr() {
			submodel := model.FieldByName(field)
			if context.DB.NewRecord(submodel.Interface()) && !context.DB.NewRecord(model.Addr().Interface()) {
				if submodel.CanAddr() {
					context.DB.Model(model.Addr().Interface()).Association(field).Find(submodel.Addr().Interface())
					model = submodel
				} else {
					break
				}
			} else {
				model = submodel
			}
		}
	}

	if model.CanAddr() {
		return model.Addr().Interface()
	}
	return nil
}
