package resource

import (
	"fmt"

	"github.com/ecletus/validations"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
	"github.com/moisespsena-go/aorm"
)

type DuplicateUniqueIndexError struct {
	*aorm.DuplicateUniqueIndexError
	record   interface{}
	resource Resourcer
}

func (d *DuplicateUniqueIndexError) Record() interface{} {
	return d.record
}

func (d *DuplicateUniqueIndexError) Resource() Resourcer {
	return d.resource
}

func (d DuplicateUniqueIndexError) Cause() error {
	return d.DuplicateUniqueIndexError
}

func ErrCantBeBlank(ctx *core.Context, record interface{}, fieldName string, label ...string) error {
	if len(label) == 0 {
		label = append(label, utils.HumanizeString(fieldName))
	}
	return validations.NewError(record, fieldName, fmt.Sprintf(ctx.ErrorTS(core.ErrCantBeBlank), label[0]))
}

func ErrMetaCantBeBlank(context *core.Context, metaor Metaor) error {
	return ErrCantBeBlank(context, metaor.GetBaseResource().GetModelStruct().Value, metaor.GetName(), metaor.GetLabelC(context))
}

func ErrField(ctx *core.Context, record interface{}, fieldName string, label ...string) func(err interface{}) error {
	if len(label) == 0 {
		label = append(label, utils.HumanizeString(fieldName))
	}
	return func(err interface{}) error {
		var msg string
		if e, ok := err.(error); ok {
			msg = ctx.ErrorTS(e)
		} else {
			msg = err.(string)
		}
		return validations.NewError(record, fieldName, "<b>"+label[0]+"</b>: "+msg)
	}
}
