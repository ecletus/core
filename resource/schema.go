package resource

import (
	"strings"

	"github.com/ecletus/core"
)

// Decode decode context to result according to resource definition
func Decode(context *core.Context, result interface{}, res Resourcer, notLoad ...bool) (err error) {
	var errors core.Errors
	var metaValues *MetaValues

	if !res.IsSingleton() {
		if parent := res.GetParentResource(); parent != nil {
			res.GetParentRelation().SetRelatedID(result, context.ParentResourceID[parent.GetPathLevel()])
		}
	}

	metaors := res.GetContextMetas(context)

	if strings.Contains(context.Request.Header.Get("Content-Type"), "json") {
		metaValues, err = ConvertJSONToMetaValues(context, context.Request.Body, metaors)
		context.Request.Body.Close()
	} else {
		metaValues, err = ConvertFormToMetaValues(context, context.Request, metaors, "QorResource.")
	}

	errors.AddError(err)
	processor := DecodeToResource(res, result, metaValues, context, notLoad...)
	errors.AddError(processor.Start())

	if errors.HasError() {
		return errors
	}
	return nil
}
