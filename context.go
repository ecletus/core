package core

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/moisespsena/go-default-logger"
	"github.com/op/go-logging"

	"github.com/aghape/common"
	"github.com/aghape/core/config"
	"github.com/aghape/core/contextdata"
	"github.com/aghape/core/utils/uri"
	uurl "github.com/aghape/core/utils/url"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena-go/xroute"
	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/moisespsena/template/html/template"
)

var CONTEXT_KEY = PREFIX

var (
	DefaultLocale string
)

func init() {
	lang := os.Getenv("LANG")

	if len(lang) >= 5 {
		DefaultLocale = strings.Replace(strings.Split(lang, ".")[0], "_", "-", 1)
	}
}

type i18nGroup struct {
	Prev  *i18nGroup
	Value string
}

// Context qor context, which is used for many qor components, used to share information between them
type Context struct {
	Parent           *Context
	currentUser      common.User
	Request          *http.Request
	Writer           http.ResponseWriter
	Roles            []string
	ResourceID       string
	ParentResourceID []string
	Errors

	DB             *aorm.DB
	Config         *config.Config
	I18nContext    i18nmod.Context
	DefaultLocale  string
	Locale         string
	Prefix         string
	StaticURL      string
	data           *contextdata.ContextData
	top            *Context
	isTop          bool
	Site           SiteInterface
	OriginalURL    *url.URL
	breadcrumbs    *Breadcrumbs
	RouteContext   *xroute.RouteContext
	Translator     *i18nmod.Translator
	ContextFactory *ContextFactory
	I18nGroupStack *i18nGroup
	NotFound       bool
	Api            bool

	logger     *logging.Logger
	RedirectTo string
}

func (context *Context) Read(p []byte) (n int, err error) {
	return context.Request.Body.Read(p)
}

func (context *Context) Write(p []byte) (n int, err error) {
	return context.Writer.Write(p)
}

func (context *Context) CurrentUser() common.User {
	if context.currentUser == nil && context.Parent != nil {
		c := context.Parent
		for c != nil && c.currentUser == nil {
			c = c.Parent
		}
		if c != nil {
			context.currentUser = c.currentUser
		}
	}
	return context.currentUser
}

func (context *Context) SetCurrentUser(user common.User) {
	if user != nil {
		if defaultLocale := user.GetDefaultLocale(); defaultLocale != "" {
			for _, locale := range context.Translator.Locales {
				if locale == defaultLocale {
					context.DefaultLocale = locale
					break
				}
			}
		}
	}
	context.currentUser = user
}

func (context *Context) Breadcrumbs() *Breadcrumbs {
	if context.breadcrumbs == nil {
		context.breadcrumbs = &Breadcrumbs{}
	}
	return context.breadcrumbs
}

func (context *Context) PushI18nGroup(group string) func() {
	context.I18nGroupStack = &i18nGroup{context.I18nGroupStack, group}
	return context.PopI18nGroup
}

func (context *Context) PopI18nGroup() {
	context.I18nGroupStack = context.I18nGroupStack.Prev
}

func (context *Context) URLParams() *xroute.OrderedMap {
	return context.RouteContext.URLParams
}

func (context *Context) URLParam(key string) string {
	if context.RouteContext != nil {
		return context.RouteContext.URLParam(key)
	}
	return ""
}

func (context *Context) GetFormOrQuery(key string) (value string) {
	if context.Request.Form != nil {
		if value = context.Request.Form.Get(key); value != "" {
			return
		}
	}
	value = context.Request.URL.Query().Get(key)
	return
}

func (context *Context) SetRequest(r *http.Request) {
	context.Request = r
}

func (context *Context) Data() *contextdata.ContextData {
	if context.data == nil {
		if context.Request != nil {
			context.Request, context.data = contextdata.GetOrSetRequestContextData(context.Request)
		} else {
			context.data = contextdata.NewRequestContextData()
		}
	}
	return context.data
}

func (context *Context) Path() string {
	return context.Request.URL.Path
}

func (context *Context) GlobalPath() string {
	return context.GenGlobalURL(context.Request.URL.Path)
}

