package resource

import "reflect"

func IsDefaultMetaSetter(meta *Meta) bool {
	fpkg := reflect.TypeOf(meta.Setter).PkgPath()
	return fpkg == pkg
}

