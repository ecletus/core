package core

import (
	"context"
	"net/http"

	"github.com/ecletus/core/config"
	"github.com/moisespsena-go/xroute"
	"github.com/moisespsena-go/i18n-modular/i18nmod"
)

type ContextFactory struct {
	Translator *i18nmod.Translator
}

func NewContextFactory(translator *i18nmod.Translator) *ContextFactory {
	return &ContextFactory{translator}
}

func (cf *ContextFactory) NewContextForRequest(req *http.Request, prefix ...string) (*http.Request, *Context) {
	rctx := req.Context()
	parent := ContextFromRequest(req)

	var ctx *Context

	URL := *req.URL
	URL.Path = req.RequestURI

	if parent == nil {
		ctx = &Context{
			ContextFactory: cf,
			Config:         config.NewConfig(),
			Request:        req,
			OriginalURL:    &URL,
			StaticURL:      stringOrDefault(rctx.Value("STATIC_URL")),
			Translator:     cf.Translator,
			DefaultLocale:  cf.Translator.DefaultLocale,
		}
		ctx.AsTop()
		if len(prefix) > 0 && prefix[0] != "" {
			req, ctx = ctx.NewChild(req, prefix...)
		}
	} else {
		req, ctx = parent.NewChild(req, prefix...)
	}

	req = req.WithContext(context.WithValue(req.Context(), CONTEXT_KEY, ctx))
	if ctx.RouteContext == nil {
		req, ctx.RouteContext = xroute.GetOrNewRouteContextForRequest(req)
	}
	ctx.Request = req
	return req, ctx
}

func (cf *ContextFactory) NewContextFromRequestPair(w http.ResponseWriter, r *http.Request, prefix ...string) (*http.Request, *Context) {
	r, ctx := cf.NewContextForRequest(r, prefix...)
	ctx.Writer = w
	return r, ctx
}

func (cf *ContextFactory) GetOrNewContextFromRequestPair(w http.ResponseWriter, r *http.Request) (*http.Request, *Context) {
	ctx := ContextFromRequest(r)
	if ctx == nil {
		r, ctx = cf.NewContextFromRequestPair(w, r)
	}
	return r, ctx
}
