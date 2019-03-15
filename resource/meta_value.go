package resource

import (
	"reflect"

	"github.com/ecletus/core"
)

// MetaValues is slice of MetaValue
type MetaValues struct {
	Values []*MetaValue
}

func (mvs *MetaValues) IsEmpty() bool {
	return mvs == nil || len(mvs.Values) == 0
}

// Get get meta value from MetaValues with name
func (mvs MetaValues) Get(name string) *MetaValue {
	for _, mv := range mvs.Values {
		if mv.Name == name {
			return mv
		}
	}

	return nil
}

// MetaValue a struct used to hold information when convert inputs from HTTP form, JSON, CSV files and so on to meta values
// It will includes field name, field value and its configured Meta, if it is a nested resource, will includes nested metas in its MetaValues
type MetaValue struct {
	Parent     *MetaValue
	Name       string
	Value      interface{}
	Index      int
	MetaValues *MetaValues
	Meta       Metaor
	error      error
}

func decodeMetaValuesToField(res Resourcer, field reflect.Value, metaValue *MetaValue, context *core.Context, merge ...bool) (err error) {
	//if field.Kind() == reflect.Struct {
	if metaValue.Meta.IsInline() {
		typ := field.Type()
		isPtr := typ.Kind() == reflect.Ptr
		if isPtr {
			typ = typ.Elem()
		}
		var value = field
		if len(merge) > 0 && merge[0] {
			if isPtr {
				value = field.Elem()
			} else {
				value = field.Addr()
			}
		} else {
			value = reflect.New(typ)
		}
		valueInterface := value.Interface()
		associationProcessor := DecodeToResource(res, valueInterface, metaValue.MetaValues, context)
		err = associationProcessor.Start()
		if err != nil {
			return
		}
		if !associationProcessor.SkipLeft {
			if isPtr {
				field.Set(value)
			} else {
				field.Set(value.Elem())
			}
		}
	} else if field.Kind() == reflect.Slice {
		if metaValue.Index == 0 {
			field.Set(reflect.Zero(field.Type()))
		}

		var fieldType = field.Type().Elem()
		var isPtr bool
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
			isPtr = true
		}

		value := reflect.New(fieldType)
		associationProcessor := DecodeToResource(res, value.Interface(), metaValue.MetaValues, context)
		err = associationProcessor.Start()
		if err != nil {
			return
		}
		if !associationProcessor.SkipLeft {
			if !reflect.DeepEqual(reflect.Zero(fieldType).Interface(), value.Elem().Interface()) {
				if isPtr {
					field.Set(reflect.Append(field, value))
				} else {
					field.Set(reflect.Append(field, value.Elem()))
				}
			}
		}
	}
	return
}
