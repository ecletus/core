package resource

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/moisespsena-go/maps"

	"github.com/ecletus/core/utils"

	"github.com/ecletus/roles"

	"github.com/ecletus/core"
	"github.com/go-aorm/aorm"
)

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

type FContextResourcer = func(meta Metaor, context *core.Context) Resourcer
type FSetter = func(resource interface{}, metaValue *MetaValue, context *core.Context) error
type FValuer = func(interface{}, *core.Context) interface{}
type FFormattedValuer = func(interface{}, *core.Context) *FormattedValue

// Meta meta struct definition
type Meta struct {
	*MetaName
	Alias                      *MetaName
	FieldName                  string
	FieldStruct                *aorm.StructField
	ContextResourcer           FContextResourcer
	Setter                     FSetter
	Valuer                     FValuer
	GetRecordHandler           func(ctx *core.Context, record interface{}) interface{}
	FormattedValuer            FFormattedValuer
	Config                     MetaConfigInterface
	BaseResource               Resourcer
	Resource                   Resourcer
	Permission                 *roles.Permission
	Help                       string
	HelpLong                   string
	SaveID                     bool
	Inline                     bool
	Required                   bool
	Icon                       bool
	validators                 []func(record interface{}, values *MetaValue, ctx *core.Context) (err error)
	Data                       maps.Map
	Typ                        reflect.Type
	UIValidatorFunc            func(ctx *core.Context, recorde interface{}) string
	LoadRelatedBeforeSave      bool
	DisableSiblingsRequirement SiblingsRequirementCheckDisabled
	IsCollection               bool
	Permissioners              []core.Permissioner

	Severity       core.Severity
	SeveritifyFunc func(fv *FormattedValue)

	DefaultDeny bool
}

func (this *Meta) IsLoadRelatedBeforeSave() bool {
	return this.LoadRelatedBeforeSave
}

func (this *Meta) DefaultPermissionDeny() bool {
	return this.DefaultDeny
}

func (this *Meta) Record(record interface{}) interface{} {
	ix := this.FieldStruct.StructIndex
	if len(ix) == 1 {
		return record
	}
	ix = ix[0 : len(ix)-1]
	recordValue := reflect.Indirect(reflect.ValueOf(record)).FieldByIndex(ix)
	if recordValue.Kind() != reflect.Ptr {
		recordValue = recordValue.Addr()
	}
	return recordValue.Interface()
}

func (this *Meta) GetReflectStructValueOrInstantiate(record reflect.Value) reflect.Value {
	recordValue := record.FieldByIndex(this.FieldStruct.StructIndex)
	return recordValue
}

func (this *Meta) Proxier() bool {
	return false
}

func (this *Meta) IsAlone() bool {
	return false
}

func (this *Meta) CanCollection() bool {
	return this.IsCollection || (this.FieldStruct != nil && this.FieldStruct.Struct.Type.Kind() == reflect.Slice)
}

func (this *Meta) IsSiblingsRequirementCheckDisabled() SiblingsRequirementCheckDisabled {
	return this.DisableSiblingsRequirement
}

func (this *Meta) UIValidator(ctx *core.Context, recorde interface{}) string {
	if this.UIValidatorFunc != nil {
		return this.UIValidatorFunc(ctx, recorde)
	}
	return ""
}

func (this *Meta) Validators() []func(record interface{}, values *MetaValue, ctx *core.Context) (err error) {
	return this.validators
}

func (this *Meta) RecordValidator(f ...func(record interface{}, ctx *core.Context) (err error)) *Meta {
	this.BaseResource.GetResource().AddProcessor(func(i interface{}, _ *MetaValues, context *core.Context) (err error) {
		for _, f := range f {
			if err = f(i, context); err != nil {
				return
			}
		}
		return
	})
	return this
}

func (this *Meta) Validator(f ...func(record interface{}, values *MetaValue, ctx *core.Context) (err error)) *Meta {
	this.validators = append(this.validators, f...)
	return this
}

func (this *Meta) IsZero(recorde, value interface{}) bool {
	return value == nil
}

func (this *Meta) GetLabelC(ctx *core.Context) string {
	return utils.HumanizeString(this.MetaName.Name)
}

func (this *Meta) GetRecordLabelC(ctx *core.Context, record interface{}) string {
	return utils.HumanizeString(this.MetaName.Name)
}

func (this *Meta) Namer() *MetaName {
	if this.Alias != nil {
		return this.Alias
	}
	return this.MetaName
}

func (this *Meta) IsInline() bool {
	return this.Inline
}

func (this *Meta) IsRequired() bool {
	return this.Required
}

func (this *Meta) RecordRequirer() func(ctx *core.Context, record interface{}) bool {
	return nil
}

// GetBaseResource get base resource from meta
func (this *Meta) GetBaseResource() Resourcer {
	return this.BaseResource
}

// GetFieldStruct get aorm field struct
func (this *Meta) GetFieldStruct() *aorm.StructField {
	return this.FieldStruct
}

// GetContextResource get resource from meta
func (this *Meta) GetContextResourcer() func(meta Metaor, context *core.Context) Resourcer {
	return this.ContextResourcer
}

func (this *Meta) GetContextResource(context *core.Context) Resourcer {
	if this.ContextResourcer != nil {
		return this.ContextResourcer(this, context)
	}
	return this.Resource
}

