package core

import (
	"net/http"

	"github.com/moisespsena-go/httpu"
)

type ContextHandler interface {
	ContextHandle(ctx *Context)
}

type ContextHandlerFunc func(ctx *Context)

func (this ContextHandlerFunc) ContextHandle(ctx *Context) {
	this(ctx)
}

func HTTPHandlerOf(handler ContextHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ContextHandle(w, r, handler)
	})
}

func ContextHandle(w http.ResponseWriter, r *http.Request, handler ContextHandler) {
	ctx := ContextFromRequest(r)
	ctx.Request = r
	ctx.Writer = httpu.ResponseWriterOf(w)
	handler.ContextHandle(ctx)
}
