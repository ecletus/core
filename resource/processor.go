package resource

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/ecletus/roles"

	"github.com/moisespsena-go/aorm"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
)

// ErrProcessorSkipLeft skip left processors error, if returned this error in validation, before callbacks, then qor will stop process following processors
var ErrProcessorSkipLeft = errors.New("resource: skip left")

type ProcessorFlag uint16

const (
	ProcNone ProcessorFlag = 1 << iota
	ProcSkipLeft
	ProcSkipRequireCheck
	ProcSkipProcessors
	ProcSkipValidations
	ProcSkipPermissions
	ProcSkipLoad
	ProcSkipChildLoad
	ProcMerge
)

func (b ProcessorFlag) Set(flag ProcessorFlag) ProcessorFlag    { return b | flag }
func (b ProcessorFlag) Clear(flag ProcessorFlag) ProcessorFlag  { return b &^ flag }
func (b ProcessorFlag) Toggle(flag ProcessorFlag) ProcessorFlag { return b ^ flag }
func (b ProcessorFlag) Has(flag ProcessorFlag) bool             { return b&flag != 0 }

type Processor struct {
	defaultDenyMode bool
	Result          interface{}
	Resource        Resourcer
	Context         *core.Context
	MetaValue       *MetaValue
	MetaValues      *MetaValues
	Flag            ProcessorFlag
	newRecord       bool
	reqCheck        bool
	deleted         bool
	parent          *Processor
}

// DecodeToResource decode meta values to resource result
func DecodeToResource(res Resourcer, result interface{}, metaValue *MetaValue, context *core.Context, flag ...ProcessorFlag) *Processor {
	metaValues := metaValue.MetaValues
	if !metaValues.IsEmpty() && metaValues.Values[0].Parent.Meta == nil {
		metaValues.Values[0].Parent.Meta = &Meta{Resource: res}
	}
	var f ProcessorFlag
	for _, flag := range flag {
		f |= flag
	}

	p := &Processor{
		defaultDenyMode: res.DefaultDenyMode(),
		Resource:        res,
		Result:          result,
		Context:         context.MetaContextFactory(context, res, result),
		MetaValue:       metaValue,
		MetaValues:      metaValues,
		Flag:            f,
		reqCheck:        !f.Has(ProcSkipRequireCheck) && metaValues.IsRequirementCheck(),
	}
	if metaValue.Parent != nil {
		p.parent = metaValue.Parent.Processor
	}
	metaValue.Processor = p

	if res.GetModelStruct().Parent == nil {
		p.newRecord = aorm.ZeroIdOf(result)
	}
	return p
}

func (this *Processor) Internal() *Processor {
	this.Flag |= ProcSkipRequireCheck | ProcSkipValidations | ProcSkipProcessors | ProcSkipPermissions
	return this
}

func (this *Processor) Deleted() bool {
	return this.deleted
}

func (this *Processor) checkSkipLeft(errs ...error) bool {
	if this.Flag.Has(ProcSkipLeft) {
		return true
	}

	for _, err := range errs {
		if err == ErrProcessorSkipLeft {
			this.Flag |= ProcSkipLeft
			return true
		}
	}
	return this.Flag.Has(ProcSkipLeft)
}

func (this *Processor) set(metaValue *MetaValue) (err error) {
	meta := metaValue.Meta

	if setter := meta.GetSetter(); setter != nil {
		if err = setter(this.Result, metaValue, this.Context); err != nil {
			return
		}
		if !metaValue.NoBlank && this.reqCheck {
			var required bool
			if requirer := meta.RecordRequirer(); requirer != nil {
				required = requirer(this.Context, this.Result)
			} else {
				required = meta.IsRequired()
			}
			if required {
				if meta.IsZero(this.Result, meta.GetValuer()(this.Result, this.Context)) {
					return ErrCantBeBlank(this.Context, this.Result, meta.GetName(), meta.GetRecordLabelC(this.Context, this.Result))
				}
			}
		}
	}
	return
}

