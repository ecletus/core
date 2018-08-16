package serializer

import "reflect"

var SerializableFieldType = reflect.TypeOf((*SerializableField)(nil)).Elem()

type SerializableField interface {
	GetSerializableField(name string) (interface{}, bool)
}
