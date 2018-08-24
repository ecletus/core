package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/aghape/core"
	"github.com/aghape/core/config"
	"github.com/aghape/core/utils"
	"github.com/aghape/roles"
	"github.com/jinzhu/inflection"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-i18n-modular/i18nmod"
)

const DEFAULT_LAYOUT = "default"
const BASIC_LAYOUT = "basic"

// Resourcer interface
type Resourcer interface {
	edis.EventDispatcherInterface
	GetID() string
	GetResource() *Resource
	GetPrimaryFields() []*aorm.StructField
	GetMetas([]string) []Metaor
	NewSlice() interface{}
	NewStruct(site ...core.SiteInterface) interface{}
	GetPathLevel() int
	SetParent(parent Resourcer, fieldName string)
	GetParentResource() Resourcer
	GetParentFieldName() string
	GetParentFieldDBName() string
	IsParentFieldVirtual() bool
	ToParam() string
	ParamIDPattern() string
	ParamIDName() string
	GetBeforeFindCallbacks() map[string]*Callback
	GetFakeScope() *aorm.Scope
	HasPermission(mode roles.PermissionMode, context *core.Context) bool
	BasicValue(recorde interface{}) BasicValue
	Crud(ctx *core.Context) *CRUD
	CrudDB(db *aorm.DB) *CRUD
	Layout(name string, layout LayoutInterface)
	GetLayoutOrDefault(name string) LayoutInterface
	GetLayout(name string, defaul ...string) LayoutInterface
}

// ConfigureResourceBeforeInitializeInterface if a struct implemented this interface, it will be called before everything when create a resource with the struct
type ConfigureResourceBeforeInitializeInterface interface {
	ConfigureQorResourceBeforeInitialize(Resourcer)
}

// ConfigureResourceInterface if a struct implemented this interface, it will be called after configured by user
type ConfigureResourceInterface interface {
	ConfigureQorResource(Resourcer)
}

type BasicValue interface {
	BasicID() string
	BasicLabel() string
	BasicIcon() string
}

type Basic struct {
	ID    string
	Label string
	Icon  string
}

func (b *Basic) BasicID() string {
	return b.ID
}

func (b *Basic) BasicLabel() string {
	return b.Label
}

func (b *Basic) BasicIcon() string {
	return b.Icon
}

type ToBasicInterface interface {
	GetBasicValue() *BasicValue
}

type CalbackFunc func(resourcer Resourcer, value interface{}, context *core.Context) error

type Callback struct {
	Name    string
	Handler CalbackFunc
}

// Resource is a struct that including basic definition of qor resource
type Resource struct {
	edis.EventDispatcher
	UID                 string
	ID                  string
	Name                string
	PluralName          string
	PKG                 string
	PkgPath             string
	I18nPrefix          string
	Value               interface{}
	PrimaryFields       []*aorm.StructField
	Permission          *roles.Permission
	Validators          []func(interface{}, *MetaValues, *core.Context) error
	Processors          []func(interface{}, *MetaValues, *core.Context) error
	primaryField        *aorm.Field
	newStructCallbacks  []func(obj interface{}, site core.SiteInterface)
	FakeScope           *aorm.Scope
	DefaultFilters      []func(context *core.Context, db *aorm.DB) *aorm.DB
	PathLevel           int
	ParentFieldName     string
	ParentFieldDBName   string
	ParentFieldVirtual  bool
	parentResource      Resourcer
	Data                config.OtherConfig
	beforeSaveCallbacks map[string]*Callback
	beforeFindCallbacks map[string]*Callback
	Layouts             map[string]LayoutInterface
}