func (this *Meta) GetContextMetas(record interface{}, context *core.Context) (metas []Metaor) {
	return this.GetContextResource(context).GetMetas([]string{})
}

func (this *Meta) GetMetas() (metas []Metaor) {
	return
}

func (this *Meta) GetResource() Resourcer {
	return this.Resource
}

// GetFieldName get meta's field name
func (this *Meta) GetFieldName() string {
	return this.FieldName
}

// SetFieldName set meta's field name
func (this *Meta) SetFieldName(name string) {
	this.FieldName = name
}

// GetSetter get setter from meta
func (this Meta) GetSetter() func(recorde interface{}, metaValue *MetaValue, context *core.Context) error {
	return this.Setter
}

// SetSetter set setter to meta
func (this *Meta) SetSetter(fc func(recorde interface{}, metaValue *MetaValue, context *core.Context) error) {
	this.Setter = fc
}

// GetValuer get valuer from meta
func (this *Meta) GetValuer() func(interface{}, *core.Context) interface{} {
	return this.Valuer
}

// SetValuer set valuer for meta
func (this *Meta) SetValuer(fc func(interface{}, *core.Context) interface{}) {
	this.Valuer = fc
}

func (this *Meta) Severitify(fv *FormattedValue) *FormattedValue {
	if fv.Severity != 0 {
		return fv
	}
	if this.SeveritifyFunc != nil {
		this.SeveritifyFunc(fv)
	}
	if fv.Severity == 0 {
		fv.Severity = this.Severity
	}
	return fv
}

// GetFormattedValuer get formatted valuer from meta
func (this *Meta) GetFormattedValuer() func(interface{}, *core.Context) *FormattedValue {
	if this.FormattedValuer != nil {
		return this.FormattedValuer
	}
	return func(r interface{}, context *core.Context) *FormattedValue {
		return this.Severitify(&FormattedValue{Record: r, Raw: this.GetValuer()(r, context), IsZeroF: this.IsZero})
	}
}

// SetFormattedValuer set formatted valuer for meta
func (this *Meta) SetFormattedValuer(fc func(interface{}, *core.Context) *FormattedValue) {
	this.FormattedValuer = fc
}

// AdminHasContextPermission check has permission or not
func (this *Meta) HasPermission(mode roles.PermissionMode, context *core.Context) (perm roles.Perm) {
	if this.Permission == nil {
		return
	}
	return this.Permission.HasPermission(context, mode, context.Roles.Interfaces()...)
}

func (this *Meta) Permissioner(p ...core.Permissioner) {
	this.Permissioners = append(this.Permissioners, p...)
}

// SetPermission set permission for meta
func (this *Meta) SetPermission(permission *roles.Permission) {
	this.Permission = permission
}

// PreInitialize when will be run before initialize, used to fill some basic necessary information
func (this *Meta) PreInitialize() error {
	if this.Name == "" {
		panic(fmt.Errorf("Meta should have name: %v", reflect.TypeOf(this)))
	} else if this.FieldName == "" {
		this.FieldName = this.Name
	}
	if this.Typ == nil {
		if this.FieldStruct = this.BaseResource.GetModelStruct().FieldByPath(this.FieldName); this.FieldStruct != nil {
			this.Typ = this.FieldStruct.Struct.Type
		}
	}
	return nil
}

// Initialize initialize meta, will set valuer, setter if haven't configure it
func (this *Meta) Initialize(virtual bool) error {
	if virtual {
		return nil
	}
	if this.Valuer == nil {
		setupValuer(this, this.FieldName, this.GetBaseResource().GetValue())
	}

	if this.Valuer == nil {
		panic(fmt.Errorf("Meta %q is not supported for resource %v, no `Valuer` configured for it", this.Name, reflect.TypeOf(this.BaseResource.GetResource().Value)))
	}

	if this.Setter == nil {
		setupSetter(this, this.FieldName, this.GetBaseResource().NewStruct())
	}
	return nil
}

func (this *Meta) DBName() string {
	if this.FieldStruct != nil {
		return this.FieldStruct.DBName
	}
	return ""
}

func (this *Meta) IsNewRecord(value interface{}) bool {
	if value == nil {
		return true
	}
	if this.Resource == nil || reflect.ValueOf(this.Resource).IsNil() || this.Resource.GetModelStruct().Dummy {
		return false
	}
	if this.FieldStruct != nil && this.FieldStruct.IsChild {
		return false
	}
	if idGetter, ok := value.(aorm.IDGetter); ok {
		return idGetter.GetID().IsZero()
	}
	if struc := aorm.StructOf(value); struc != nil && len(struc.PrimaryFields) > 0 && struc.GetID(value).IsZero() {
		return true
	}
	return false
}

func getNestedModel(value interface{}, fieldName string, context *core.Context) interface{} {
	model := reflect.Indirect(reflect.ValueOf(value))
	fields := strings.Split(fieldName, ".")
	for _, field := range fields[:len(fields)-1] {
		if model.CanAddr() {
			submodel := model.FieldByName(field)
			if !submodel.IsValid() {
				return nil
			}
			if aorm.ZeroIdOf(submodel.Interface()) && aorm.ZeroIdOf(model.Addr().Interface()) {
				if submodel.CanAddr() {
					if err := context.DB().Model(model.Addr().Interface()).Association(field).Find(submodel.Addr().Interface()).Error(); err != nil {
						if !aorm.IsRecordNotFoundError(err) {
							panic(err)
						}
					}
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