func (this *Processor) Initialize() (err error) {
	if this.Resource.HasKey() {
		var (
			key = this.Resource.GetKey(this.Result)
		)

		if this.MetaValues.GetString("_destroy") == "1" {
			this.deleted = true
		}

		if key.IsZero() {
			if metaValue := this.MetaValues.ByName["id"]; metaValue != nil {
				if metaValue.Meta == nil {
					idS := metaValue.StringValue()
					if idS == "" && this.newRecord && this.deleted {
						this.Flag |= ProcSkipLeft
						return
					}
					if key, err = this.Resource.ParseID(idS); err != nil {
						return
					}
					key.SetTo(this.Result)
				} else {
					if err = this.set(metaValue); err != nil {
						return
					}
					key = this.Resource.GetKey(this.Result)
				}

				if !this.deleted && !this.Flag.Has(ProcSkipLoad) && !this.Resource.GetModelStruct().Dummy {
					crud := this.Resource.Crud(this.Context).SetMetaValues(this.MetaValues)

					if err = crud.FindOne(this.Result, aorm.IDOf(this.Result)); err != nil {
						if aorm.IsRecordNotFoundError(err) {
							err = nil
						} else if this.checkSkipLeft(err) {
							err = nil
						}
					}
				}
			}
			if this.deleted && key.IsZero() {
				return fmt.Errorf("ID value of %s is blank", this.Context.MetaTreeStack.String())
			}
		} else if !this.Flag.Has(ProcSkipLoad) && !this.deleted {
			crud := this.Resource.Crud(this.Context).SetMetaValues(this.MetaValues)

			if err = crud.FindOne(this.Result, key); err != nil {
				if aorm.IsRecordNotFoundError(err) {
					err = nil
				} else if this.checkSkipLeft(err) {
					err = nil
				}
			}
		}

		if this.deleted {
			this.Flag |= ProcSkipLeft
			if this.parent != nil {
				SliceMetaAppendDeleted(reflect.ValueOf(this.parent.Result), this.MetaValue.Parent.Name, key)
			}
			this.Context.DecoderExcludes.Add(key, this.MetaValue.Path(), &ExcludeData{Res: this.Resource})
		}
	}
	return
}

func (this *Processor) Validate() error {
	if this.checkSkipLeft() || this.Flag.Has(ProcSkipValidations) {
		return nil
	}
	var errors core.Errors
	this.Resource.GetResource().Validate(this.Result, this.MetaValues, this.Context, func(err error) (stop bool) {
		errors.AddError(err)
		return this.checkSkipLeft(err)
	})
	if errors.HasError() {
		return errors
	}
	return nil
}

