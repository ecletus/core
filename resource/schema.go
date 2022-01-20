package resource

import (
	"strings"

	"github.com/ecletus/core"
)

const DefaultFormInputPrefix = "QorResource"

// Decode decode context to result according to resource definition
func Decode(context *core.Context, result interface{}, res Resourcer, f ...ProcessorFlag) (err error) {
	var metaValues *MetaValues

	if !res.IsSingleton() {
		if parent := res.GetParentResource(); parent != nil {
			res.GetParentRelation().SetRelatedID(result, context.ParentResourceID[parent.GetPathLevel()])
		}
	}

	var (
		metaors = res.GetContextMetas(context)
		prefix  = context.FormOptions.InputPrefix
	)
	if prefix == "" {
		prefix = DefaultFormInputPrefix
	}

	if strings.Contains(context.Request.Header.Get("Content-Type"), "json") {
		metaValues, err = ConvertJSONToMetaValues(context, context.Request.Body, metaors)
		context.Request.Body.Close()
	} else {
		metaValues, err = ConvertFormToMetaValues(context, context.Request, metaors, prefix+".")
	}

	if err != nil {
		return
	}

	var errors core.Errors
	if len(metaors) > 0 && len(metaValues.Values) == 0 {
		for _, metaor := range metaors {
			if metaor.IsRequired() {
				errors.AddError(ErrMetaCantBeBlank(context, metaor))
			}
		}
		if errors.HasError() {
			return errors
		}
	}
	processor := DecodeToResource(res, result, &MetaValue{Name: prefix, MetaValues: metaValues}, context, f...)
	errors.AddError(processor.Start())

	if errors.HasError() {
		return errors
	}
	return nil
}

func DecodeMetaValues(context *core.Context, result interface{}, res Resourcer, prefix string, metaValues *MetaValues, f ...ProcessorFlag) (err error) {
	var errors core.Errors
	processor := DecodeToResource(res, result, &MetaValue{Name: prefix, MetaValues: metaValues}, context, f...)
	errors.AddError(processor.Start())

	if errors.HasError() {
		return errors
	}
	return nil
}
