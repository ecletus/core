package resource

import "github.com/ecletus/core"

// ContextScanner interface
// Allow to scan value with reflect.StructField arg
type ContextScanner interface {
	// ContextScan scan value
	ContextScan(ctx *core.Context, src interface{}) error
}