// New initialize qor resource
func New(value interface{}, id, uid string) *Resource {
	if id == "" {
		id = utils.ModelType(value).Name()
	}
	if uid == "" {
		uid = utils.TypeId(value)
	}

	pkgPath := reflect.TypeOf(value).Elem().PkgPath()
	pkg := pkgPath
	var groupSuffix []string
	parts := strings.Split(pkg, string(os.PathSeparator))

	for i, pth := range parts {
		if pth == "models" {
			groupSuffix = parts[i:]
			pkg = filepath.Join(parts[:i]...)
			break
		}
	}

	if len(groupSuffix) == 0 {
		groupSuffix = append(groupSuffix, "models")
	}

	var (
		name = utils.HumanizeString(id)
		res  = &Resource{
			UID:                uid,
			PKG:                pkg,
			ID:                 id,
			Value:              value,
			Name:               name,
			PluralName:         inflection.Plural(name),
			PkgPath:            pkgPath,
			FakeScope:          core.FakeDB.NewScope(value),
			Data:               make(config.OtherConfig),
			I18nPrefix:         i18nmod.PkgToGroup(pkg, groupSuffix...) + "." + utils.ModelType(value).Name(),
			Layouts:            make(map[string]LayoutInterface),
			newStructCallbacks: []func(obj interface{}, site core.SiteInterface){},
		}
	)

	res.SetDispatcher(res)
	res.SetPrimaryFields()
	return res
}

func (res *Resource) BasicValue(record interface{}) BasicValue {
	return record.(BasicValue)
}

func (res *Resource) GetPrimaryFields() []*aorm.StructField {
	return res.PrimaryFields
}

func (res *Resource) GetPrivateLabel() string {
	return res.Name
}

func (res *Resource) Layout(name string, layout LayoutInterface) {
	res.Layouts[name] = layout
}

func (res *Resource) GetLayoutOrDefault(name string) LayoutInterface {
	return res.GetLayout(name, DEFAULT_LAYOUT)
}

func (res *Resource) GetLayout(name string, defaul ...string) LayoutInterface {
	if v, ok := res.Layouts[name]; ok {
		return v
	}
	if len(defaul) == 0 && defaul[0] != "" {
		return res.GetLayout(defaul[0])
	}
	return nil
}

func (res *Resource) SetI18nName(name string) {
	res.I18nPrefix = i18nmod.PkgToGroup(res.PkgPath, name)
}

func (res *Resource) SetI18nModel(value interface{}) {
	res.I18nPrefix = i18nmod.StructGroup(value)
}

func (res *Resource) BeforeSave(callbacks ...*Callback) {
	if res.beforeSaveCallbacks == nil {
		res.beforeSaveCallbacks = make(map[string]*Callback)
	}
	for _, cb := range callbacks {
		res.beforeSaveCallbacks[cb.Name] = cb
	}
}

func (res *Resource) GetBeforeFindCallbacks() map[string]*Callback {
	return res.beforeFindCallbacks
}

func (res *Resource) BeforeFind(callbacks ...*Callback) {
	if res.beforeFindCallbacks == nil {
		res.beforeFindCallbacks = make(map[string]*Callback)
	}
	for _, cb := range callbacks {
		res.beforeFindCallbacks[cb.Name] = cb
	}
}

func (res *Resource) GetID() string {
	return res.ID
}

func (res *Resource) ParamIDPattern() string {
	return ""
}

func (res *Resource) ParamIDName() string {
	return ""
}

func (res *Resource) ToParam() string {
	return ""
}

func (res *Resource) GetFakeScope() *aorm.Scope {
	return res.FakeScope
}

func (res *Resource) SetParent(parent Resourcer, fieldName string) {
	res.parentResource = parent
	res.ParentFieldName = fieldName
	if f, ok := res.FakeScope.FieldByName(fieldName); ok {
		res.ParentFieldDBName = f.DBName
		res.ParentFieldVirtual = false
	} else {
		res.ParentFieldVirtual = true
	}
}

func (res *Resource) IsParentFieldVirtual() bool {
	return res.ParentFieldVirtual
}

func (res *Resource) GetParentResource() Resourcer {
	return res.parentResource
}

func (res *Resource) GetParentFieldName() string {
	return res.ParentFieldName
}

func (res *Resource) GetParentFieldDBName() string {
	return res.ParentFieldDBName
}

func (res *Resource) GetPathLevel() int {
	return res.PathLevel
}

