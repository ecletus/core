package resource

import (
	"fmt"
	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
	"reflect"
)

type BytesParser interface {
	ParseBytes(b []byte) error
}

// MetaValuesToPrimaryQuery to primary query params from meta values
func MetaValuesToPrimaryQuery(ctx *core.Context, res Resourcer, metaValues *MetaValues, exclude bool) (string, []interface{}, error) {
	if metaValues != nil {
		if metaField := metaValues.Get("ID"); metaField != nil {
			return StringToPrimaryQuery(ctx, res, metaField.FirstStringValue(), exclude)
		}
	}
	return "", nil, nil
}

// ValuesToPrimaryQuery to primary query params from values
func ValuesToPrimaryQuery(ctx *core.Context, res Resourcer, exclude bool, values ...interface{}) (string, []interface{}) {
	var (
		sql, op string
		scope   = ctx.DB().NewScope(res.GetValue())
	)

	if values != nil {
		field := res.GetPrimaryFields()[0]
		if exclude {
			op = " NOT"
		}
		sql = fmt.Sprintf("%v.%v"+op+" IN %v", scope.QuotedTableName(), scope.Quote(field.DBName),
			aorm.TupleQueryArgs(len(values)))
	}

	return sql, values
}

type ValueSetter struct {
	value reflect.Value
	ptr bool
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
