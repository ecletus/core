package resource

import (
	"reflect"
)

type Struct interface {
	New() interface{}
	NewSlice() interface{}
	NewSliceArgs(len, cap int) interface{}
	NewSlicePtr() interface{}
	NewSlicePtrArgs(len, cap int) interface{}
	NewSliceRecord() (slice interface{}, recorde interface{})
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
	obj := reflect.New(reflect.Indirect(reflect.ValueOf(v.Value)).Type()).Interface()

	if init, ok := obj.(interface {
		Init()
	}); ok {
		init.Init()
	}

	return obj
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
