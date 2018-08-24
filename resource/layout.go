package resource

import (
	"reflect"
)

// context.Data().Set("skip.fragments", true)

type LayoutInterface interface {
	GetType() interface{}
	Prepare(crud *CRUD) *CRUD
	NewStruct() interface{}
	NewSlice() interface{}
	FormatResult(crud *CRUD, result interface{})
}

type Layout struct {
	Type             interface{}
	PrepareFunc      func(crud *CRUD) *CRUD
	FormatResultFunc func(crud *CRUD, result interface{})
}

func (l *Layout) GetType() interface{} {
	return l.Type
}

func (l *Layout) Prepare(crud *CRUD) *CRUD {
	if l.PrepareFunc != nil {
		return l.PrepareFunc(crud)
	}
	return crud
}

func (l *Layout) NewStruct() interface{} {
	return reflect.New(reflect.Indirect(reflect.ValueOf(l.Type)).Type()).Interface()
}

func (l *Layout) NewSlice() interface{} {
	sliceType := reflect.SliceOf(reflect.TypeOf(l.Type))
	slice := reflect.MakeSlice(sliceType, 0, 0)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	return slicePtr.Interface()
}

func (l *Layout) FormatResult(crud *CRUD, recorde interface{}) {
	if l.FormatResultFunc != nil {
		l.FormatResultFunc(crud, recorde)
	}
}

func NewBasicLayout(res Resourcer) LayoutInterface {
	return &Layout{
		Type: res.GetResource().Value,
		FormatResultFunc: func(crud *CRUD, result interface{}) {
			if items, ok := result.([]interface{}); ok {
				for i, r := range items {
					if _, ok := r.(BasicValue); !ok {
						items[i] = crud.res.BasicValue(r)
					}
				}
			}
		},
	}
}
