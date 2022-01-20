package resource

import (
	"github.com/ecletus/core"
)

// ContextScanner interface
// Allow to scan value with reflect.StructField arg
type ContextScanner interface {
	// ContextScan scan value
	ContextScan(ctx *core.Context, src interface{}) error
}

// ContextStringsScanner interface
// Allow to scan value with reflect.StructField arg
type ContextStringsScanner interface {
	// ContextScan scan value
	StringsScan(ctx *core.Context, src []string) error
}

// StringsScanner interface
// Allow to scan value with reflect.StructField arg
type StringsScanner interface {
	// ContextScan scan value
	StringsScan(src []string) error
}

type MetaValueScanner interface {
	MetaValueScan(ctx *core.Context, value *MetaValue) (err error)
}
