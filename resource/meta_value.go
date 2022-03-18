package resource

import (
	"reflect"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/ecletus/validations"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

// MetaValues is slice of MetaValue
type MetaValues struct {
	Disabled bool
	Values   []*MetaValue
	ByName   map[string]*MetaValue
}

func (this *MetaValues) Reset() {
	this.Values = nil
	this.ByName = map[string]*MetaValue{}
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
	if this.Disabled || len(this.Values) == 0 {
		return false
	}

	if this.Values[0].Meta != nil && len(this.Values) == 1 && this.Values[0].Meta.IsAlone() {
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

func (this *MetaValues) Add(metaValue ...*MetaValue) {
	if this.ByName == nil {
		this.ByName = map[string]*MetaValue{}
	}
	for _, metaValue := range metaValue {
		this.ByName[metaValue.Name] = metaValue
		this.Values = append(this.Values, metaValue)
	}
}

func (this *MetaValues) CheckRequirement(context *core.Context, metaors ...Metaor) error {
	if this.IsRequirementCheck() {
		errors := core.Errors{}
		for _, metaor := range metaors {
			if !metaor.Proxier() {
				name := metaor.GetName()
				if _, ok := this.ByName[name]; !ok && metaor.RecordRequirer() == nil && metaor.IsRequired() {
					errors.AddError(ErrMetaCantBeBlank(context, metaor))
				}
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
	Processor  *Processor
	Parent     *MetaValue
	Name       string
	Value      interface{}
	Index      int
	MetaValues *MetaValues
	ReadOnly   bool
	Meta       Metaor
	error      error
	NoBlank    bool
}

func (this *MetaValue) Path() string {
	var s []string
	var el = this
	for el != nil && el.Name != "" {
		s = append(s, el.Name)
		el = el.Parent
	}
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return strings.Join(s, ".")
}

func (this *MetaValue) FirstStringValue() (value string) {
	if this.Value != nil {
		value = this.Value.([]string)[0]
	}
	return
}

func (this *MetaValue) StringValue() string {
	if this.Value != nil {
		switch t := this.Value.(type) {
		case string:
			return t
		case []string:
			if len(t) > 0 {
				return t[len(t)-1]
			}
		}
	}
	return ""
}

func (this *MetaValue) FirstInterfaceValue() (value interface{}) {
	if this.Value != nil {
		value = this.Value.([]interface{})[0]
	}
	return
}

func (this *MetaValue) EachQueryVal(prefix []string, cb func(prefix []string, name string, value interface{})) {
	if this.MetaValues != nil {
		prefix = append(prefix, this.Name)
		for _, v := range this.MetaValues.Values {
			v.EachQueryVal(prefix, cb)
		}
	} else if this.Value != nil {
		switch t := this.Value.(type) {
		case []string:
			for _, v := range t {
				cb(prefix, this.Name, v)
			}
		default:
			cb(prefix, this.Name, t)
		}
	}
}

func decodeMetaValuesToField(res Resourcer, record interface{}, field reflect.Value, metaValue *MetaValue, context *core.Context, flag ProcessorFlag) (err error) {
	defer func() {
		if err != nil {
			if !validations.IsError(err) {
				err = errors.Wrap(err, "decode meta values")
			}
		}
	}()

	// if field.Kind() == reflect.Struct {
	if metaValue.Meta.IsInline() {
		if field := metaValue.Meta.GetFieldStruct(); field != nil && field.TagSettings["-"] == "-" {
			flag |= ProcSkipLoad
		}
		typ := field.Type()
		isPtr := typ.Kind() == reflect.Ptr
		if isPtr {
			typ = typ.Elem()
		}
		var value = field
		if flag.Has(ProcMerge) {
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
		associationProcessor := DecodeToResource(res, valueInterface, metaValue, context.MetaContextFactory(context, res, valueInterface), flag)
		if err = associationProcessor.Start(); err != nil {
			return
		}
		if !associationProcessor.Flag.Has(ProcSkipLeft) {
			if isPtr {
				field.Set(value)
			} else {
				field.Set(value.Elem())
			}
		}
	} else if field.Kind() == reflect.Slice {
		field.Set(reflect.Zero(field.Type()))
		var (
			fieldType = field.Type().Elem()
			isPtr     bool
			deletions uint
		)
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
			isPtr = true
		}

		for _, mv := range metaValue.MetaValues.Values {
			var (
				value                = reflect.New(fieldType)
				associationProcessor = DecodeToResource(res, value.Interface(), mv, context)
			)

			if err = associationProcessor.Start(); err != nil {
				return
			}

			if associationProcessor.deleted {
				deletions++
				continue
			}

			if associationProcessor.Flag.Has(ProcSkipLeft) {
				continue
			}
			if !reflect.DeepEqual(reflect.Zero(fieldType).Interface(), value.Elem().Interface()) {
				if isPtr {
					field.Set(reflect.Append(field, value))
				} else {
					field.Set(reflect.Append(field, value.Elem()))
				}
			}
		}

		if deletions > 0 && field.IsNil() {
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		}
	}
	return
}
