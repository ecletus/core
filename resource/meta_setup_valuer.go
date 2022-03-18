package resource

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

type contextKey string

const (
	AutoLoadLinkDisabled contextKey = "autoload_link_disabled"
	AutoLoadDisabled     contextKey = "autoload_disabled"
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
				if !recordReflectValue.IsValid() {
					return nil
				}

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
					recordPtr := recordReflectValue.Addr()
					recordValueInterface := recordPtr.Interface()

					if rel.Kind.Is(aorm.BELONGS_TO) {
						if fieldValue.Kind() == reflect.Ptr {
							if fieldValue.IsNil() {
								if context == nil {
									return nil
								}
								relID := rel.GetRelatedID(record)
								if relID.IsZero() {
									return nil
								}
								if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
									fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
								}
								if context.Flag(AutoLoadDisabled) {
									v := fieldValue.Interface()
									relID.SetTo(v)
									return v
								}
							} else {
								return fieldValue.Interface()
							}
						} else if context == nil {
							return nil
						}

						if fieldID := field.Model.GetID(fieldValue.Interface()); fieldID.IsZero() {
							relatedValue := reflect.Indirect(fieldValue).Addr().Interface()
							var DB = context.DB().New()
							DB = DB.ModelStruct(modelStruct, recordValueInterface).Association(field.Name).DB().Unscoped()
							if err := DB.Error; err != nil {
								aorm.SetZero(fieldValue)
								if err != aorm.ErrBlankRelatedKey {
									context.AddError(errors.Wrapf(err, "meta preload related field %q", meta.FieldName))
								}
							} else if err = DB.First(relatedValue).Error; err != nil {
								if aorm.IsRecordNotFoundError(err) {
									aorm.SetZero(fieldValue)
								} else {
									context.AddError(errors.Wrapf(err, "meta preload related field %q", meta.FieldName))
								}
							}
							return relatedValue
						}
						return fieldValue.Interface()
					} else if rel.Kind.Is(aorm.HAS_MANY, aorm.M2M, aorm.HAS_ONE, aorm.BELONGS_TO) {
						if modelStruct.PrimaryField() != nil {
							if ID := modelStruct.GetID(recordValueInterface); ID == nil || ID.IsZero() {
								goto done
							}
						}

						switch rel.Kind {
						case aorm.HAS_MANY, aorm.M2M:
							if fieldValue.IsNil() {
								if context == nil {
									return nil
								}
								var DB = context.DB().New()
								DB = DB.ModelStruct(modelStruct, recordValueInterface)
								if err := DB.Association(field.Name).Find(fieldValue.Addr().Interface()).Error(); err != nil {
									panic(err)
								}
							}

							if deletedIDs := SliceMetaGetDeleted(recordPtr, fieldName); deletedIDs != nil {
								val := &SliceValue{
									DeletedID: deletedIDs,
								}
								if fieldValue.Len() > 0 {
									val.Current = fieldValue.Interface()
								}
								var (
									deleted = reflect.New(fieldValue.Type())
									assoc   = context.DB().New().ModelStruct(modelStruct, recordValueInterface).Association(field.Name)
								)
								deleted.Elem().Set(reflect.MakeSlice(fieldValue.Type(), len(deletedIDs), len(deletedIDs)))
								assoc.Find(&aorm.RelatedResult{
									Result: deleted.Interface(),
									Prepare: func(db *aorm.DB) *aorm.DB {
										return db.Where(aorm.InID(deletedIDs...))
									},
								})
								deleted = deleted.Elem()
								if deleted.Len() > 0 {
									val.Deleted = deleted.Interface()
								}

								if val.Current != nil || val.Deleted != nil {
									return val
								}
							}
						case aorm.HAS_ONE, aorm.BELONGS_TO:
							if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
								if context == nil {
									return nil
								}
								if rel.GetRelatedID(record).IsZero() {
									return nil
								}
								if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
									fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
								}
							} else if context == nil {
								return nil
							}

							if fieldID := field.Model.GetID(fieldValue.Interface()); fieldID.IsZero() {
								relatedValue := reflect.Indirect(fieldValue).Addr().Interface()
								var DB = context.DB().New()
								DB = DB.ModelStruct(modelStruct, recordValueInterface).Association(field.Name).DB().Unscoped()
								if err := DB.Error; err != nil {
									aorm.SetZero(fieldValue)
									if err != aorm.ErrBlankRelatedKey {
										context.AddError(errors.Wrapf(err, "meta preload related field %q", meta.FieldName))
									}
								} else if err = DB.First(relatedValue).Error; err != nil {
									if aorm.IsRecordNotFoundError(err) {
										aorm.SetZero(fieldValue)
									} else {
										context.AddError(errors.Wrapf(err, "meta preload related field %q", meta.FieldName))
									}
								}
							}
						}
					} else if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
						return nil
					}
				} else if link := field.Link; link != nil {
					if fieldValue.IsNil() {
						if context == nil {
							return nil
						}

						if context.Flag(AutoLoadLinkDisabled) {
							return nil
						}

						var (
							recordValueInterface = recordReflectValue.Addr().Interface()
							ID                   = modelStruct.GetID(recordValueInterface)
						)

						if ID.IsZero() {
							return nil
						}

						if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
							if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
								fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
							}
						}

						relatedValue := reflect.Indirect(fieldValue).Addr().Interface()

						var DB = context.DB().New().Unscoped()

						DB = link.Load(field, DB, ID, relatedValue, aorm.LinkFlagTagColumnsSelector)
						if err := DB.Error; err != nil {
							aorm.SetZero(fieldValue)
							if !aorm.IsRecordNotFoundError(err) {
								context.AddError(errors.Wrapf(err, "meta preload related field %q", meta.FieldName))
							}
						}
					}
				}
			done:
				switch fieldValue.Kind() {
				case reflect.Ptr:
					if fieldValue.IsNil() {
						return nil
					}
				case reflect.Struct:
					fieldValue = fieldValue.Addr()
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
