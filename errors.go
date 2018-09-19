package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/moisespsena/go-error-wrap"
)

// Errors is a struct that used to hold errors array
type Errors struct {
	errors []error
}

// Error get formatted error message
func (errs Errors) Error() string {
	var errors []string
	for _, err := range errs.errors {
		errors = append(errors, err.Error())
	}
	return strings.Join(errors, "; ")
}

// AddError add error to Errors struct
func (errs *Errors) AddError(errors ...error) {
	for _, err := range errors {
		if err != nil {
			if e, ok := err.(errorsInterface); ok {
				errs.errors = append(errs.errors, e.GetErrors()...)
			} else {
				errs.errors = append(errs.errors, err)
			}
		}
	}
}

// HasError return has error or not
func (errs Errors) HasError() bool {
	return len(errs.errors) != 0
}

// GetErrors return error array
func (errs Errors) GetErrors() []error {
	return errs.errors
}

func (errs Errors) String() string {
	var strs = make([]string, len(errs.errors))
	for i, err := range errs.errors {
		if ew, ok := err.(errwrap.ErrorWrapper); ok {
			sub := fmt.Sprintf("[%s]: %s", errwrap.TypeOf(ew.Err()), ew.Err())
			ew.Prev().EachType(func(typ reflect.Type, err error) error {
				sub += fmt.Sprintf("\n    from > [%s]: %s", typ, err)
				return nil
			})
			strs[i] = sub
		} else {
			strs[i] = fmt.Sprint(err)
		}
	}
	return strings.Join(strs, "\n -")
}

type errorsInterface interface {
	GetErrors() []error
}
