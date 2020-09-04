package resource

import (
	"database/sql"
	"errors"
	"reflect"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
	"github.com/ecletus/roles"
	"github.com/moisespsena-go/aorm"
)

// ErrProcessorSkipLeft skip left processors error, if returned this error in validation, before callbacks, then qor will stop process following processors
var ErrProcessorSkipLeft = errors.New("resource: skip left")

type processor struct {
	defaultDenyMode bool
	Result          interface{}
	Resource        Resourcer
	Context         *core.Context
	MetaValues      *MetaValues
	SkipLeft        bool
	newRecord       bool
	notLoad         bool
}

// DecodeToResource decode meta values to resource result
func DecodeToResource(res Resourcer, result interface{}, metaValues *MetaValues, context *core.Context, notLoad ...bool) *processor {
	if !metaValues.IsEmpty() && metaValues.Values[0].Parent.Meta == nil {
		metaValues.Values[0].Parent.Meta = &Meta{Resource: res}
	}
	if len(notLoad) == 0 {
		notLoad = append(notLoad, false)
	}
	p := &processor{
		defaultDenyMode: res.DefaultDenyMode(),
		Resource:        res,
		Result:          result,
		Context:         context,
		MetaValues:      metaValues,
		notLoad:         notLoad[0],
	}
	if res.GetModelStruct().Parent == nil {
		p.newRecord = aorm.ZeroIdOf(result)
	} else {
		p.notLoad = true
	}
	return p
}

func (processor *processor) checkSkipLeft(errs ...error) bool {
	if processor.SkipLeft {
		return true
	}

	for _, err := range errs {
		if err == ErrProcessorSkipLeft {
			processor.SkipLeft = true
			break
		}
	}
	return processor.SkipLeft
}

func (processor *processor) Initialize() error {
	if !processor.notLoad {
		if processor.Resource.GetModelStruct().Parent == nil && processor.Resource.HasKey() {
			err := processor.Resource.Crud(processor.Context).SetMetaValues(processor.MetaValues).FindOne(processor.Result)
			if !aorm.IsRecordNotFoundError(err) {
				processor.checkSkipLeft(err)
				return err
			}
		}
	}
	return nil
}

func (processor *processor) Validate() error {
	if processor.checkSkipLeft() {
		return nil
	}
	var errors core.Errors
	processor.Resource.GetResource().Validate(processor.Result, processor.MetaValues, processor.Context, func(err error) (stop bool) {
		errors.AddError(err)
		return processor.checkSkipLeft(err)
	})
	return errors
}

func (processor *processor) decode() (errors []error) {
	if processor.checkSkipLeft() || processor.MetaValues == nil {
		return
	}

	if destroy := processor.MetaValues.Get("_destroy"); destroy != nil {
		return
	}

	var reqCheck = processor.MetaValues.IsRequirementCheck()

	for _, metaValue := range processor.MetaValues.Values {
		meta := metaValue.Meta
		if meta == nil {
			continue
		}

		defer processor.Context.MetaTreeStack.WithNamer(func() string {
			return meta.GetRecordLabelC(processor.Context, processor.Result)
		}, meta.GetName())()

		if processor.newRecord && !meta.HasPermission(roles.Create, processor.Context).Ok(!processor.defaultDenyMode) {
			continue
		} else if !meta.HasPermission(roles.Update, processor.Context).Ok(!processor.defaultDenyMode) {
			continue
		}

		if metaValue.MetaValues != nil && len(metaValue.MetaValues.Values) > 0 {
			if !meta.IsRequired() && metaValue.MetaValues.IsBlank() {
				continue
			}
			if res := metaValue.Meta.GetResource(); res != nil && !reflect.ValueOf(res).IsNil() {
				field := reflect.Indirect(reflect.ValueOf(processor.Result)).FieldByName(meta.GetFieldName())
				if utils.ModelType(field.Addr().Interface()) == utils.ModelType(res.NewStruct(processor.Context.Site)) {
					if _, ok := field.Addr().Interface().(sql.Scanner); !ok {
						err := decodeMetaValuesToField(res, field, metaValue, processor.Context)
						if err != nil {
							errors = append(errors, err)
						}
						continue
					}
				}
			}
		}

		if setter := meta.GetSetter(); setter != nil {
			err := setter(processor.Result, metaValue, processor.Context)
			if err != nil {
				errors = append(errors, err)
			} else if reqCheck && meta.IsRequired() && meta.IsZero(processor.Result, meta.GetValuer()(processor.Result, processor.Context)) {
				errors = append(errors, ErrCantBeBlank(processor.Context, processor.Result, meta.GetName(), meta.GetRecordLabelC(processor.Context, processor.Result)))
			}
		}
	}

	return
}

func (processor *processor) Commit() error {
	var errors core.Errors
	errors.AddError(processor.decode()...)

	if processor.checkSkipLeft(errors.GetErrors()...) {
		return nil
	}

	if errors.HasError() {
		return errors
	}

	for _, fc := range processor.Resource.GetResource().Processors {
		if err := fc(processor.Result, processor.MetaValues, processor.Context); err != nil {
			if processor.checkSkipLeft(err) {
				break
			}
			errors.AddError(err)
		}
	}
	return errors
}

func (processor *processor) Start() error {
	var errors core.Errors
	processor.Initialize()
	if errors.AddError(processor.Validate()); !errors.HasError() {
		errors.AddError(processor.Commit())
	}
	if errors.HasError() {
		return errors
	}
	return nil
}
