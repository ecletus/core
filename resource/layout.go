package resource

import (
	"reflect"
)

// context.Data().Set("skip.fragments", true)

type LayoutInterface interface {
	Struct
	GetType() interface{}
	Prepare(crud *CRUD) *CRUD
	FormatResult(crud *CRUD, result interface{}) interface{}
}

type Layout struct {
	StructValue
	PrepareFunc      func(crud *CRUD) *CRUD
	FormatResultFunc func(crud *CRUD, result interface{}) interface{}
	SelectColumns    []interface{}
}

func (l *Layout) GetType() interface{} {
	return l.Value
}

func (l *Layout) Select(columns ...interface{}) *Layout {
	l.SelectColumns = columns
	return l
}

func (l *Layout) Prepare(crud *CRUD) *CRUD {
	if l.SelectColumns != nil {
		crud.SetDB(crud.DB().Select(l.SelectColumns))
	}
	if l.PrepareFunc != nil {
		return l.PrepareFunc(crud)
	}
	return crud
}

func (l *Layout) NewStruct() interface{} {
	return l.New()
}

func (l *Layout) FormatResult(crud *CRUD, recorde interface{}) interface{} {
	if l.FormatResultFunc != nil {
		return l.FormatResultFunc(crud, recorde)
	}
	return recorde
}

func ResultFormatter(slice interface{}, formatter func(i int, record interface{}), makeSlice ...func(len int)) {
	sliceValue := reflect.Indirect(reflect.ValueOf(slice))
	l := sliceValue.Len()
	if makeSlice != nil {
		makeSlice[0](l)
	}
	for i := 0; i < l; i++ {
		r := sliceValue.Index(i).Addr().Interface()
		formatter(i, r)
	}
}

func NewBasicLayout() *Layout {
	return &Layout{
		StructValue: StructValue{&Basic{}},
		FormatResultFunc: func(crud *CRUD, result interface{}) interface{} {
			var out []BasicValue
			ResultFormatter(result, func(i int, r interface{}) {
				if bv, ok := r.(BasicValue); ok {
					out[i] = bv
				} else {
					out[i] = crud.res.BasicValue(crud.context, r)
				}
			}, func(len int) {
				out = make([]BasicValue, len, len)
			})
			return out
		},
	}
}