// PatchCurrentURL is a convinent wrapper for qor/utils.PatchURL
func (context *Context) PatchCurrentURL(params ...interface{}) (patchedURL string, err error) {
	return uurl.PatchURL(context.OriginalURL.String(), params...)
}

// PatchURL is a convinent wrapper for qor/utils.PatchURL
func (context *Context) PatchURL(url string, params ...interface{}) (patchedURL string, err error) {
	return uurl.PatchURL(url, params...)
}

// JoinCurrentURL is a convinent wrapper for qor/utils.JoinURL
func (context *Context) JoinCurrentURL(params ...interface{}) (joinedURL string, err error) {
	return uurl.JoinURL(context.OriginalURL.String(), params...)
}

// JoinURL is a convinent wrapper for qor/utils.JoinURL
func (context *Context) JoinURL(url string, params ...interface{}) (joinedURL string, err error) {
	return uurl.JoinURL(url, params...)
}

func (context *Context) GetLocale() string {
	if context.Locale != "" {
		return context.Locale
	}

	var locale string

	if locale = context.Request.Header.Get("Locale"); locale == "" {
		locale = context.Request.URL.Query().Get("locale")
		if locale == "" {
			if local, err := context.Request.Cookie("locale"); err == nil {
				locale = local.Value
			}
		} else if context.Writer != nil {
			context.Request.Header.Set("Locale", locale)
			cookie := http.Cookie{Name: "locale", Value: locale, Expires: time.Now().AddDate(1, 0, 0)}
			cookie.HttpOnly = true

			// set https cookie
			if context.Request != nil && context.Request.URL.Scheme == "https" {
				cookie.Secure = true
			}

			// set default path
			if cookie.Path == "" {
				cookie.Path = "/"
			}

			http.SetCookie(context.Writer, &cookie)
		}
	}

	locale = context.Translator.ValidOrDefaultLocale(locale)
	context.Locale = locale

	return locale
}

func (context *Context) GetI18nContext() i18nmod.Context {
	if context.I18nContext == nil {
		locale := context.GetLocale()
		context.I18nContext = context.Translator.NewContext(locale)
	}
	return context.I18nContext
}

func (context *Context) I18nT(key string) *i18nmod.T {
	return context.GetI18nContext().T(key)
}

func (context *Context) T(key string, defaul ...interface{}) template.HTML {
	return template.HTML(context.Ts(key, defaul...))
}

func (context *Context) TT(key string, data interface{}, defaul ...interface{}) template.HTML {
	return template.HTML(context.TTs(key, data, defaul...))
}

func (context *Context) Ts(key string, defaul ...interface{}) string {
	if key[0] == '.' {
		key = context.I18nGroupStack.Value + key
	}
	return context.I18nT(key).DefaultArgs(defaul...).Get()
}

func (context *Context) TTs(key string, data interface{}, defaul ...interface{}) string {
	if key[0] == '.' {
		key = context.I18nGroupStack.Value + key
	}
	return context.GetI18nContext().TT(key).DefaultArgs(defaul...).Data(data).Get()
}

// Clone clone current context
func (context *Context) Clone() *Context {
	var clone = *context
	return &clone
}

// Clone clone current context
func (context *Context) CloneBasic() *Context {
	return &Context{
		Request:       context.Request,
		Writer:        context.Writer,
		StaticURL:     context.StaticURL,
		Prefix:        context.Prefix,
		OriginalURL:   context.OriginalURL,
		data:          context.data,
		DB:            context.DB,
		Site:          context.Site,
		Parent:        context.Parent,
		currentUser:   context.currentUser,
		Locale:        context.Locale,
		I18nContext:   context.I18nContext,
		Translator:    context.Translator,
		NotFound:      context.NotFound,
		Api:           context.Api,
		logger:        context.logger,
		DefaultLocale: context.DefaultLocale,
	}
}

func (context *Context) Top() *Context {
	if context.top != nil {
		return context.top
	}
	if context.Parent != nil && !context.isTop {
		context.top = context.Parent.Top()
		return context.top
	}
	return context
}

func (context *Context) AsTop() *Context {
	context.isTop = true
	return context
}

func (context *Context) IsTop() bool {
	return context.isTop || context.Parent == nil
}

