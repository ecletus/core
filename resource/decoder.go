package resource

import (
	"strings"

	"github.com/ecletus/core"
)

type Decoder struct {
	defaultDenyMode bool
	res     Resourcer
	context *core.Context
	notLoad bool
}

func NewDecoder(res Resourcer, context *core.Context) *Decoder {
	return &Decoder{res: res, context: context}
}

func (this *Decoder) NotLoad() bool {
	return this.notLoad
}

func (this Decoder) SetNotLoad(notLoad bool) *Decoder {
	this.notLoad = notLoad
	return &this
}

func (this Decoder) Decode(result interface{}) (err error) {
	var errors core.Errors
	var metaValues *MetaValues

	if !this.res.IsSingleton() {
		if parent := this.res.GetParentResource(); parent != nil {
			if parentRel := this.res.GetParentRelation(); parentRel.GetRelatedID(result).IsZero() {
				parentId := this.context.ParentResourceID[parent.GetPathLevel()]
				parentRel.SetRelatedID(result, parentId)
			}
		}
	}

	metaors := this.res.GetContextMetas(this.context)

	if strings.Contains(this.context.Request.Header.Get("Content-Type"), "json") {
		defer this.context.Request.Body.Close()
		metaValues, err = ConvertJSONToMetaValues(this.context, this.context.Request.Body, metaors)
	} else {
		metaValues, err = ConvertFormToMetaValues(this.context, this.context.Request, metaors, "QorResource.")
	}

	errors.AddError(err)
	processor := DecodeToResource(this.res, result, metaValues, this.context, this.notLoad)
	err = processor.Start()
	errors.AddError(err)

	if errors.HasError() {
		return errors
	}
	return nil
}
