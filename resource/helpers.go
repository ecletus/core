package resource

import (
	"reflect"

	"github.com/go-aorm/aorm"
)

type BytesParser interface {
	ParseBytes(b []byte) error
}

// MetaValuesToID to primary query params from meta values
func MetaValuesToID(res Resourcer, metaValues *MetaValues) (aorm.ID, error) {
	if metaValues != nil {
		if metaField := metaValues.Get("ID"); metaField != nil {
			return res.ParseID(metaField.StringValue())
		}
		if metaField := metaValues.Get("id"); metaField != nil {
			return res.ParseID(metaField.StringValue())
		}
	}
	return nil, nil
}

type ValueSetter struct {
	value reflect.Value
	ptr   bool
}

func NewValueSetter(value reflect.Value) *ValueSetter {
	return &ValueSetter{value, value.Kind() == reflect.Ptr}
}

func (this ValueSetter) SetNil(v bool) {

}

func (this ValueSetter) SetBool(x, null bool) {
	if this.ptr {
		v := reflect.New(this.value.Type().Elem())
		this.value.Set(v)
		if !null {
			v.Elem().SetBool(x)
		}
	} else {
		this.value.SetBool(x)
	}
}

func (this ValueSetter) SetInt(x int64) {
	if this.ptr {
		v := reflect.New(this.value.Type())
		this.value.Set(v)
		v.SetInt(x)
	} else {
		this.value.SetInt(x)
	}
}
