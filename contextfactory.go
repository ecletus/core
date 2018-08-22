package core

import (
	"context"
	"net/http"

	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/moisespsena/go-route"
	"github.com/aghape/core/config"
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

	URL := route.GetOriginalUrl(req)
	if URL == nil {
		urlCopy := *req.URL
		URL = &urlCopy
	}

	if parent == nil {
		ctx = &Context{
			ContextFactory: cf,
			Config:         config.NewConfig(),
			Request:        req,
			OriginalURL:    URL,
			Prefix:         stringOrDefault(rctx.Value("PREFIX")),
			StaticURL:      stringOrDefault(rctx.Value("STATIC_URL")),
			Translator:     cf.Translator,
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
		req, ctx.RouteContext = route.GetOrNewRouteContextForRequest(req)
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

func (cf *ContextFactory) GetCleanSkipPrefixFromRequest(r *http.Request) (string, *http.Request) {
	if v, ok := r.Context().Value(PREFIX + ".skip_prefix").(string); ok && v != "" {
		r = cf.SetSkipPrefixToRequest(r, "")
		return v, r
	}
	return "", r
}

func (cf *ContextFactory) SetSkipPrefixToRequest(r *http.Request, prefix string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), PREFIX+".skip_prefix", prefix))
}

func (cf *ContextFactory) NewContextFromChain(chain *route.ChainHandler) (*http.Request, *Context) {
	prefix, r := cf.GetCleanSkipPrefixFromRequest(chain.Request())
	chain.SetRequest(r)
	return cf.NewContextFromRequestPair(chain.Writer, chain.Request(), prefix)
}
