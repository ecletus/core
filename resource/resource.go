package resource

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/moisespsena-go/maps"

	path_helpers "github.com/moisespsena-go/path-helpers"

	"github.com/ecletus/roles"
	"github.com/jinzhu/inflection"
	"github.com/moisespsena-go/edis"
	"github.com/moisespsena-go/i18n-modular/i18nmod"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
	"github.com/moisespsena-go/aorm"
)

const DEFAULT_LAYOUT = "default"
const BASIC_LAYOUT = "basic"

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
	newStructCallbacks []func(obj interface{}, site *core.Site)
	ModelStruct        *aorm.ModelStruct
	PathLevel          int
	ParentRelation     *aorm.Relationship
	parentResource     Resourcer
	Data               maps.SyncedMap
	Layouts            map[string]LayoutInterface
	Singleton          bool

	ContextSetupFunc func(ctx *core.Context) *core.Context

	ContextPermissioners,
	Permissioners []core.Permissioner
	defaultDenyMode func() bool
	Tags            aorm.TagSetting
}

// New initialize qor resource
func New(value interface{}, id, uid string, modelStruct *aorm.ModelStruct) *Resource {
	if id == "" {
		id = utils.ModelType(value).Name()
	}
	if uid == "" {
		uid = utils.TypeId(value)
	}

	if modelStruct == nil {
		modelStruct = aorm.StructOf(value)
	}
	pkgPath := modelStruct.PkgPath()
	pkg := pkgPath
	parts := strings.Split(pkg, string(os.PathSeparator))

	for i, pth := range parts {
		if pth == "models" {
			pkg = filepath.Join(parts[:i]...)
			break
		}
	}
	var (
		name = utils.HumanizeString(path.Base(id))
	)
	if reflect.TypeOf(value).Elem() != modelStruct.Type {
		panic("")
	}
	var (
		res = &Resource{
			UID:                uid,
			PKG:                pkg,
			ID:                 id,
			Name:               name,
			PluralName:         inflection.Plural(name),
			PkgPath:            pkgPath,
			ModelStruct:        modelStruct,
			Layouts:            make(map[string]LayoutInterface),
			newStructCallbacks: []func(obj interface{}, site *core.Site){},
		}
	)

	res.SetI18nModelStruct(modelStruct)
	res.Value = modelStruct.Value
	res.SetDispatcher(res)
	res.SetPrimaryFields()
	return res
}

func (res *Resource) ConfigSet(key, value interface{}) {
	res.Data.Set(key, value)
}

func (res *Resource) ConfigGet(key interface{}) (value interface{}, ok bool) {
	return res.Data.Get(key)
}

func (res *Resource) Options(opt ...core.Option) *Resource {
	for _, opt := range opt {
		opt.Apply(res)
	}
	return res
}

func (res *Resource) GetContextMetas(*core.Context) []Metaor {
	return res.GetMetas([]string{})
}

func (res *Resource) FullID() string {
	if res.parentResource != nil {
		return res.parentResource.FullID() + "." + res.ID
	}
	return res.ID
}

func (res *Resource) SetDefaultDenyMode(defaultDenyMode func() bool) {
	res.defaultDenyMode = defaultDenyMode
}

func (res *Resource) DefaultDenyMode() bool {
	if res.defaultDenyMode != nil {
		return res.defaultDenyMode()
	}
	return false
}

func (res *Resource) GetModelStruct() *aorm.ModelStruct {
	return res.ModelStruct
}

func (res *Resource) ContextSetup(ctx *core.Context) *core.Context {
	if res.ContextSetupFunc != nil {
		return res.ContextSetupFunc(ctx)
	}
	return ctx
}

func (res *Resource) ParseID(s string) (ID aorm.ID, err error) {
	return res.ModelStruct.ParseIDString(s)
}

func (res *Resource) SetID(record interface{}, id aorm.ID) {
	id.SetTo(record)
}

func (res *Resource) GetKey(value interface{}) aorm.ID {
	if value == nil {
		return nil
	}
	if idg, ok := value.(interface{ GetID() aorm.ID }); ok {
		return idg.GetID()
	}
	return res.ModelStruct.GetID(value)
}

func (res *Resource) Validate(record interface{}, values *MetaValues, ctx *core.Context, onError func(err error) (stop bool)) {
	if values.IsRequirementCheck() {
		var hasBlank bool

		for _, value := range values.Values {
			if value.Meta == nil {
				continue
			}
			if valueStr := utils.ToString(value.Value); strings.TrimSpace(valueStr) == "" {
				if value.Meta.IsRequired() {
					label := value.Meta.GetLabelC(ctx)
					onError(ErrCantBeBlank(ctx, record, value.Meta.GetName(), label))
					hasBlank = true
				}
			} else {
				for _, vldr := range value.Meta.Validators() {
					if err := vldr(record, value, ctx); err != nil {
						if onError(err) {
							return
						}
					}
				}
			}
		}
		if hasBlank {
			return
		}
	}

	for _, vldr := range res.Validators {
		if err := vldr(record, values, ctx); err != nil {
			if onError(err) {
				return
			}
		}
	}
	return
}

func (res *Resource) IsSingleton() bool {
	return res.Singleton
}

// NewStruct initialize a struct for the Resource
func (res *Resource) NewStruct(site ...*core.Site) interface{} {
	obj := res.New()
	if len(site) != 0 && site[0] != nil {
		if init, ok := obj.(interface {
			Init(siteInterface *core.Site)
		}); ok {
			init.Init(site[0])
		}

		for _, cb := range res.newStructCallbacks {
			cb(obj, site[0])
		}
	}

	return obj
}

