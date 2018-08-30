package serializer

import "reflect"

var SerializableFieldType = reflect.TypeOf((*SerializableField)(nil)).Elem()

type SerializableField interface {
	GetVirtualField(name string) (interface{}, bool)
}
