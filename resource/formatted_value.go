package resource

import (
	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

type FormattedValue struct {
	Record       interface{}
	IsZeroF      func(record, value interface{}) bool
	NoZero, Zero bool

	Raw interface{}
	// Raws is a slice of raw values
	Raws interface{}

	Value  string
	Values []string

	SafeValue  string
	SafeValues []string

	// Slice if Raw is Slice
	Slice bool

	Severity core.Severity

	Data map[interface{}]interface{}
}

func (v *FormattedValue) IsZero() bool {
	if v.NoZero {
		return false
	}
	if v.Zero {
		return true
	}
	if v.Slice {
		if v.Raws == nil {
			return true
		}
		if v.IsZeroF == nil {
			if z, ok := v.Raws.(aorm.Zeroer); ok {
				return z.IsZero()
			}
			return false
		}
		return v.IsZeroF(v.Record, v.Raws)
	} else if v.Raw == nil {
		return true
	} else if z, ok := v.Raw.(aorm.Zeroer); ok {
		return z.IsZero()
	} else if v.IsZeroF != nil {
		return v.IsZeroF(v.Record, v.Raw)
	}
	return false
}

func (v *FormattedValue) SetNonZero() *FormattedValue {
	v.NoZero = true
	return v
}

func (v *FormattedValue) SetZero() *FormattedValue {
	v.Zero = true
	return v
}

func (v *FormattedValue) GetSafeValue() string {
	if v.SafeValue != "" {
		return v.SafeValue
	}
	return v.Value
}

func (v *FormattedValue) GetSafeValues() []string {
	if len(v.SafeValues) > 0 {
		return v.SafeValues
	}
	return v.Values
}
