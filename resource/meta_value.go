package resource

import (
	"reflect"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/ecletus/validations"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

// MetaValues is slice of MetaValue
type MetaValues struct {
	Values []*MetaValue
	ByName map[string]int
}

func (this *MetaValues) Reset() {
	this.Values = nil
	this.ByName = map[string]int{}
}

func (mvs *MetaValues) IsEmpty() bool {
	return mvs == nil || len(mvs.Values) == 0
}

func (mvs *MetaValues) IsBlank() bool {
	if mvs.IsEmpty() {
		return true
	}
	for _, v := range mvs.Values {
		if v.MetaValues != nil {
			if !v.MetaValues.IsEmpty() {
				return false
			}
		} else if v.NoBlank {
			return false
		} else if v.Value != nil {
			switch t := v.Value.(type) {
			case []string:
				for _, v := range t {
					if v != "" {
						return false
					}
				}
			default:
				if !aorm.IsBlank(reflect.ValueOf(v.Value)) {
					return false
				}
			}
		}
	}
	return true
}

// Get get meta value from MetaValues with name
func (mvs MetaValues) Get(name string) *MetaValue {
	for _, mv := range mvs.Values {
		if mv.Name == name {
			return mv
		}
	}

	return nil
}

// Get get meta value from MetaValues with name
func (mvs MetaValues) GetString(name string) string {
	for _, mv := range mvs.Values {
		if mv.Name == name {
			return mv.FirstStringValue()
		}
	}

	return ""
}

func (this *MetaValues) IsRequirementCheck() bool {
	if len(this.Values) == 1 && this.Values[0].Meta.IsAlone() {
		return false
	}
	for _, v := range this.Values {
		if v.Meta != nil {
			if cd := v.Meta.IsSiblingsRequirementCheckDisabled(); cd.OnTrue() {
				s := v.FirstStringValue()
				return s == "true" || s == "on"
			} else if cd.OnFalse() {
				s := v.FirstStringValue()
				return s == "false" || s == "no"
			}
		}
	}
	return true
}

func (this *MetaValues) CheckRequirement(context *core.Context, metaors ...Metaor) error {
	if this.IsRequirementCheck() {
		errors := core.Errors{}
		for _, metaor := range metaors {
			name := metaor.GetName()
			if _, ok := this.ByName[name]; !ok && metaor.IsRequired() {
				errors.AddError(ErrMetaCantBeBlank(context, metaor))
			}
		}
		if errors.HasError() {
			return errors
		}
	}
	return nil
}

// MetaValue a struct used to hold information when convert inputs from HTTP form, JSON, CSV files and so on to meta values
// It will includes field name, field value and its configured Meta, if it is a nested resource, will includes nested metas in its MetaValues
type MetaValue struct {
	Parent               *MetaValue
	Name                 string
	Value                interface{}
	Index                int
	MetaValues           *MetaValues
	Meta                 Metaor
	error                error
	NoBlank bool
}

func (this *MetaValue) FirstStringValue() (value string) {
	if this.Value != nil {
		value = this.Value.([]string)[0]
	}
	return
}

func (this *MetaValue) FirstInterfaceValue() (value interface{}) {
	if this.Value != nil {
		value = this.Value.([]interface{})[0]
	}
	return
}

func decodeMetaValuesToField(res Resourcer, field reflect.Value, metaValue *MetaValue, context *core.Context, merge ...bool) (err error) {
	defer func() {
		if err != nil {
			if !validations.IsError(err) {
				err = errors.Wrap(err, "decode meta values")
			}
		}
	}()
	// if field.Kind() == reflect.Struct {
	if metaValue.Meta.IsInline() {
		var notLoad bool
		if field := metaValue.Meta.GetFieldStruct(); field != nil && field.TagSettings["-"] == "-" {
			notLoad = true
		}
		typ := field.Type()
		isPtr := typ.Kind() == reflect.Ptr
		if isPtr {
			typ = typ.Elem()
		}
		var value = field
		if len(merge) > 0 && merge[0] {
			if isPtr {
				value = field.Elem()
			} else {
				value = field.Addr()
			}
		} else {
			newValue := reflect.New(typ)
			if err = copier.Copy(newValue.Interface(), value.Interface()); err != nil {
				return
			}
			value = newValue
		}
		valueInterface := value.Interface()
		associationProcessor := DecodeToResource(res, valueInterface, metaValue.MetaValues, context, notLoad)
		if err = associationProcessor.Start(); err != nil {
			return
		}
		if !associationProcessor.SkipLeft {
			if isPtr {
				field.Set(value)
			} else {
				field.Set(value.Elem())
			}
		}
	} else if field.Kind() == reflect.Slice {
		if metaValue.Index == 0 {
			field.Set(reflect.Zero(field.Type()))
		}

		var fieldType = field.Type().Elem()
		var isPtr bool
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
			isPtr = true
		}

		value := reflect.New(fieldType)
		associationProcessor := DecodeToResource(res, value.Interface(), metaValue.MetaValues, context)
		if err = associationProcessor.Start(); err != nil {
			return
		}
		if !associationProcessor.SkipLeft {
			if !reflect.DeepEqual(reflect.Zero(fieldType).Interface(), value.Elem().Interface()) {
				if isPtr {
					field.Set(reflect.Append(field, value))
				} else {
					field.Set(reflect.Append(field, value.Elem()))
				}
			}
		}
	}
	return
}