func (this *Processor) decode() (errors []error) {
	if this.checkSkipLeft() || this.MetaValues == nil {
		return
	}

	if bc, ok := this.Resource.(ResourcerMetaValuesBeforeCommiter); ok {
		bc.BeforeCommitMetaValues(this.Context, this.Result, this.MetaValues)
	}

	if bc, ok := this.Result.(ResourceResultMetaValuesBeforeCommiter); ok {
		bc.BeforeCommitMetaValues(this.Context, this.Resource, this.MetaValues)
	}

	var (
		reqCheck  = !this.Flag.Has(ProcSkipRequireCheck) && this.MetaValues.IsRequirementCheck()
		ctx       = this.Context
		available = func(metaValue *MetaValue) bool {
			meta := metaValue.Meta
			if meta == nil {
				return false
			}
			if !this.Flag.Has(ProcSkipPermissions) {
				var perm roles.Perm

				if this.newRecord {
					perm = meta.HasPermission(roles.Create, ctx)
				} else {
					perm = meta.HasPermission(roles.Update, ctx)
				}

				if perm == roles.UNDEF {
					if metaValue.Parent.Meta != nil {
						perm = metaValue.Parent.Meta.HasPermission(roles.Update, ctx)
					}
				}

				if perm == roles.UNDEF {
					return true
				}
				return perm.Allow()
			}
			return true
		}

		set = func(metaValue *MetaValue) {
			meta := metaValue.Meta

			if setter := meta.GetSetter(); setter != nil {
				err := setter(this.Result, metaValue, this.Context)
				if err != nil {
					errors = append(errors, err)
				} else if !metaValue.NoBlank && reqCheck {
					var required bool
					if requirer := meta.RecordRequirer(); requirer != nil {
						required = requirer(this.Context, this.Result)
					} else {
						required = meta.IsRequired()
					}
					if required {
						if meta.IsZero(this.Result, meta.GetValuer()(this.Result, this.Context)) {
							errors = append(errors, ErrCantBeBlank(this.Context, this.Result, meta.GetName(), meta.GetRecordLabelC(this.Context, this.Result)))
						}
					}
				}
			}
		}
	)

	if metaValue := this.MetaValues.ByName["id"]; metaValue != nil {
		if available(metaValue) {
			func() {
				meta := metaValue.Meta

				defer ctx.MetaTreeStack.WithNamer(func() string {
					return meta.GetRecordLabelC(ctx, this.Result)
				}, meta.GetName())()

				set(metaValue)

				if len(errors) != 0 {
					return
				}

				crud := this.Resource.Crud(this.Context).SetMetaValues(this.MetaValues)
				if id := aorm.IDOf(this.Result); !id.IsZero() {
					if err := crud.FindOne(this.Result); err != nil {
						if aorm.IsRecordNotFoundError(err) {
							err = nil
						} else {
							this.checkSkipLeft(err)
							errors = append(errors, err)
						}
					}
				}
			}()
		}
	}

	if len(errors) > 0 {
		return
	}

	var subMetaValues []*MetaValue

	for _, metaValue := range this.MetaValues.Values {
		metaValue.Processor = this

		if metaValue.Name == "id" {
			continue
		}

		if !available(metaValue) {
			continue
		}

		if metaValue.MetaValues != nil {
			subMetaValues = append(subMetaValues, metaValue)
			continue
		}

		func() {
			meta := metaValue.Meta

			defer ctx.MetaTreeStack.WithNamer(func() string {
				return meta.GetRecordLabelC(ctx, this.Result)
			}, meta.GetName())()

			set(metaValue)
		}()
	}

	for _, metaValue := range subMetaValues {
		func() {
			meta := metaValue.Meta

			defer ctx.MetaTreeStack.WithNamer(func() string {
				return meta.GetRecordLabelC(ctx, this.Result)
			}, meta.GetName())()

			if metaValue.MetaValues.Disabled {
				field := reflect.Indirect(reflect.ValueOf(this.Result)).FieldByName(meta.GetFieldName())
				field.Set(reflect.Zero(field.Type()))
				return
			} else if len(metaValue.MetaValues.Values) > 0 {
				if !meta.IsRequired() && metaValue.MetaValues.IsBlank() {
					return
				}
				if res := metaValue.Meta.GetResource(); res != nil && !reflect.ValueOf(res).IsNil() {
					field := meta.GetReflectStructValueOrInstantiate(reflect.Indirect(reflect.ValueOf(this.Result)))
					if utils.IndirectItemType(field.Type()) == utils.ModelType(res.GetValue()) {
						if _, ok := field.Addr().Interface().(sql.Scanner); !ok {
							err := decodeMetaValuesToField(res, this.Result, field, metaValue, ctx, this.Flag)
							if err != nil {
								errors = append(errors, err)
							}
							return
						}
					}
				}
			}

			set(metaValue)
		}()
	}

	return
}

func (this *Processor) Commit() error {
	var errors core.Errors
	errors.AddError(this.decode()...)

	if this.checkSkipLeft(errors.GetErrors()...) {
		return nil
	}

	if errors.HasError() {
		return errors
	}

	if this.Flag.Has(ProcSkipProcessors) {
		return nil
	}

	for _, fc := range this.Resource.GetResource().Processors {
		if err := fc(this.Result, this.MetaValues, this.Context); err != nil {
			if this.checkSkipLeft(err) {
				break
			}
			errors.AddError(err)
		}
	}

	if errors.HasError() {
		return errors
	}
	return nil
}

func (this *Processor) Start() (err error) {
	if err = this.Initialize(); err != nil {
		return
	}
	if this.Flag.Has(ProcSkipLeft) {
		return nil
	}

	if err = this.Validate(); err != nil {
		return
	}

	if err = this.Commit(); err != nil {
		return
	}
	return
}

type ResourcerMetaValuesBeforeCommiter interface {
	BeforeCommitMetaValues(ctx *core.Context, record interface{}, metaValues *MetaValues)
}

type ResourceResultMetaValuesBeforeCommiter interface {
	BeforeCommitMetaValues(ctx *core.Context, res Resourcer, metaValues *MetaValues)
}

type ExcludeData struct {
	Res  Resourcer
	Path string
}