func (context *Context) GetStaticURL() string {
	prefix := context.StaticURL

	if prefix == "" {
		v2 := context.Request.Context().Value("STATIC_URL")
		if v3, ok := v2.(string); ok {
			context.StaticURL = v3
			prefix = v3
		}
	}
	return prefix
}

func (context *Context) GenStaticURL(path ...string) string {
	path = uri.Clean(path)
	prefix := context.GetStaticURL()

	if len(path) == 0 {
		return prefix
	}

	// is external url
	if strings.HasPrefix(path[0], "//") {
		return uri.Join(path...)
	}

	if path[0][0:1] == "/" {
		pth := prefix + path[0]
		return pth
	}

	return uri.Join(append([]string{prefix}, path...)...)
}

func (context *Context) GenURL(path ...string) string {
	path = uri.Clean(path)
	prefix := context.Prefix

	if prefix == "" {
		v2 := context.Request.Context().Value("PREFIX")
		if v3, ok := v2.(string); ok {
			context.Prefix = v3
			prefix = v3
		}
	}

	if len(path) == 0 || path[0] == "" {
		return prefix
	}

	return uri.Join(append([]string{prefix}, path...)...)
}

func (context *Context) GenGlobalStaticURL(path ...string) string {
	return context.Top().GenStaticURL(path...)
}

func (context *Context) GenGlobalURL(path ...string) string {
	return context.Top().GenURL(path...)
}

func (context *Context) JoinPath(path ...string) string {
	if len(path) == 0 {
		return ""
	}

	var parts []string
	if context.Prefix != "" {
		parts = append(parts, context.Prefix)
	}
	return uri.Join(append(parts, path...)...)
}

func (context *Context) Root() *Context {
	c := context
	for c.Parent != nil {
		c = c.Parent
	}
	return c
}

func (context *Context) NewChild(r *http.Request, prefix ...string) (*http.Request, *Context) {
	var path string
	if len(prefix) > 0 && prefix[0] != "" {
		path = "/" + strings.Trim(prefix[0], "/")
	}

	child := context.Clone()
	child.isTop = false
	child.Parent = context

	if path != "" {
		child.StaticURL += path
		child.Prefix += path
		if r == nil {
			r = context.Request
		}
		if r != nil {
			nurl := *r.URL
			nurl2 := &nurl
			nurl2.Path = strings.TrimPrefix(r.URL.Path, path)
			var err error
			r.URL, err = url.Parse(nurl2.String())
			if err != nil {
				panic(err)
			}
		}
	} else if r != nil {
		child.Request = r
	}
	return r, child
}

func (context *Context) GetDB() *aorm.DB {
	return context.DB
}

func (context *Context) SetDB(db *aorm.DB) {
	if db != nil {
		db = db.Set(CONTEXT_KEY, context)
	}
	context.DB = db
}

func (context *Context) Logger() *logging.Logger {
	if context.logger == nil {
		context.logger = defaultlogger.NewLogger(context.OriginalURL.String())
	}
	return context.logger
}

func (context *Context) Htmlify(value interface{}) template.HTML {
	switch vt := value.(type) {
	case interface{ Htmlify() template.HTML }:
		return vt.Htmlify()
	case interface{ Htmlify(*Context) template.HTML }:
		return vt.Htmlify(context)
	default:
		return template.HTML(fmt.Sprint(vt))
	}
}

func stringOrDefault(value interface{}, defaul ...string) string {
	if str, ok := value.(string); ok {
		return str
	}
	if len(defaul) > 0 {
		return defaul[0]
	}
	return ""
}

func ContextFromRequest(req *http.Request) (ctx *Context) {
	v := req.Context().Value(CONTEXT_KEY)
	if v != nil {
		ctx, _ = v.(*Context)
	}
	return
}

func ContextFromDB(db *aorm.DB) *Context {
	v, _ := db.Get(CONTEXT_KEY)
	if v == nil {
		return nil
	}
	return v.(*Context)
}

func ContexFromRouteContext(rctx *xroute.RouteContext) *Context {
	v, ok := rctx.Data[CONTEXT_KEY]
	if ok {
		return v.(*Context)
	}
	return nil
}

func ContexFromChain(chain *xroute.ChainHandler) *Context {
	return ContexFromRouteContext(chain.Context)
}
