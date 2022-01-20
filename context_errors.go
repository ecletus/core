package core

type FormattedError struct {
	Label  string
	Errors []string
}

func (this *Context) GetFormattedErrors() (formatedErrors []FormattedError) {
	return this.GetFormattedErrorsOf(&this.Errors)
}

func (this *Context) GetFormattedErrorsOf(errs *Errors) (formatedErrors []FormattedError) {
	type labelInterface interface {
		Label() string
	}

	ctx := this.GetI18nContext()

	for _, err := range *errs {
		if labelErr, ok := err.(labelInterface); ok {
			var found bool
			label := labelErr.Label()
			for _, formatedError := range formatedErrors {
				if formatedError.Label == label {
					formatedError.Errors = append(formatedError.Errors, StringifyErrorT(ctx, err))
					found = true
				}
			}
			if !found {
				formatedErrors = append(formatedErrors, FormattedError{Label: label, Errors: []string{StringifyErrorT(ctx, err)}})
			}
		} else {
			formatedErrors = append(formatedErrors, FormattedError{Errors: []string{StringifyErrorT(ctx, err)}})
		}
	}
	return
}

func (this *Context) GetCleanFormattedErrors() (formatedErrors []FormattedError) {
	return this.GetCleanFormattedErrorsOf(&this.Errors)
}

func (this *Context) GetCleanFormattedErrorsOf(errs *Errors) (formatedErrors []FormattedError) {
	formatedErrors = this.GetFormattedErrorsOf(errs)
	*errs = nil
	return
}
