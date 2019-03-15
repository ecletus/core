package resource

import (
	"reflect"
	"strings"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

func setupValuer(meta *Meta, fieldName string, record interface{}) {
	nestedField := strings.Contains(fieldName, ".")

	// Setup nested fields
	if nestedField {
		fieldNames := strings.Split(fieldName, ".")
		setupValuer(meta, strings.Join(fieldNames[1:], "."), getNestedModel(record, strings.Join(fieldNames[0:2], "."), nil))

		oldValuer := meta.Valuer
		meta.Valuer = func(record interface{}, context *core.Context) interface{} {
			return oldValuer(getNestedModel(record, strings.Join(fieldNames[0:2], "."), context), context)
		}
		return
	}

	if meta.FieldStruct != nil {
		meta.Valuer = func(value interface{}, context *core.Context) interface{} {
			if value == nil {
				return nil
			}
			if context == nil {
				v := reflect.ValueOf(value)
				for _, f := range strings.Split(meta.FieldName, ".") {
					if v = reflect.Indirect(v).FieldByName(f); !v.IsValid() {
						return nil
					}
				}

				return v.Interface()
			}
			scope := context.DB.NewScope(value)
			fieldName := meta.FieldName
			if nestedField {
				fields := strings.Split(fieldName, ".")
				fieldName = fields[len(fields)-1]
			}

			if f, ok := scope.FieldByName(fieldName); ok {
				if relationship := f.Relationship; relationship != nil && f.Field.CanAddr() {
					if !scope.PrimaryKeyZero() {
						if (relationship.Kind == "has_many" || relationship.Kind == "many_to_many") && f.Field.Len() == 0 {
							context.DB.Model(value).Related(f.Field.Addr().Interface(), meta.FieldName)
						} else if relationship.Kind == "has_one" || relationship.Kind == "belongs_to" {
							if f.Field.Kind() == reflect.Ptr && f.Field.IsNil() {
								var idValues []interface{}
								value := reflect.Indirect(reflect.ValueOf(value))
								for _, fieldName := range relationship.ForeignFieldNames {
									if idValue := value.FieldByName(fieldName); idValue.IsValid() {
										idValues = append(idValues, idValue.Interface())
									} else {
										idValues = append(idValues, nil)
									}
								}

								if aorm.Key(idValues...).String() == "" {
									return nil
								}

								if f.Field.Kind() == reflect.Ptr && f.Field.IsNil() {
									f.Field.Set(reflect.New(f.Field.Type().Elem()))
								}
							}

							if scope := context.DB.NewScope(f.Field.Interface()); scope.PrimaryKeyZero() {
								relatedValue := reflect.Indirect(f.Field).Addr().Interface()
								context.DB.Model(value).AutoInlinePreload(relatedValue).Related(relatedValue, meta.FieldName)
							}
						}
					} else if f.Field.Kind() == reflect.Ptr && f.Field.IsNil() {
						return nil
					}
				}

				return f.Field.Interface()
			}

			return ""
		}
	}
}
