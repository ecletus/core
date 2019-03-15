package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/moisespsena/go-path-helpers"

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
	Struct
	core.Permissioner
	GetID() string
	GetResource() *Resource
	GetPrimaryFields() []*aorm.StructField
	GetMetas([]string) []Metaor
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
	GetFakeScope() *aorm.Scope
	BasicValue(ctx *core.Context, recorde interface{}) BasicValue
	Crud(ctx *core.Context) *CRUD
	CrudDB(db *aorm.DB) *CRUD
	Layout(name string, layout LayoutInterface)
	GetLayoutOrDefault(name string) LayoutInterface
	GetLayout(name string, defaul ...string) LayoutInterface
	HasKey() bool
}

// ConfigureResourceBeforeInitializeInterface if a struct implemented this interface, it will be called before everything when create a resource with the struct
type ConfigureResourceBeforeInitializeInterface interface {
	ConfigureQorResourceBeforeInitialize(Resourcer)
}

// ConfigureResourceInterface if a struct implemented this interface, it will be called after configured by user
type ConfigureResourceInterface interface {
	ConfigureQorResource(Resourcer)
}

type CalbackFunc func(resourcer Resourcer, value interface{}, context *core.Context) error

type Callback struct {
	Name    string
	Handler CalbackFunc
}

// Resource is a struct that including basic definition of qor resource
type Resource struct {
	edis.EventDispatcher
	StructValue
	UID                string
	ID                 string
	Name               string
	PluralName         string
	PKG                string
	PkgPath            string
	I18nPrefix         string
	PrimaryFields      []*aorm.StructField
	Permission         *roles.Permission
	Validators         []func(interface{}, *MetaValues, *core.Context) error
	Processors         []func(interface{}, *MetaValues, *core.Context) error
	primaryField       *aorm.Field
	newStructCallbacks []func(obj interface{}, site core.SiteInterface)
	FakeScope          *aorm.Scope
	PathLevel          int
	ParentFieldName    string
	ParentFieldDBName  string
	ParentFieldVirtual bool
	parentResource     Resourcer
	Data               config.OtherConfig
	Layouts            map[string]LayoutInterface
}

// New initialize qor resource
func New(fakeScope *aorm.Scope, id, uid string) *Resource {
	value := fakeScope.Value

	if id == "" {
		id = utils.ModelType(value).Name()
	}
	if uid == "" {
		uid = utils.TypeId(value)
	}

	pkgPath := reflect.TypeOf(value).Elem().PkgPath()
	pkg := pkgPath
	parts := strings.Split(pkg, string(os.PathSeparator))

	for i, pth := range parts {
		if pth == "models" {
			pkg = filepath.Join(parts[:i]...)
			break
		}
	}

	var (
		name = utils.HumanizeString(id)
		res  = &Resource{
			UID:                uid,
			PKG:                pkg,
			ID:                 id,
			Name:               name,
			PluralName:         inflection.Plural(name),
			PkgPath:            pkgPath,
			FakeScope:          fakeScope,
			Data:               make(config.OtherConfig),
			Layouts:            make(map[string]LayoutInterface),
			newStructCallbacks: []func(obj interface{}, site core.SiteInterface){},
		}
	)

	res.SetI18nModel(value)
	res.Value = value
	res.SetDispatcher(res)
	res.SetPrimaryFields()
	return res
}

// NewStruct initialize a struct for the Resource
func (res *Resource) NewStruct(site ...core.SiteInterface) interface{} {
	obj := res.New()
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

func (res *Resource) HasKey() bool {
	return len(res.PrimaryFields) > 0
}

func (res *Resource) BasicValue(ctx *core.Context, record interface{}) BasicValue {
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
	if len(defaul) > 0 && defaul[0] != "" {
		return res.GetLayout(defaul[0])
	}
	return nil
}

func (res *Resource) SetI18nName(name string) {
	res.I18nPrefix = i18nmod.PkgToGroup(res.PkgPath, name)
}

func (res *Resource) SetI18nModel(value interface{}) {
	pkgPath := path_helpers.PkgPathOf(value)
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
	res.I18nPrefix = i18nmod.PkgToGroup(pkg, groupSuffix...) + "." + utils.ModelType(value).Name()
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
	if fieldName != "" {
		res.ParentFieldName = fieldName
		if f, ok := res.FakeScope.FieldByName(fieldName); ok {
			res.ParentFieldDBName = f.DBName
			res.ParentFieldVirtual = false
		} else {
			res.ParentFieldVirtual = true
		}
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

// GetMetas get defined metas, to match interface `Resourcer`
func (res *Resource) GetMetas([]string) []Metaor {
	panic("not defined")
}

func (res *Resource) HasPermissionE(mode roles.PermissionMode, context *core.Context) (ok bool, err error) {
	if res == nil || res.Permission == nil {
		return true, roles.ErrDefaultPermission
	}

	var roles_ = []interface{}{}
	for _, role := range context.Roles {
		roles_ = append(roles_, role)
	}
	return roles.HasPermissionDefaultE(true, res.Permission, mode, roles_...)
}

// StringToPrimaryQuery to primary query params
func (res *Resource) PrimaryQuery(primaryValue string, exclude ...bool) (string, []interface{}) {
	return StringToPrimaryQuery(res, primaryValue, exclude...)
}

// MetaValuesToPrimaryQuery to primary query params from meta values
func (res *Resource) MetaValuesToPrimaryQuery(metaValues *MetaValues) (string, []interface{}) {
	return MetaValuesToPrimaryQuery(res, metaValues)
}

// ValuesToPrimaryQuery to primary query params from slice values
func (res *Resource) ValuesToPrimaryQuery(exclude bool, values ...interface{}) (string, []interface{}) {
	return ValuesToPrimaryQuery(res, exclude, values...)
}

func (res *Resource) Crud(ctx *core.Context) *CRUD {
	return NewCrud(res, ctx)
}

func (res *Resource) CrudDB(db *aorm.DB) *CRUD {
	return NewCrud(res, &core.Context{DB: db})
}
