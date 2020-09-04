package resource

import (
	"database/sql"
	"fmt"
	"mime/multipart"
	"reflect"
	"strings"
	"time"

	"github.com/moisespsena-go/aorm"
	"github.com/pkg/errors"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
)

func SingleFieldSetter(meta *Meta, fieldName string, setter func(ptr bool, field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
	return func(record interface{}, metaValue *MetaValue, context *core.Context) (err error) {
		if metaValue == nil {
			return
		}

		defer func() {
			if err != nil {
				err = errors.Wrapf(err, "failed to set meta %v's value to %v", meta.Name, metaValue.Value)
			}
		}()

		field := reflect.Indirect(reflect.ValueOf(record)).FieldByName(fieldName)
		ptr := field.Kind() == reflect.Ptr
		if ptr {
			if utils.ToString(metaValue.Value) == "" {
				utils.SetZero(field)
				return nil
			}
		} else if !field.IsValid() || !field.CanAddr() {
			return nil
		}
		return setter(ptr, field, metaValue, context, record)
	}
}

func GenericSetter(meta *Meta, fieldName string, setter func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
	return func(record interface{}, metaValue *MetaValue, context *core.Context) (err error) {
		if metaValue == nil {
			return
		}

		defer func() {
			if err != nil {
				err = errors.Wrapf(err, "failed to set meta %v's value to %v", meta.Name, metaValue.Value)
			}
		}()

		field := reflect.Indirect(reflect.ValueOf(record)).FieldByName(fieldName)
		if field.Kind() == reflect.Ptr {
			if utils.ToString(metaValue.Value) != "" {
				if fieldStruct := metaValue.Meta.GetFieldStruct(); field.Type().Elem().Kind() == reflect.Struct &&
					metaValue.MetaValues == nil &&
					fieldStruct != nil &&
					fieldStruct.Relationship != nil &&
					(fieldStruct.Relationship.Kind == "belongs_to" ||
						fieldStruct.Relationship.Kind == "has_one") {
					metaID := metaValue.Parent.Meta.GetResource().GetMetas([]string{fieldStruct.Relationship.ForeignFieldNames[0]})[0]
					err = metaID.GetSetter()(record, metaValue, context)

					if !meta.LoadRelatedBeforeSave && !field.IsNil() {
						// set to nil
						field.Set(reflect.Zero(field.Type()))
					}

					return err
				} else {
					return setter(field, metaValue, context, record)
				}
			} else {
				field.Set(reflect.Zero(field.Type()))
			}
			return nil
		}
		return setter(field, metaValue, context, record)
	}
}

func setupSetter(meta *Meta, fieldName string, record interface{}) {
	nestedField := strings.Contains(fieldName, ".")

	// Setup nested fields
	if nestedField {
		fieldNames := strings.Split(fieldName, ".")
		setupSetter(meta, strings.Join(fieldNames[1:], "."), getNestedModel(record, strings.Join(fieldNames[0:2], "."), nil))

		oldSetter := meta.Setter
		meta.Setter = func(record interface{}, metaValue *MetaValue, context *core.Context) error {
			return oldSetter(getNestedModel(record, strings.Join(fieldNames[0:2], "."), context), metaValue, context)
		}
		return
	}

	commonSetter := func(setter func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
		return GenericSetter(meta, fieldName, setter)
	}

	// Setup child / belongs_to / many_to_many GenericSetter
	if meta.FieldStruct != nil {
		if relationship := meta.FieldStruct.Relationship; relationship != nil && !meta.FieldStruct.IsChild {
			if relationship.Kind == "belongs_to" || relationship.Kind == "many_to_many" {
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) (err error) {
					var (
						indirectValue = reflect.Indirect(reflect.ValueOf(record))
						primaryKeys   []aorm.ID
					)

					// associations not changed for belongs to
					if relationship.Kind == "belongs_to" && len(relationship.ForeignFieldNames) == 1 {
						panic("not implemented: parse primary keys")
						var oldPrimaryKeys []aorm.ID
						for _, fieldName := range relationship.ForeignFieldNames {
							oldPrimaryKeys = append(oldPrimaryKeys, indirectValue.FieldByName(fieldName).Addr().Interface().(aorm.ID))
						}
						// if not changed
						if fmt.Sprint(primaryKeys) == fmt.Sprint(oldPrimaryKeys) {
							return
						}

						// if removed
						if len(primaryKeys) == 0 {
							field := indirectValue.FieldByName(relationship.ForeignFieldNames[0])
							field.Set(reflect.Zero(field.Type()))
						}
					}

					// set current field value to blank
					field.Set(reflect.Zero(field.Type()))

					sender := aorm.SenderOf(field.Addr())
					elType, _, _ := aorm.StructTypeOf(field.Type())

					for _, value := range metaValue.Value.([]string) {
						if value == "" {
							continue
						}
						el := reflect.New(elType)
						if err := aorm.IdStringTo(value, el); err != nil {
							return errors.Wrapf(err, "value = %q", value)
						}
						sender(el)
					}

					// Replace many 2 many relations
					if relationship.Kind == "many_to_many" {
						if !aorm.ZeroIdOf(record) {
							context.DB().Model(record).Association(meta.FieldName).Replace(field.Interface())
							field.Set(reflect.Zero(field.Type()))
						}
					}
					return nil
				})
			}
		}
	}

	recordType := reflect.TypeOf(record)
	for recordType.Kind() == reflect.Ptr {
		recordType = recordType.Elem()
	}

	field := reflect.Indirect(reflect.ValueOf(record)).FieldByName(fieldName)
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(utils.NewValue(field.Type().Elem()))
		}
		field = field.Elem()
	}

	if !field.IsValid() {
		return
	}

	switch field.Addr().Interface().(type) {
	case ContextScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(ContextScanner)
			var merge bool
			if metaValue.Value != nil {
				if err := scanner.ContextScan(context, metaValue.Value); err != nil {
					return errors.Wrap(err, "context scan")
				}
				merge = true
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, field, metaValue, context, merge)
			}
			return nil
		})
	case ContextStringsScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(ContextStringsScanner)
			var merge bool
			if metaValue.Value != nil {
				if err := scanner.StringsScan(context, metaValue.Value.([]string)); err != nil {
					return errors.Wrap(err, "context strings scan")
				}
				merge = true
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, field, metaValue, context, merge)
			}
			return nil
		})
	case StringsScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(StringsScanner)
			var merge bool
			if metaValue.Value != nil {
				if err := scanner.StringsScan(metaValue.Value.([]string)); err != nil {
					return errors.Wrap(err, "context strings scan")
				}
				merge = true
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, field, metaValue, context, merge)
			}
			return nil
		})
	case aorm.StringParser:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			if scanner, ok := field.Addr().Interface().(aorm.StringParser); ok {
				return errors.Wrap(scanner.ParseString(utils.ToString(metaValue.Value)), "scan")
			}
			return nil
		})
	default:
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
				field.SetInt(utils.ToInt(metaValue.Value))
				return nil
			})
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
				field.SetUint(utils.ToUint(metaValue.Value))
				return nil
			})
		case reflect.Float32, reflect.Float64:
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
				field.SetFloat(utils.ToFloat(metaValue.Value))
				return nil
			})
		case reflect.Bool:
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
				s := utils.ToString(metaValue.Value)
				NewValueSetter(field).SetBool(s == "true" || s == "on", s == "")
				return nil
			})
		default:
			switch field.Addr().Interface().(type) {
			case sql.Scanner:
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
					if scanner, ok := field.Addr().Interface().(sql.Scanner); ok {
						if metaValue.Value == nil && len(metaValue.MetaValues.Values) > 0 {
							return decodeMetaValuesToField(meta.Resource, field, metaValue, context)
						}

						if scanner.Scan(metaValue.Value) != nil {
							return errors.Wrap(scanner.Scan(utils.ToString(metaValue.Value)), "scan")
						}
					}
					return nil
				})
			case *time.Time:
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
					if str := utils.ToString(metaValue.Value); str != "" {
						if newTime, err := utils.ParseTime(str, context); err == nil {
							if field.Kind() == reflect.Ptr {
								newValue := reflect.New(field.Type().Elem())
								newValue.Elem().Set(reflect.ValueOf(newTime))
								field.Set(newValue)
							} else {
								field.Set(reflect.ValueOf(newTime))
							}
						}
					} else {
						field.Set(reflect.Zero(field.Type()))
					}
					return nil
				})
			case *multipart.FileHeader:
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
					values := metaValue.Value.([]*multipart.FileHeader)
					if len(values) > 0 {
						v := reflect.ValueOf(values[0])
						field.Set(v)
					}
					return nil
				})
			case []*multipart.FileHeader:
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
					values := metaValue.Value.([]*multipart.FileHeader)
					if len(values) > 0 {
						field.Set(reflect.ValueOf(values))
					}
					return nil
				})
			default:
				if reflect.TypeOf("").ConvertibleTo(field.Type()) {
					meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
						field.Set(reflect.ValueOf(strings.TrimSpace(utils.ToString(metaValue.Value))).Convert(field.Type()))
						return nil
					})
				} else if reflect.TypeOf([]string{}).ConvertibleTo(field.Type()) {
					meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
						field.Set(reflect.ValueOf(utils.ToArray(metaValue.Value)).Convert(field.Type()))
						return nil
					})
				}
			}
		}
	}
}
