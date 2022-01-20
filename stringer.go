package core

type ContextStringer interface {
	ContextString(ctx *Context) string
}
