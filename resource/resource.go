package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/moisespsena-go/aorm"
	"github.com/jinzhu/inflection"
	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/aghape/aghape"
	"github.com/aghape/aghape/config"
	"github.com/aghape/aghape/utils"
	"github.com/aghape/roles"
)

const DEFAULT_LAYOUT = "default"
const BASIC_LAYOUT = "basic"

type InlineResourcer struct {
	Resource            Resourcer
	FieldName           string
	Slice               bool
	Options             map[interface{}]interface{}
	fieldIndex          []int
	keyFieldsIndex      [][]int
	BeforeSaveCallbacks []CalbackFunc
	AfterSaveCallbacks  []CalbackFunc
	Index               int
}

func (ir *InlineResourcer) BeforeSave(callbacks ...CalbackFunc) *InlineResourcer {
	ir.BeforeSaveCallbacks = append(ir.BeforeSaveCallbacks, callbacks...)
	return ir
}

func (ir *InlineResourcer) AfterSave(callbacks ...CalbackFunc) *InlineResourcer {
	ir.AfterSaveCallbacks = append(ir.AfterSaveCallbacks, callbacks...)
	return ir
}

type Parent struct {
	Resource Resourcer
	Index    int
	Parent   *Parent
	Record   interface{}
	Inline   *InlineResourcer
}

// Resourcer interface
type Resourcer interface {
	edis.EventDispatcherInterface
	GetID() string
	GetResource() *Resource
	GetPrimaryFields() []*aorm.StructField
	GetMetas([]string) []Metaor
	FindMany(result interface{}, context *qor.Context) error
	FindOne(result interface{}, metaValues *MetaValues, context *qor.Context) error
	CallFindManyLayout(r Resourcer, result interface{}, context *qor.Context, layout LayoutInterface) error
	CallFindOneLayout(r Resourcer, result interface{}, metaValues *MetaValues, context *qor.Context, layout LayoutInterface) error
	FindManyLayout(result interface{}, context *qor.Context, layout LayoutInterface) error
	FindOneLayout(result interface{}, metaValues *MetaValues, context *qor.Context, layout LayoutInterface) error
	CallSave(Resourcer, interface{}, *qor.Context) error
	CallDelete(Resourcer, interface{}, *qor.Context) error
	Save(interface{}, *qor.Context) error
	Delete(interface{}, *qor.Context) error
	NewSlice() interface{}
	NewStruct(site qor.SiteInterface) interface{}
	GetPathLevel() int
	SetParent(parent Resourcer, fieldName string)
	GetParentResource() Resourcer
	GetParentFieldName() string
	GetParentFieldDBName() string
	IsParentFieldVirtual() bool
	ToParam() string
	ParamIDPattern() string
	ParamIDName() string
	Inline(inline ...*InlineResourcer)
	GetInlines() *Inlines
	DBSave(resourcer Resourcer, result interface{}, context *qor.Context, parent *Parent) error
	GetBeforeFindCallbacks() map[string]*Callback
	GetFakeScope() *aorm.Scope
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

type CalbackFunc func(resourcer Resourcer, value interface{}, context *qor.Context, parent *Parent) error

type Callback struct {
	Name    string
	Handler CalbackFunc
}

type LayoutInterface interface {
	GetType() interface{}
	GetMany() func(Resourcer, interface{}, *qor.Context) error
	GetOne() func(Resourcer, interface{}, *MetaValues, *qor.Context) error
	GetPrepare() func(Resourcer, *qor.Context)
}

type Layout struct {
	Type    interface{}
	Many    func(Resourcer, interface{}, *qor.Context) error
	One     func(Resourcer, interface{}, *MetaValues, *qor.Context) error
	Prepare func(Resourcer, *qor.Context)
}

func (l *Layout) GetType() interface{} {
	return l.Type
}
func (l *Layout) GetMany() func(Resourcer, interface{}, *qor.Context) error {
	return l.Many
}
func (l *Layout) GetOne() func(Resourcer, interface{}, *MetaValues, *qor.Context) error {
	return l.One
}
func (l *Layout) GetPrepare() func(Resourcer, *qor.Context) {
	return l.Prepare
}

func (l *Layout) NewStruct() interface{} {
	return reflect.New(reflect.Indirect(reflect.ValueOf(l.Type)).Type()).Interface()
}

func (l *Layout) NewSlice() interface{} {
	sliceType := reflect.SliceOf(reflect.TypeOf(l.Type))
	slice := reflect.MakeSlice(sliceType, 0, 0)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	return slicePtr.Interface()
}

type Inlines struct {
	Items       []*InlineResourcer
	ByFieldName map[string]*InlineResourcer
	Len         int
}

func (inlines *Inlines) add(inline *InlineResourcer) {
	inline.Index = len(inlines.Items)
	inlines.Items = append(inlines.Items, inline)
	inlines.ByFieldName[inline.FieldName] = inline
	inlines.Len++
}

func (inlines *Inlines) Has(fieldName string) bool {
	_, ok := inlines.ByFieldName[fieldName]
	return ok
}

func (inlines *Inlines) Each(f func(inline *InlineResourcer) bool) bool {
	for _, inline := range inlines.Items {
		if !f(inline) {
			return false
		}
	}
	return true
}

// Resource is a struct that including basic definition of qor resource
type Resource struct {
	edis.EventDispatcher
	UID                       string
	ID                        string
	Name                      string
	PluralName                string
	PKG                       string
	PkgPath                   string
	I18nPrefix                string
	Value                     interface{}
	PrimaryFields             []*aorm.StructField
	FindManyHandler           func(Resourcer, interface{}, *qor.Context) error
	FindOneHandler            func(Resourcer, interface{}, *MetaValues, *qor.Context) error
	SaveHandler               func(Resourcer, interface{}, *qor.Context) error
	DeleteHandler             func(Resourcer, interface{}, *qor.Context) error
	Permission                *roles.Permission
	Validators                []func(interface{}, *MetaValues, *qor.Context) error
	Processors                []func(interface{}, *MetaValues, *qor.Context) error
	primaryField              *aorm.Field
	newStructCallbacks        []func(obj interface{}, site qor.SiteInterface)
	FakeScope                 *aorm.Scope
	DefaultFilters            []func(context *qor.Context, db *aorm.DB) *aorm.DB
	PathLevel                 int
	ParentFieldName           string
	ParentFieldDBName         string
	ParentFieldVirtual        bool
	parentResource            Resourcer
	Data                      config.OtherConfig
	beforeSaveCallbacks       map[string]*Callback
	beforeFindCallbacks       map[string]*Callback
	Layouts                   map[string]*Layout
	TransformToBasicValueFunc func(record interface{}) BasicValue
	Inlines                   *Inlines
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
			FakeScope:          qor.FakeDB.NewScope(value),
			Data:               make(config.OtherConfig),
			Inlines:            &Inlines{ByFieldName: make(map[string]*InlineResourcer)},
			I18nPrefix:         i18nmod.PkgToGroup(pkg, groupSuffix...) + "." + utils.ModelType(value).Name(),
			Layouts:            make(map[string]*Layout),
			newStructCallbacks: []func(obj interface{}, site qor.SiteInterface){},
		}
	)

	res.SetDispatcher(res)
	res.SaveHandler = res.saveHandler
	res.DeleteHandler = res.deleteHandler
	res.SetPrimaryFields()
	return res
}

