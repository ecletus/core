package resource

import "reflect"

func SliceMetaAppendDeleted(record reflect.Value, metaName string, pks ...ID) {
	if m := record.MethodByName("AppendDeleted" + metaName); m.IsValid() {
		var args = make([]reflect.Value, len(pks))
		for i, pk := range pks {
			args[i] = reflect.ValueOf(pk)
		}
		m.Call(args)
	}
}

func SliceMetaGetDeleted(record reflect.Value, metaName string) []ID {
	if m := record.MethodByName("GetDeleted" + metaName); m.IsValid() {
		res := m.Call(nil)
		return res[0].Interface().([]ID)
	}
	return nil
}