func (res *Resource) NewStructCallback(callbacks ...func(obj interface{}, site core.SiteInterface)) *Resource {
	res.newStructCallbacks = append(res.newStructCallbacks, callbacks...)
	return res
}

func (res *Resource) DefaultFilter(fns ...func(context *core.Context, db *aorm.DB) *aorm.DB) {
	res.DefaultFilters = append(res.DefaultFilters, fns...)
}

// SetPrimaryFields set primary fields
func (res *Resource) SetPrimaryFields(fields ...string) error {
	scope := res.FakeScope
	res.PrimaryFields = nil

	if len(fields) > 0 {
		for _, fieldName := range fields {
			if field, ok := scope.FieldByName(fieldName); ok {
				res.PrimaryFields = append(res.PrimaryFields, field.StructField)
			} else {
				return fmt.Errorf("%v is not a valid field for resource %v", fieldName, res.Name)
			}
		}
		return nil
	}

	if primaryField := scope.PrimaryField(); primaryField != nil {
		res.PrimaryFields = []*aorm.StructField{primaryField.StructField}
		return nil
	}

	return fmt.Errorf("no valid primary field for resource %v", res.Name)
}

// GetResource return itself to match interface `Resourcer`
func (res *Resource) GetResource() *Resource {
	return res
}

// AddValidator add validator to resource, it will invoked when creating, updating, and will rollback the change if validator return any error
func (res *Resource) AddValidator(fc func(interface{}, *MetaValues, *core.Context) error) {
	res.Validators = append(res.Validators, fc)
}

// AddProcessor add processor to resource, it is used to process data before creating, updating, will rollback the change if it return any error
func (res *Resource) AddProcessor(fc func(interface{}, *MetaValues, *core.Context) error) {
	res.Processors = append(res.Processors, fc)
}

// NewStruct initialize a struct for the Resource
func (res *Resource) NewStruct(site ...core.SiteInterface) interface{} {
	if res.Value == nil {
		return nil
	}
	obj := reflect.New(reflect.Indirect(reflect.ValueOf(res.Value)).Type()).Interface()

	if init, ok := obj.(interface {
		Init()
	}); ok {
		init.Init()
	}

	if len(site) != 0 && site[0] != nil {
		if init, ok := obj.(interface {
			Init(siteInterface core.SiteInterface)
		}); ok {
			init.Init(site[0])
		}

		for _, cb := range res.newStructCallbacks {
			cb(obj, site[0])
		}
	}

	return obj
}

// NewSlice initialize a slice of struct for the Resource
func (res *Resource) NewSlice() interface{} {
	if res.Value == nil {
		return nil
	}
	sliceType := reflect.SliceOf(reflect.TypeOf(res.Value))
	slice := reflect.MakeSlice(sliceType, 0, 0)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	return slicePtr.Interface()
}

// GetMetas get defined metas, to match interface `Resourcer`
func (res *Resource) GetMetas([]string) []Metaor {
	panic("not defined")
}

// HasPermission check permission of resource
func (res *Resource) HasPermission(mode roles.PermissionMode, context *core.Context) bool {
	if res == nil || res.Permission == nil {
		return true
	}

	var roles = []interface{}{}
	for _, role := range context.Roles {
		roles = append(roles, role)
	}
	return res.Permission.HasPermission(mode, roles...)
}

// ToPrimaryQueryParams to primary query params
func (res *Resource) ToPrimaryQueryParams(primaryValue string) (string, []interface{}) {
	return ToPrimaryQueryParams(res, primaryValue)
}

// ToPrimaryQueryParamsFromMetaValue to primary query params from meta values
func (res *Resource) ToPrimaryQueryParamsFromMetaValue(metaValues *MetaValues) (string, []interface{}) {
	return ToPrimaryQueryParamsFromMetaValue(res, metaValues)
}

func (res *Resource) Crud(ctx *core.Context) *CRUD {
	return NewCrud(res, ctx)
}

func (res *Resource) CrudDB(db *aorm.DB) *CRUD {
	return NewCrud(res, &core.Context{DB: db})
}
