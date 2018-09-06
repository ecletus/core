package resource

import (
	"reflect"
	"strings"

	"github.com/aghape/core"
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

						context.DB.Model(value).AutoInlinePreload(value).Related(f.Field.Addr().Interface(), meta.FieldName)
					}
				}

				return f.Field.Interface()
			}

			return ""
		}
	}
}
