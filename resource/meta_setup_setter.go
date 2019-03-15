package resource

import (
	"database/sql"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aghape/core"
	"github.com/aghape/core/utils"
	"github.com/aghape/validations"
)

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

	commonSetter := func(setter func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{})) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
		return func(record interface{}, metaValue *MetaValue, context *core.Context) (err error) {
			if metaValue == nil {
				return
			}

			defer func() {
				if r := recover(); r != nil {
					debug.PrintStack()
					fmt.Println(r)
					context.AddError(validations.NewError(record, meta.Name, fmt.Sprintf("Failed to set Meta %v's value with %v, got %v", meta.Name, metaValue.Value, r)))
				}
			}()

			field := reflect.Indirect(reflect.ValueOf(record)).FieldByName(fieldName)
			if field.Kind() == reflect.Ptr {
				if utils.ToString(metaValue.Value) != "" {
					if fieldStruct := metaValue.Meta.GetFieldStruct(); field.Type().Elem().Kind() == reflect.Struct &&
						metaValue.MetaValues == nil &&
						fieldStruct != nil &&
						fieldStruct.Relationship != nil && fieldStruct.Relationship.Kind == "belongs_to" {
						metaID := metaValue.Parent.Meta.GetResource().GetMetas([]string{fieldStruct.Relationship.ForeignFieldNames[0]})[0]
						err = metaID.GetSetter()(record, metaValue, context)

						if !field.IsNil() {
							// set to nil
							field.Set(reflect.Zero(field.Type()))
						}

						return err
					} else {
						setter(field, metaValue, context, record)
						return
					}
				} else {
					field.Set(reflect.Zero(field.Type()))
					return
				}
			}

			if field.IsValid() && field.CanAddr() {
				setter(field, metaValue, context, record)
			}
			return nil
		}
	}

	// Setup belongs_to / many_to_many Setter
	if meta.FieldStruct != nil {
		if relationship := meta.FieldStruct.Relationship; relationship != nil {
			if relationship.Kind == "belongs_to" || relationship.Kind == "many_to_many" {
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
					var (
						scope         = context.GetDB().NewScope(record)
						indirectValue = reflect.Indirect(reflect.ValueOf(record))
					)
					primaryKeys := utils.ToArray(metaValue.Value)
					if metaValue.Value == nil {
						primaryKeys = []string{}
					}

					// associations not changed for belongs to
					if relationship.Kind == "belongs_to" && len(relationship.ForeignFieldNames) == 1 {
						oldPrimaryKeys := utils.ToArray(indirectValue.FieldByName(relationship.ForeignFieldNames[0]).Interface())
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

					if len(primaryKeys) > 0 {
						// replace it with new value
						context.GetDB().Where(primaryKeys).Find(field.Addr().Interface())
					}

					// Replace many 2 many relations
					if relationship.Kind == "many_to_many" {
						if !scope.PrimaryKeyZero() {
							context.GetDB().Model(record).Association(meta.FieldName).Replace(field.Interface())
							field.Set(reflect.Zero(field.Type()))
						}
					}
				})
				return
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

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
			field.SetInt(utils.ToInt(metaValue.Value))
		})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
			field.SetUint(utils.ToUint(metaValue.Value))
		})
	case reflect.Float32, reflect.Float64:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
			field.SetFloat(utils.ToFloat(metaValue.Value))
		})
	case reflect.Bool:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
			if s := utils.ToString(metaValue.Value); s == "true" || s == "on" {
				field.SetBool(true)
			} else {
				field.SetBool(false)
			}
		})
	default:
		if _, ok := field.Addr().Interface().(ContextScanner); ok {
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
				if scanner, ok := field.Addr().Interface().(ContextScanner); ok {
					var merge bool
					if metaValue.Value != nil {
						if err := scanner.ContextScan(context, metaValue.Value); err != nil {
							context.AddError(err)
							return
						}
						merge = true
					}
					if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
						decodeMetaValuesToField(meta.Resource, field, metaValue, context, merge)
						return
					}
				}
			})
		} else if _, ok := field.Addr().Interface().(sql.Scanner); ok {
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
				if scanner, ok := field.Addr().Interface().(sql.Scanner); ok {
					if metaValue.Value == nil && len(metaValue.MetaValues.Values) > 0 {
						decodeMetaValuesToField(meta.Resource, field, metaValue, context)
						return
					}

					if scanner.Scan(metaValue.Value) != nil {
						if err := scanner.Scan(utils.ToString(metaValue.Value)); err != nil {
							context.AddError(err)
							return
						}
					}
				}
			})
		} else if reflect.TypeOf("").ConvertibleTo(field.Type()) {
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
				field.Set(reflect.ValueOf(utils.ToString(metaValue.Value)).Convert(field.Type()))
			})
		} else if reflect.TypeOf([]string{}).ConvertibleTo(field.Type()) {
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
				field.Set(reflect.ValueOf(utils.ToArray(metaValue.Value)).Convert(field.Type()))
			})
		} else if _, ok := field.Addr().Interface().(*time.Time); ok {
			meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) {
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
			})
		}
	}
}
