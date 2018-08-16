package resource

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/moisespsena-go/aorm"
	"github.com/aghape/aghape"
	"github.com/aghape/aghape/utils"
	"github.com/aghape/roles"
	"github.com/aghape/validations"
)

type MetaScanner interface {
	MetaScan(value interface{}) error
}

// Metaor interface
type Metaor interface {
	GetName() string
	GetFieldName() string
	GetSetter() func(resource interface{}, metaValue *MetaValue, context *qor.Context) error
	GetFormattedValuer() func(recorde interface{}, context *qor.Context) interface{}
	GetValuer() func(recorde interface{}, context *qor.Context) interface{}
	GetContextResourcer() func(meta Metaor, context *qor.Context) Resourcer
	GetResource() Resourcer
	GetMetas() []Metaor
	GetContextMetas(recorde interface{}, context *qor.Context) []Metaor
	GetContextResource(context *qor.Context) Resourcer
	HasPermission(roles.PermissionMode, *qor.Context) bool
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
	ContextResourcer func(meta Metaor, context *qor.Context) Resourcer
	Setter           func(resource interface{}, metaValue *MetaValue, context *qor.Context) error
	Valuer           func(interface{}, *qor.Context) interface{}
	FormattedValuer  func(interface{}, *qor.Context) interface{}
	Config           MetaConfigInterface
	BaseResource     Resourcer
	Resource         Resourcer
	Permission       *roles.Permission
	Help             string
	HelpLong         string
	EditName         string
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

// GetContextResource get resource from meta
func (meta *Meta) GetContextResourcer() func(meta Metaor, context *qor.Context) Resourcer {
	return meta.ContextResourcer
}

func (meta *Meta) GetContextResource(context *qor.Context) Resourcer {
	if meta.ContextResourcer != nil {
		return meta.ContextResourcer(meta, context)
	}
	return meta.Resource
}

func (meta *Meta) GetContextMetas(recort interface{}, context *qor.Context) (metas []Metaor) {
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
func (meta Meta) GetSetter() func(resource interface{}, metaValue *MetaValue, context *qor.Context) error {
	return meta.Setter
}

// SetSetter set setter to meta
func (meta *Meta) SetSetter(fc func(resource interface{}, metaValue *MetaValue, context *qor.Context) error) {
	meta.Setter = fc
}

// GetValuer get valuer from meta
func (meta *Meta) GetValuer() func(interface{}, *qor.Context) interface{} {
	return meta.Valuer
}

// SetValuer set valuer for meta
func (meta *Meta) SetValuer(fc func(interface{}, *qor.Context) interface{}) {
	meta.Valuer = fc
}

// GetFormattedValuer get formatted valuer from meta
func (meta *Meta) GetFormattedValuer() func(interface{}, *qor.Context) interface{} {
	if meta.FormattedValuer != nil {
		return meta.FormattedValuer
	}
	return meta.Valuer
}

// SetFormattedValuer set formatted valuer for meta
func (meta *Meta) SetFormattedValuer(fc func(interface{}, *qor.Context) interface{}) {
	meta.FormattedValuer = fc
}

// HasPermission check has permission or not
func (meta *Meta) HasPermission(mode roles.PermissionMode, context *qor.Context) bool {
	if meta.Permission == nil {
		return true
	}
	var roles = []interface{}{}
	for _, role := range context.Roles {
		roles = append(roles, role)
	}
	return meta.Permission.HasPermission(mode, roles...)
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
	var (
		nestedField = strings.Contains(meta.FieldName, ".")
		field       = meta.FieldStruct
		hasColumn   = meta.FieldStruct != nil
	)

	var fieldType reflect.Type
	if hasColumn {
		fieldType = field.Struct.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
	}

	// Set Meta Valuer
	if meta.Valuer == nil {
		if hasColumn {
			meta.Valuer = func(value interface{}, context *qor.Context) interface{} {
				scope := context.DB.NewScope(value)
				fieldName := meta.FieldName
				if nestedField {
					fields := strings.Split(fieldName, ".")
					fieldName = fields[len(fields)-1]
				}

				if f, ok := scope.FieldByName(fieldName); ok {
					if relationship := f.Relationship; relationship != nil && f.Field.CanAddr() && !scope.PrimaryKeyZero() {
						if (relationship.Kind == "has_many" || relationship.Kind == "many_to_many") && f.Field.Len() == 0 {
							context.DB.Model(value).Related(f.Field.Addr().Interface(), meta.FieldName)
						} else if (relationship.Kind == "has_one" || relationship.Kind == "belongs_to") && context.DB.NewScope(f.Field.Interface()).PrimaryKeyZero() {
							if f.Field.Kind() == reflect.Ptr && f.Field.IsNil() {
								f.Field.Set(reflect.New(f.Field.Type().Elem()))
							}

							context.DB.Model(value).Related(f.Field.Addr().Interface(), meta.FieldName)
						}
					}

					return f.Field.Interface()
				}

				return ""
			}
		} else {
			utils.ExitWithMsg("Meta %v is not supported for resource %v, no `Valuer` configured for it", meta.FieldName, reflect.TypeOf(meta.BaseResource.GetResource().Value))
		}
	}

	if meta.Setter == nil && hasColumn {
		if relationship := field.Relationship; relationship != nil {
			if relationship.Kind == "belongs_to" || relationship.Kind == "many_to_many" {
				meta.Setter = func(resource interface{}, metaValue *MetaValue, context *qor.Context) (err error) {
					scope := &aorm.Scope{Value: resource}
					reflectValue := reflect.Indirect(reflect.ValueOf(resource))
					field := reflectValue.FieldByName(meta.FieldName)

					var isNil bool

					if field.Kind() == reflect.Ptr {
						if field.IsNil() {
							isNil = true
							field.Set(utils.NewValue(field.Type()).Elem())
						}

						for field.Kind() == reflect.Ptr {
							field = field.Elem()
						}
					}

					primaryKeys := utils.ToArray(metaValue.Value)
					// associations not changed for belongs to
					if relationship.Kind == "belongs_to" && len(relationship.ForeignFieldNames) == 1 {
						oldPrimaryKeys := utils.ToArray(reflectValue.FieldByName(relationship.ForeignFieldNames[0]).Interface())
						// if not changed
						if fmt.Sprint(primaryKeys) == fmt.Sprint(oldPrimaryKeys) {
							if isNil && len(primaryKeys) > 0 {
								for i, afName := range relationship.AssociationForeignFieldNames {
									field.FieldByName(afName).Set(reflect.ValueOf(primaryKeys[i]))
								}
							}
							return
						}

						// if removed
						if len(primaryKeys) == 0 {
							fkField := reflectValue.FieldByName(relationship.ForeignFieldNames[0])
							fkField.Set(reflect.Zero(fkField.Type()))
							// set nil
							field.Set(reflect.Zero(field.Type()))
							return
						} else {
							// changed
							fkField := reflectValue.FieldByName(relationship.ForeignFieldNames[0])
							fkField.Set(reflect.ValueOf(primaryKeys[0]))
							if !isNil {
								// create empty instance
								field.Set(utils.NewValue(field.Type()).Elem())
							}

							for i, afName := range relationship.AssociationForeignFieldNames {
								field.FieldByName(afName).Set(reflect.ValueOf(primaryKeys[i]))
							}
							return
						}
					}

					if len(primaryKeys) > 0 {
						// set current field value to blank and replace it with new value
						field.Set(reflect.Zero(field.Type()))
						if err = context.DB.Where(primaryKeys).Find(field.Addr().Interface()).Error; err != nil {
							return err
						}
					}

					// Replace many 2 many relations
					if relationship.Kind == "many_to_many" {
						if !scope.PrimaryKeyZero() {
							context.DB.Model(resource).Association(meta.FieldName).Replace(field.Interface())
							field.Set(reflect.Zero(field.Type()))
						}
					}
					return
				}
			}
		} else {
			meta.Setter = func(resource interface{}, metaValue *MetaValue, context *qor.Context) (err error) {
				if metaValue == nil {
					return
				}

				var (
					value     = metaValue.Value
					fieldName = meta.FieldName
				)

				defer func() {
					if r := recover(); r != nil {
						context.AddError(validations.Failed(resource, meta.Name, fmt.Sprintf("Can't set value %v", value)))
					}
				}()

				if nestedField {
					fields := strings.Split(fieldName, ".")
					fieldName = fields[len(fields)-1]
				}

				field := reflect.Indirect(reflect.ValueOf(resource)).FieldByName(fieldName)
				if field.Kind() == reflect.Ptr {
					if field.IsNil() && utils.ToString(value) != "" {
						field.Set(utils.NewValue(field.Type()).Elem())
					}

					if utils.ToString(value) == "" {
						field.Set(reflect.Zero(field.Type()))
						return nil
					}

					for field.Kind() == reflect.Ptr {
						field = field.Elem()
					}
				}

				if field.IsValid() && field.CanAddr() {
					switch field.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						field.SetInt(utils.ToInt(value))
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						field.SetUint(utils.ToUint(value))
					case reflect.Float32, reflect.Float64:
						field.SetFloat(utils.ToFloat(value))
					case reflect.Bool:
						// TODO: add test
						if stringValue := utils.ToString(value); stringValue == "true" || stringValue == "on" {
							field.SetBool(true)
						} else {
							field.SetBool(false)
						}
					default:
						if scanner, ok := field.Addr().Interface().(MetaScanner); ok {
							if value == nil && len(metaValue.MetaValues.Values) > 0 {
								context.AddError(decodeMetaValuesToField(meta.Resource, field, metaValue, context))
								return
							}

							if scanner.MetaScan(value) != nil {
								if err := scanner.MetaScan(utils.ToString(value)); err != nil {
									context.AddError(err)
									return nil
								}
							}
						} else if scanner, ok := field.Addr().Interface().(sql.Scanner); ok {
							if value == nil && len(metaValue.MetaValues.Values) > 0 {
								context.AddError(decodeMetaValuesToField(meta.Resource, field, metaValue, context))
								return
							}

							if scanner.Scan(value) != nil {
								if err := scanner.Scan(utils.ToString(value)); err != nil {
									context.AddError(err)
									return nil
								}
							}
						} else if reflect.TypeOf("").ConvertibleTo(field.Type()) {
							field.Set(reflect.ValueOf(utils.ToString(value)).Convert(field.Type()))
						} else if reflect.TypeOf([]string{}).ConvertibleTo(field.Type()) {
							field.Set(reflect.ValueOf(utils.ToArray(value)).Convert(field.Type()))
						} else if rvalue := reflect.ValueOf(value); reflect.TypeOf(rvalue.Type()).ConvertibleTo(field.Type()) {
							field.Set(rvalue.Convert(field.Type()))
						} else if _, ok := field.Addr().Interface().(*time.Time); ok {
							if str := utils.ToString(value); str != "" {
								if newTime, err := utils.ParseTime(str, context); err == nil {
									field.Set(reflect.ValueOf(newTime))
								}
							} else {
								field.Set(reflect.Zero(field.Type()))
							}
						} else {
							var buf = bytes.NewBufferString("")
							json.NewEncoder(buf).Encode(value)
							if err := json.NewDecoder(strings.NewReader(buf.String())).Decode(field.Addr().Interface()); err != nil {
								utils.ExitWithMsg("Can't set value %v to %v [meta %v]", reflect.TypeOf(value), field.Type(), meta)
							}
						}
					}
				}
				return
			}
		}
	}

	if nestedField {
		oldvalue := meta.Valuer
		meta.Valuer = func(value interface{}, context *qor.Context) interface{} {
			return oldvalue(getNestedModel(value, meta.FieldName, context), context)
		}
		oldSetter := meta.Setter
		meta.Setter = func(resource interface{}, metaValue *MetaValue, context *qor.Context) error {
			return oldSetter(getNestedModel(resource, meta.FieldName, context), metaValue, context)
		}
	}
	return nil
}

func getNestedModel(value interface{}, fieldName string, context *qor.Context) interface{} {
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