func (res *Resource) GetPrimaryFields() []*aorm.StructField {
	return res.PrimaryFields
}

func (res *Resource) GetPrivateLabel() string {
	return res.Name
}

func (res *Resource) Inline(inline ...*InlineResourcer) {
	value := reflect.TypeOf(res.Value).Elem()
	for _, i := range inline {
		if field, ok := value.FieldByName(i.FieldName); ok {
			i.fieldIndex = field.Index
			keyFields := i.Resource.GetPrimaryFields()
			fmt.Println("FIELD: ", field.Name, field.Index)
			if len(keyFields) == 1 {
				fmt.Println("FK KEY FIELD: ", keyFields[0].Name, keyFields[0].StructIndex)
				if keyField, ok := value.FieldByName(i.FieldName + "ID"); ok {
					fmt.Println("KEY FIELD: ", keyField.Name, keyField.Index)
					i.keyFieldsIndex = append(i.keyFieldsIndex, keyField.Index)
				} else {
					panic(fmt.Errorf("Register inline failed: Struct %v does not have field %q", value.String(), i.FieldName+"ID"))
				}
			} else {
				for _, rKeyField := range keyFields {
					keyFieldName := i.FieldName + "Id" + rKeyField.Name
					if keyField, ok := value.FieldByName(keyFieldName); ok {
						i.keyFieldsIndex = append(i.keyFieldsIndex, keyField.Index)
					} else {
						panic(fmt.Errorf("Register inline failed: Struct %v does not have field %q", value.String(), keyFieldName))
					}
				}
			}
		} else {
			panic(fmt.Errorf("Register inline failed: Struct %v does not have field %q", value.String(), i.FieldName))
		}

		if i.Options == nil {
			i.Options = make(map[interface{}]interface{})
		}

		res.Inlines.add(i)
	}
}

func (res *Resource) GetInlines() *Inlines {
	return res.Inlines
}

func (res *Resource) Layout(name string, layout *Layout) {
	res.Layouts[name] = layout
}

func (res *Resource) GetLayoutOrDefault(name string) *Layout {
	return res.GetLayout(name, DEFAULT_LAYOUT)
}

func (res *Resource) GetLayout(name string, defaul ...string) *Layout {
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

func (res *Resource) NewStructCallback(callbacks ...func(obj interface{}, site qor.SiteInterface)) *Resource {
	res.newStructCallbacks = append(res.newStructCallbacks, callbacks...)
	return res
}

func (res *Resource) DefaultFilter(fns ...func(context *qor.Context, db *aorm.DB) *aorm.DB) {
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
func (res *Resource) AddValidator(fc func(interface{}, *MetaValues, *qor.Context) error) {
	res.Validators = append(res.Validators, fc)
}

// AddProcessor add processor to resource, it is used to process data before creating, updating, will rollback the change if it return any error
func (res *Resource) AddProcessor(fc func(interface{}, *MetaValues, *qor.Context) error) {
	res.Processors = append(res.Processors, fc)
}

// NewStruct initialize a struct for the Resource
func (res *Resource) NewStruct(site qor.SiteInterface) interface{} {
	if res.Value == nil {
		return nil
	}
	obj := reflect.New(reflect.Indirect(reflect.ValueOf(res.Value)).Type()).Interface()

	if init, ok := obj.(interface {
		Init()
	}); ok {
		init.Init()
	}

	if init, ok := obj.(interface {
		Init(siteInterface qor.SiteInterface)
	}); ok {
		init.Init(site)
	}
	for _, cb := range res.newStructCallbacks {
		cb(obj, site)
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
func (res *Resource) HasPermission(mode roles.PermissionMode, context *qor.Context) bool {
	if res == nil || res.Permission == nil {
		return true
	}

	var roles = []interface{}{}
	for _, role := range context.Roles {
		roles = append(roles, role)
	}
	return res.Permission.HasPermission(mode, roles...)
}
