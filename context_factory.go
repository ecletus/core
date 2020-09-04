package core

import (
	"context"
	"net/http"
	"net/url"

	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/i18n-modular/i18nmod"
	"github.com/moisespsena-go/xroute"
)

type ContextFactory struct {
	Translator *i18nmod.Translator
	afterCreateCallbacks []func(ctx *Context)
}

func NewContextFactory(translator *i18nmod.Translator) *ContextFactory {
	return &ContextFactory{Translator: translator}
}

func (cf *ContextFactory) AfterCreate(f ...func(ctx *Context)) {
	cf.afterCreateCallbacks = append(cf.afterCreateCallbacks, f...)
}

func (cf *ContextFactory) NewSiteContext(site *Site) *Context {
	ctx := site.NewContext()
	ctx.Translator = cf.Translator
	ctx.DefaultLocale = cf.Translator.DefaultLocale
	return ctx
}

func (cf *ContextFactory) NewContextForRequest(req *http.Request, prefix ...string) (*http.Request, *Context) {
	rctx := req.Context()
	parent := ContextFromRequest(req)

	var ctx *Context

	URL, _ := url.ParseRequestURI(req.RequestURI)

	if URL.Scheme = req.URL.Scheme; URL.Scheme != "" {
		URL.Host = req.Host
	}

	if parent == nil {
		ctx = &Context{
			ContextFactory: cf,
			OriginalURL:    URL,
			StaticURL:      stringOrDefault(rctx.Value("STATIC_URL")),
			Translator:     cf.Translator,
			DefaultLocale:  cf.Translator.DefaultLocale,
		}
		ctx.SetRequest(req)
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

	for _, f := range cf.afterCreateCallbacks {
		f(ctx)
	}

	return req, ctx
}

func (cf *ContextFactory) NewContextFromRequestPair(w http.ResponseWriter, r *http.Request, prefix ...string) (*http.Request, *Context) {
	r, ctx := cf.NewContextForRequest(r, prefix...)
	ctx.Writer = httpu.ResponseWriterOf(w)
	return r, ctx
}

func (cf *ContextFactory) GetOrNewContextFromRequestPair(w http.ResponseWriter, r *http.Request) (*http.Request, *Context) {
	ctx := ContextFromRequest(r)
	if ctx == nil {
		r, ctx = cf.NewContextFromRequestPair(w, r)
	}
	return r, ctx
}
