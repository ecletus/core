package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-errors/errors"
	"github.com/moisespsena-go/i18n-modular/i18nmod"

	errwrap "github.com/moisespsena-go/error-wrap"
)

// Errors is a struct that used to hold errors array
type Errors []error

func NewErrors(e ...error) (errors *Errors) {
	errors = &Errors{}
	errors.AddError(e...)
	return
}

func (errs Errors) Len() int {
	return len(errs)
}

// error get formatted error message
func (errs Errors) Error() string {
	return errs.String()
}

// AddError add error to Errors struct
func (errs *Errors) AddError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			if e, ok := err.(errorsInterface); ok {
				*errs = append(*errs, e.GetErrors()...)
			} else {
				*errs = append(*errs, err)
			}
		}
	}
	if len(*errs) > 0 {
		return errs
	}
	return nil
}

// HasError return has error or not
func (errs Errors) HasError() bool {
	return len(errs) != 0
}

// GetErrors return error array
func (errs Errors) GetErrors() []error {
	return errs
}

// ExcludeError exclude errors when match matcher and returns new Errors with not mached errors
func (errs Errors) Filter(matcher func(err error) error) Errors {
	var newErrors Errors
	for _, err := range errs {
		if filtered := matcher(err); filtered != nil {
			newErrors = append(newErrors, filtered)
		}
	}
	return newErrors
}

func (errs Errors) String() string {
	var strs = make([]string, len(errs))
	for i, err := range errs {

		strs[i] = StringifyError(err)
	}
	return strings.Join(strs, "\n -")
}

func (errs Errors) GetErrorsT(ctx i18nmod.Context) (l []error) {
	l = make([]error, len(errs))
	for i, err := range errs {
		switch et := err.(type) {
		case i18nmod.TError:
			l[i] = errors.New(et.Translate(ctx))
		default:
			l[i] = err
		}
	}
	return
}

func (errs Errors) GetErrorsTS(ctx i18nmod.Context) (strs []string) {
	strs = make([]string, len(errs))
	for i, err := range errs {
		strs[i] = StringifyErrorT(ctx, err)
	}
	return
}

func (errs Errors) StringT(ctx i18nmod.Context) string {
	return strings.Join(errs.GetErrorsTS(ctx), "\n -")
}

func (errs *Errors) Reset() {
	errs = nil
}

type errorsInterface interface {
	GetErrors() []error
}

func StringifyErrorT(ctx i18nmod.Context, err error) string {
	switch et := err.(type) {
	case i18nmod.TError:
		return et.Translate(ctx)
	default:
		return StringifyError(err)
	}
}

func StringifyError(err error) string {
	switch et := err.(type) {
	case errwrap.ErrorWrapper:
		sub := fmt.Sprintf("[%s]: %s", errwrap.TypeOf(et.Err()), et.Err())
		et.Prev().EachType(func(typ reflect.Type, err error) error {
			sub += fmt.Sprintf("\n    from [%s]: %s", typ, err)
			return nil
		})
		return sub
	default:
		return err.Error()
	}
}

type err string

func (this err) Translate(ctx i18nmod.Context) string {
	return ctx.T(i18ng + ".errors." + strings.ReplaceAll(string(this), " ", "_")).Get()
}

func (this err) Error() string {
	return string(this)
}

const (
	// ErrCantBeBlank cant blank field
	ErrCantBeBlank      err = "cant be blank"
	ErrCantBeBlankOneOf err = "cant be blank one of"
)