func (res *Resource) Validator(f ...func(recorde interface{}, values *MetaValues, ctx *core.Context) (err error)) {
	res.Validators = append(res.Validators, f...)
}

func (res *Resource) HasKey() bool {
	return res.ModelStruct.Parent == nil && len(res.PrimaryFields) > 0
}

func (res *Resource) BasicValue(ctx *core.Context, record interface{}) BasicValuer {
	return record.(BasicValuer)
}

func (res *Resource) BasicDescriptableValue(ctx *core.Context, record interface{}) BasicDescriptableValuer {
	return record.(BasicDescriptableValuer)
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

func (res *Resource) SetI18nModelStruct(ms *aorm.ModelStruct, name ...string) {
	pkg := ms.PkgPath()
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
	res.I18nPrefix = i18nmod.PkgToGroup(pkg, groupSuffix...) + "."
	if len(name) > 0 {
		res.I18nPrefix += name[0]
	} else {
		res.I18nPrefix += ms.Name
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

func (res *Resource) SetParent(parent Resourcer, rel *aorm.Relationship) {
	res.parentResource = parent
	if rel != nil {
		res.ParentRelation = rel
	}
}

func (res *Resource) GetParentResource() Resourcer {
	return res.parentResource
}

func (res *Resource) GetParentRelation() *aorm.Relationship {
	return res.ParentRelation
}

func (res *Resource) GetPathLevel() int {
	return res.PathLevel
}

func (res *Resource) NewStructCallback(callbacks ...func(obj interface{}, site *core.Site)) *Resource {
	res.newStructCallbacks = append(res.newStructCallbacks, callbacks...)
	return res
}

// SetPrimaryFields set primary fields
func (res *Resource) SetPrimaryFields(fields ...string) error {
	res.PrimaryFields = nil

	if len(fields) > 0 {
		for _, fieldName := range fields {
			if field, ok := res.ModelStruct.FieldsByName[fieldName]; ok {
				res.PrimaryFields = append(res.PrimaryFields, field)
			} else {
				return fmt.Errorf("%v is not a valid field for resource %v", fieldName, res.Name)
			}
		}
		return nil
	}

	if res.PrimaryFields = res.ModelStruct.PrimaryFields; len(res.PrimaryFields) > 0 {
		return nil
	}

	return fmt.Errorf("no valid primary field for resource %v", res.Name)
}

// GetResource return itself to match interface `Resourcer`
func (res *Resource) GetResource() *Resource {
	return res
}

// AddValidator add validator to resource, it will invoked when creating, updating, and will rollback the change if validator return any error
func (res *Resource) AddValidator(fc func(record interface{}, metaValues *MetaValues, ctx *core.Context) error) {
	res.Validators = append(res.Validators, fc)
}

// AddProcessor add processor to resource, it is used to process data before creating, updating, will rollback the change if it return any error
func (res *Resource) AddProcessor(fc func(record interface{}, metaValues *MetaValues, ctx *core.Context) error) {
	res.Processors = append(res.Processors, fc)
}

// GetMetas get defined metas, to match interface `Resourcer`
func (res *Resource) GetMetas([]string) []Metaor {
	panic("not defined")
}

func (res *Resource) ContextPermissioner(p core.Permissioner, pN ...core.Permissioner) {
	res.ContextPermissioners = append(append(res.ContextPermissioners, p), pN...)
}

func (res *Resource) Permissioner(p core.Permissioner, pN ...core.Permissioner) {
	res.Permissioners = append(append(res.Permissioners, p), pN...)
}

func (res *Resource) HasPermission(mode roles.PermissionMode, context *core.Context) (perm roles.Perm) {
	for _, permissioner := range res.Permissioners {
		if perm = permissioner.HasPermission(mode, context); perm != roles.UNDEF {
			return
		}
	}
	if res.Permission != nil {
		return res.Permission.HasPermission(context, mode, context.Roles.Interfaces()...)
	}
	return
}

func (res *Resource) HasContextPermission(mode roles.PermissionMode, context *core.Context) (perm roles.Perm) {
	for _, permissioner := range res.ContextPermissioners {
		if perm = permissioner.HasPermission(mode, context); perm != roles.UNDEF {
			return
		}
	}
	return
}

// IdToPrimaryQuery to primary query params
func (res *Resource) PrimaryQuery(ctx *core.Context, primaryValue aorm.ID, exclude ...bool) (string, []interface{}, error) {
	var ex bool
	for _, ex = range exclude {
	}
	return IdToPrimaryQuery(ctx, res, ex, primaryValue)
}

// IdToPrimaryQuery to primary query params
func (res *Resource) PrimaryValues(id aorm.ID) (args []interface{}) {
	for _, v := range id.Values() {
		args = append(args, v)
	}
	return
}

// MetaValuesToPrimaryQuery to primary query params from meta values
func (res *Resource) MetaValuesToPrimaryQuery(ctx *core.Context, metaValues *MetaValues) (string, []interface{}, error) {
	return MetaValuesToPrimaryQuery(ctx, res, metaValues, false)
}

// ValuesToPrimaryQuery to primary query params from slice values
func (res *Resource) ValuesToPrimaryQuery(ctx *core.Context, exclude bool, values ...interface{}) (string, []interface{}) {
	return ValuesToPrimaryQuery(ctx, res, exclude, values...)
}

func (res *Resource) Crud(ctx *core.Context) *CRUD {
	return NewCrud(res, ctx)
}

func (res *Resource) CrudDB(db *aorm.DB) *CRUD {
	return NewCrud(res, (&core.Context{}).SetDB(db))
}
