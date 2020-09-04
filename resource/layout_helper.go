package resource

import "reflect"

func ResultFormatter(slice interface{}, formatter func(i int, record interface{}), makeSlice ...func(len int)) {
	sliceValue := reflect.Indirect(reflect.ValueOf(slice))
	l := sliceValue.Len()
	if makeSlice != nil {
		makeSlice[0](l)
	}
	for i := 0; i < l; i++ {
		r := sliceValue.Index(i)
		if r.Kind() != reflect.Ptr {
			r = r.Addr()
		}
		formatter(i, r.Interface())
	}
}

func NewBasicLayout() *Layout {
	return &Layout{
		StructValue: StructValue{&Basic{}},
		FormatResultFunc: func(crud *CRUD, result interface{}) interface{} {
			var out []BasicValuer
			ResultFormatter(result, func(i int, r interface{}) {
				if bv, ok := r.(BasicValuer); ok {
					out[i] = bv
				} else {
					out[i] = crud.res.BasicValue(crud.context, r)
				}
			}, func(len int) {
				out = make([]BasicValuer, len, len)
			})
			return out
		},
	}
}

func NewBasicDescriptionLayout() *Layout {
	return &Layout{
		StructValue: StructValue{&BasicDescriptableValue{}},
		FormatResultFunc: func(crud *CRUD, result interface{}) interface{} {
			var out []BasicDescriptableValuer
			ResultFormatter(result, func(i int, r interface{}) {
				if bv, ok := r.(BasicDescriptableValuer); ok {
					out[i] = bv
				} else {
					out[i] = crud.res.BasicDescriptableValue(crud.context, r)
				}
			}, func(len int) {
				out = make([]BasicDescriptableValuer, len, len)
			})
			return out
		},
	}
}
