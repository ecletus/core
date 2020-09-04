package resource

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"

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
		meta.Valuer = func() func(record interface{}, context *core.Context) interface{} {
			var (
				modelStruct = meta.FieldStruct.BaseModel
				pth         = strings.Split(fieldName, ".")
				fieldName   string
				field       *aorm.StructField
				index       [][]int
			)
			pth, fieldName = pth[0:len(pth)-1], pth[len(pth)-1]

			for _, f := range pth {
				if f, ok := modelStruct.FieldsByName[f]; ok {
					field = f
					index = append(index, f.StructIndex)
					if modelStruct = f.BaseModel; modelStruct == nil {
						// TODO: not mapped field
						panic(fmt.Errorf("field %q#%q is not mapped", meta.BaseResource.FullID(), meta.FieldName))
						break
					}
				}
			}

			field = modelStruct.FieldsByName[fieldName]

			return func(record interface{}, context *core.Context) interface{} {
				if record == nil {
					return nil
				}
				var (
					recordReflectValue = reflect.Indirect(reflect.ValueOf(record))
					fieldValue         reflect.Value
				)

				if modelStruct == nil {
					// TODO: not mapped field
					return nil
				} else {
					for _, index := range index {
						recordReflectValue = recordReflectValue.FieldByIndex(index)
						if !recordReflectValue.IsValid() || (recordReflectValue.CanAddr() && recordReflectValue.IsNil()) {
							return nil
						}
						recordReflectValue = reflect.Indirect(recordReflectValue)
					}
					if fieldValue = recordReflectValue.FieldByIndex(field.StructIndex); !fieldValue.IsValid() {
						return nil
					}
				}

				if rel := field.Relationship; rel != nil && recordReflectValue.CanAddr() && !field.IsChild {
					var (
						recordValueInterface = recordReflectValue.Addr().Interface()
						ID                   = modelStruct.GetID(recordValueInterface)
					)
					if ID != nil && !ID.IsZero() {
						var DB = context.DB()
						if rel.Kind == "has_many" || rel.Kind == "many_to_many" {
							if fieldValue.IsNil() {
								DB = DB.ModelStruct(modelStruct, recordValueInterface)
								if err := DB.Association(field.Name).Find(fieldValue.Addr().Interface()).Error(); err != nil {
									panic(err)
								}
							}
						} else if rel.Kind == "has_one" || rel.Kind == "belongs_to" {
							if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
								if rel.GetRelatedID(record).IsZero() {
									return nil
								}
								if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
									fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
								}
							}

							if fieldID := field.Model.GetID(fieldValue.Interface()); fieldID.IsZero() {
								relatedValue := reflect.Indirect(fieldValue).Addr().Interface()
								DB = DB.ModelStruct(modelStruct, recordValueInterface)
								if err := DB.Association(field.Name).Find(relatedValue).Error(); err != nil {
									if !aorm.IsRecordNotFoundError(err) {
										context.AddError(errors.Wrapf(err, "inline preload related field %q", meta.FieldName))
									}
								}
							}
						}
					} else if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
						return nil
					}
				}

				value := fieldValue.Interface()
				if v, ok := value.(MetaValuer); ok {
					return v.MetaValue()
				}
				return value
			}
		}()
	}
}

type MetaValuer interface {
	MetaValue() interface{}
}
