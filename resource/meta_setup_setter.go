package resource

import (
	"database/sql"
	"mime/multipart"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/moisespsena-go/aorm"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
)

type GetFielder func(context *core.Context, record interface{}, metaValue *MetaValue) (reflect.Value, error)

func GetField(context *core.Context, record interface{}, metaValue *MetaValue) (reflect.Value, error) {
	return reflect.Indirect(reflect.ValueOf(record)).FieldByName(metaValue.Meta.GetFieldName()), nil
}

func Setter(newValue GetFielder, setter func(ptr bool, value reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
	return func(record interface{}, metaValue *MetaValue, context *core.Context) (err error) {
		if metaValue == nil {
			return
		}

		defer func() {
			if err != nil {
				err = errors.Wrapf(err, "failed to set meta %v's value to %v", metaValue.Meta.GetName(), metaValue.Value)
			}
		}()

		if newValue != nil {
			var value reflect.Value
			if value, err = newValue(context, record, metaValue); err != nil {
				return
			}
			ptr := value.Kind() == reflect.Ptr
			if ptr {
				if utils.ToString(metaValue.Value) == "" {
					utils.SetZero(value)
					return nil
				}
			} else if !value.IsValid() || !value.CanAddr() {
				return nil
			}
			return setter(ptr, value, metaValue, context, record)
		}
		return setter(false, reflect.Value{}, metaValue, context, record)
	}
}

func SingleFieldSetter(fieldName string, setter func(ptr bool, field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
	return Setter(func(context *core.Context, record interface{}, metaValue *MetaValue) (reflect.Value, error) {
		return reflect.Indirect(reflect.ValueOf(record)).FieldByName(fieldName), nil
	}, setter)
}

func SingleFieldIndexSetter(fieldIndex []int, setter func(ptr bool, field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
	return Setter(func(context *core.Context, record interface{}, metaValue *MetaValue) (reflect.Value, error) {
		return reflect.Indirect(reflect.ValueOf(record)).FieldByIndex(fieldIndex), nil
	}, setter)
}

func GenericSetter(meta Metaor, fieldName string, setter func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error) func(record interface{}, metaValue *MetaValue, context *core.Context) error {
	return func(record interface{}, metaValue *MetaValue, context *core.Context) (err error) {
		if metaValue == nil {
			return
		}

		defer func() {
			if err != nil {
				err = errors.Wrapf(err, "failed to set meta %v's value to %v", meta.GetName(), metaValue.Value)
			}
		}()

		field := reflect.Indirect(reflect.ValueOf(record)).FieldByName(fieldName)
		if field.Kind() == reflect.Ptr {
			if utils.ToString(metaValue.Value) != "" {
				if fieldStruct := metaValue.Meta.GetFieldStruct(); field.Type().Elem().Kind() == reflect.Struct &&
					metaValue.MetaValues == nil &&
					fieldStruct != nil &&
					fieldStruct.Relationship != nil &&
					(fieldStruct.Relationship.Kind == aorm.BELONGS_TO ||
						fieldStruct.Relationship.Kind == aorm.HAS_ONE) {
					metaID := metaValue.Parent.Meta.GetResource().GetMetas([]string{fieldStruct.Relationship.ForeignFieldNames[0]})[0]
					err = metaID.GetSetter()(record, metaValue, context)

					if !meta.IsLoadRelatedBeforeSave() && !field.IsNil() {
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
		model := getNestedModel(record, strings.Join(fieldNames[0:2], "."), nil)
		if model == nil {
			return
		}
		setupSetter(meta, strings.Join(fieldNames[1:], "."), model)

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
		if meta.FieldStruct.IsReadOnly {
			return
		}

		if relationship := meta.FieldStruct.Relationship; relationship != nil && !meta.FieldStruct.IsChild {
			if relationship.Kind.Is(aorm.BELONGS_TO, aorm.M2M) {
				meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) (err error) {
					var (
						indirectValue = reflect.Indirect(reflect.ValueOf(record))
					)

					// associations not changed for belongs to
					if relationship.Kind == aorm.BELONGS_TO && len(relationship.ForeignFieldNames) == 1 {
						if metaValue.MetaValues != nil {
							if metaValue.MetaValues.Disabled {
								for _, fieldName := range relationship.ForeignFieldNames {
									field := indirectValue.FieldByName(fieldName)
									field.Set(reflect.Zero(field.Type()))
								}
								// set current field value to blank
								field.Set(reflect.Zero(field.Type()))
								return
							}
						}

						field.Set(reflect.Zero(field.Type()))
						elType, _, _ := aorm.StructTypeOf(field.Type())
						el := reflect.New(elType)
						value := metaValue.FirstStringValue()
						id, err2 := relationship.AssociationModel.ParseIDString(value)
						if err2 != nil {
							return errors.Wrapf(err2, "parse id of %q", value)
						}
						id.SetTo(el.Interface())
						field.Set(el.Elem())

						for i, fieldName := range relationship.ForeignFieldNames {
							field := indirectValue.FieldByName(fieldName)
							field.Set(el.Elem().FieldByName(relationship.AssociationFieldNames[i]))
						}
						return
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
					if relationship.Kind == aorm.M2M {
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
	case MetaValueScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(MetaValueScanner)
			var f ProcessorFlag
			if metaValue.Value != nil {
				if err := scanner.MetaValueScan(context, metaValue); err != nil {
					return errors.Wrap(err, "metavalue scan")
				}
				f = ProcMerge
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, record, field, metaValue, context, f)
			}
			return nil
		})
	case ContextScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(ContextScanner)
			var f ProcessorFlag
			if metaValue.Value != nil {
				if err := scanner.ContextScan(context, metaValue.Value); err != nil {
					return errors.Wrap(err, "context scan")
				}
				f = ProcMerge
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, record, field, metaValue, context, f)
			}
			return nil
		})
	case ContextStringsScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(ContextStringsScanner)
			var f ProcessorFlag
			if metaValue.Value != nil {
				if err := scanner.StringsScan(context, metaValue.Value.([]string)); err != nil {
					return errors.Wrap(err, "context strings scan")
				}
				f = ProcMerge
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, record, field, metaValue, context, f)
			}
			return nil
		})
	case StringsScanner:
		meta.Setter = commonSetter(func(field reflect.Value, metaValue *MetaValue, context *core.Context, record interface{}) error {
			scanner := field.Addr().Interface().(StringsScanner)
			var f ProcessorFlag
			if metaValue.Value != nil {
				if err := scanner.StringsScan(metaValue.Value.([]string)); err != nil {
					return errors.Wrap(err, "context strings scan")
				}
				f = ProcMerge
			}
			if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
				return decodeMetaValuesToField(meta.Resource, record, field, metaValue, context, f)
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
							return decodeMetaValuesToField(meta.Resource, record, field, metaValue, context, 0)
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
