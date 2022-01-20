package resource

import "C"
import (
	"reflect"

	"github.com/moisespsena-go/aorm"
	"github.com/pkg/errors"
)

type Struct interface {
	New() interface{}
	NewSlice() interface{}
	NewSliceArgs(len, cap int) interface{}
	NewSlicePtr() interface{}
	NewSlicePtrArgs(len, cap int) interface{}
	NewSliceRecord() (slice interface{}, recorde interface{})
	NewChan(buf int) interface{}
	NewChanPtr(buf int) interface{}
	GetValue() interface{}
}

type StructValue struct {
	Value interface{}
}

func NewStructValue(value interface{}) StructValue {
	return StructValue{value}
}

func (v StructValue) GetValue() interface{} {
	return v.Value
}

// NewStruct initialize a struct for the Value
func (v StructValue) New() interface{} {
	if v.Value == nil {
		return nil
	}

	obj := reflect.New(aorm.MustStructTypeOfInterface(v.Value)).Interface()

	if init, ok := obj.(interface {
		Init()
	}); ok {
		init.Init()
	}

	return obj
}

func (this *Resource) NewForIdS(id string) (interface{}, error) {
	if ID, err := this.ParseID(id); err != nil {
		return nil, errors.Wrapf(err, "Resource %q: ParseID %s", this.UID, id)
	} else {
		return ID.SetTo(this.New()), nil
	}
}

func (v StructValue) NewSlicePtrArgs(len, cap int) interface{} {
	return v.newSliceArgs(len, cap, true)
}

func (v StructValue) NewSliceArgs(len, cap int) interface{} {
	return v.newSliceArgs(len, cap, false)
}

// NewSlice initialize a slice of struct for the Value
func (v StructValue) NewSlice() interface{} {
	return v.newSliceArgs(0, 0, false)
}

// NewSlice initialize a slice of struct for the Value
func (v StructValue) NewSlicePtr() interface{} {
	return v.newSliceArgs(0, 0, true)
}

// NewChan initialize a channel of struct for the Value
func (v StructValue) NewChan(buf int) interface{} {
	return v.newChanArgs(buf, false)
}

// NewChanPtr initialize a channel of struct ptr for the Value
func (v StructValue) NewChanPtr(buf int) interface{} {
	return v.newChanArgs(buf, true)
}

// NewSlice initialize a slice of struct for the Value
func (v StructValue) NewSliceRecord() (slice interface{}, recorde interface{}) {
	slice = v.NewSliceArgs(1, 1)
	recorde = reflect.ValueOf(slice).Index(0).Addr().Interface()
	return
}

// NewSlice initialize a slice of struct for the Value
func (v StructValue) newSliceArgs(len, cap int, ptr bool) interface{} {
	if v.Value == nil {
		return nil
	}
	typ := reflect.TypeOf(v.Value)
	if !ptr {
		typ = typ.Elem()
	}
	sliceType := reflect.SliceOf(typ)

	if len > 0 && cap == 0 {
		cap = len
	}

	slice := reflect.MakeSlice(sliceType, len, cap)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	if len != 0 {
		return slice.Interface()
	}
	return slicePtr.Interface()
}

// NewSlice initialize a slice of struct for the Value
func (v StructValue) newChanArgs(buf int, ptr bool) interface{} {
	if v.Value == nil {
		return nil
	}
	typ := reflect.TypeOf(v.Value)
	if !ptr {
		typ = typ.Elem()
	}
	var (
		t = reflect.ChanOf(reflect.BothDir, typ)
		c = reflect.MakeChan(t, buf)
		r = reflect.New(t)
	)
	r.Elem().Set(c)
	return r.Interface()
}
