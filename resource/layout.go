package resource

import (
	"github.com/go-aorm/aorm"
)

// context.Data().Set("skip.fragments", true)

type LayoutInterface interface {
	Struct
	GetType() interface{}
	Prepare(crud *CRUD) *CRUD
	FormatResult(crud *CRUD, result interface{}) interface{}
	Select(columns ...interface{})
	GetSelect() []interface{}
}

type Layout struct {
	StructValue
	PrepareFunc      func(crud *CRUD) *CRUD
	FormatResultFunc func(crud *CRUD, result interface{}) interface{}
	selects          []interface{}
}

func (l *Layout) GetType() interface{} {
	return l.Value
}

func (l *Layout) Select(columns ...interface{}) {
	l.selects = columns
}

func (l *Layout) GetSelect() []interface{} {
	return l.selects
}

func (l *Layout) Prepare(crud *CRUD) *CRUD {
	if len(l.selects) > 0 {
		var iqs aorm.InlineQueries

		for _, s := range l.selects {
			switch st := s.(type) {
			case string:
				iqs = append(iqs, aorm.IQ(st))
			case *aorm.FieldPathQuery:
				iqs = append(iqs, st)
			}
		}

		crud.SetDB(crud.DB().Select(iqs.Join()))
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
