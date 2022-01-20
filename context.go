package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/ecletus/roles"
	"github.com/ecletus/session"
	"golang.org/x/text/language"

	"github.com/moisespsena-go/httpu"

	"github.com/ecletus/core/site_config"

	defaultlogger "github.com/moisespsena-go/default-logger"
	"github.com/moisespsena-go/logging"

	"github.com/ecletus/common"
	"github.com/moisespsena-go/xroute"

	"github.com/moisespsena-go/i18n-modular/i18nmod"

	"github.com/moisespsena/template/html/template"

	"github.com/ecletus/core/utils/uri"
	uurl "github.com/ecletus/core/utils/url"
	"github.com/moisespsena-go/aorm"
)

var CONTEXT_KEY = PREFIX

type Result struct {
	Messages []string `json:"messages;omitempty"`
	Type     string   `json:"type;omitempty"`
	Code     int      `json:"code;omitempty"`
}

type ContextGetter interface {
	GetContext() *Context
}

type StringSlice []string

func (this StringSlice) Has(v string) bool {
	for _, el := range this {
		if el == v {
			return true
		}
	}
	return false
}

func (this StringSlice) Interfaces() []interface{} {
	result := make([]interface{}, len(this))
	for i, el := range this {
		result[i] = el
	}
	return result
}

type FormOptions struct {
	InputPrefix               string
	DefaultValueFixerDisabled bool
}

// Context qor context, which is used for many qor components, used to share information between them
type Context struct {
	LocalContext
	Errors

	Parent      *Context
	currentUser common.User
	Request     *http.Request
	Writer      httpu.ResponseWriter
	Roles       roles.Roles
	ResourceID  aorm.ID

	ParentResourceID,
	ExcludeResourceID []aorm.ID

	db             *aorm.DB
	Config         *site_config.Config
	I18nContext    i18nmod.Context
	DefaultLocale  string
	Locale         string
	timeLocation   *time.Location
	Prefix         string
	StaticURL      string
	top            *Context
	isTop          bool
	Site           *Site
	OriginalURL    *url.URL
	breadcrumbs    *Breadcrumbs
	RouteContext   *xroute.RouteContext
	Translator     *i18nmod.Translator
	ContextFactory *ContextFactory
	I18nGroupStack *i18nGroup
	Role           *roles.Role
	Lang           string
	LangTag        *language.Tag
	NotFound       bool
	Api            bool

	logger        logging.Logger
	RedirectTo    string
	MetaTreeStack *NameStacker

	DecoderExcludes *DecoderExcludes
	requestTime     time.Time

	MetaContextFactory func(parent *Context, res interface{}, record interface{}) *Context
	FormOptions        FormOptions
}

func NewContext(arg ...*Context) (ctx *Context) {
	for _, arg := range arg {
		ctx = arg
	}
	if ctx == nil {
		ctx = &Context{}
	}
	if ctx.MetaTreeStack == nil {
		ctx.MetaTreeStack = &NameStacker{}
	}
	return
}

func (this *Context) RequestTime() time.Time {
	return this.requestTime.In(this.TimeLocation())
}

func (this *Context) SetRequestTime(requestTime time.Time) {
	this.requestTime = requestTime
}

func (this *Context) Now() time.Time {
	return time.Now().In(this.TimeLocation())
}

func (this *Context) TimeLocation() *time.Location {
	if this.timeLocation == nil && this.Request != nil {
		if user := this.CurrentUser(); user != nil {
			this.timeLocation = user.GetTimeLocation()
		}
		if this.timeLocation == nil {
			if htl := this.Request.Header.Get("X-Time-Location"); htl != "" {
				this.timeLocation, _ = time.LoadLocation(htl)
			}
		}
		if this.timeLocation == nil {
			if this.Site != nil {
				this.timeLocation = this.Site.TimeLocation()
			}
			if this.timeLocation == nil {
				this.timeLocation = time.Local
			}
		}
	}

	return this.timeLocation
}

func (this *Context) SetTimeLocation(timeLocation *time.Location) {
	this.timeLocation = timeLocation
}

func (this *Context) Err() error {
	if this.context != nil {
		if err := this.context.Err(); err != nil {
			return err
		}
	}
	return this.Errors
}

func (this *Context) Anonymous() bool {
	return this.CurrentUser() == nil
}

func (this Context) WithDB(f func(ctx *Context), db ...*aorm.DB) *Context {
	if len(db) > 0 && db[0] != nil {
		this.db = db[0]
	}
	f(&this)
	return &this
}

func (this Context) WithContext(ctx context.Context) *Context {
	this.context = ctx
	return &this
}

func (this *Context) SetValue(key, value interface{}) *Context {
	this.values.Set(key, value)
	return this
}

func (this *Context) DelValue(key interface{}) *Context {
	this.values.Del(key)
	return this
}

func (this *Context) Value(key interface{}) interface{} {
	if this.values != nil {
		if value, ok := this.values[key]; ok {
			return value
		}
	}
	if this.Request != nil {
		if value := this.Request.Context().Value(key); value != nil {
			return value
		}
	}
	if this.context != nil {
		if v := this.context.Value(key); v != nil {
			return v
		}
	}
	if this.Parent != nil {
		return this.Parent.Value(key)
	}
	return nil
}

func (this *Context) GetValue(key interface{}) interface{} {
	v, _ := this.Get(key)
	return v
}

func (this *Context) GetOrDefault(key, defaul interface{}) interface{} {
	if v, ok := this.Get(key); ok {
		return v
	}
	return defaul
}

func (this *Context) Get(key interface{}) (interface{}, bool) {
	c := this
	for c != nil {
		if v, ok := c.values.Get(key); ok {
			return v, true
		}
		c = c.Parent
	}
	return nil, false
}

func (this *Context) Flag(key interface{}) bool {
	if v, ok := this.Get(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (this *Context) FlashT(msg interface{}, typ string) (err error) {
	return this.SessionManager().Flash(session.TranslatedMessage(this.GetI18nContext(), msg, typ))
}

func (this *Context) FlashTOrError(msg interface{}, typ string) (ok bool) {
	if err := this.FlashT(msg, typ); err != nil {
		http.Error(this.Writer, "gsuite setup failed: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func (this *Context) Read(p []byte) (n int, err error) {
	return this.Request.Body.Read(p)
}

func (this *Context) Write(p []byte) (n int, err error) {
	return this.Writer.Write(p)
}

func (this *Context) CurrentUser() common.User {
	if this.currentUser == nil && this.Parent != nil {
		c := this.Parent
		for c != nil && c.currentUser == nil {
			c = c.Parent
		}
		if c != nil {
			this.currentUser = c.currentUser
		}
	}
	return this.currentUser
}

func (this *Context) UserID() aorm.ID {
	return aorm.IdOf(this.CurrentUser())
}

func (this *Context) SetCurrentUser(user common.User) {
	if user != nil {
		if defaultLocale := user.GetDefaultLocale(); defaultLocale != "" {
			for _, locale := range this.Translator.Locales {
				if locale == defaultLocale {
					this.DefaultLocale = locale
					break
				}
			}
		}
	}
	this.currentUser = user
}

func (this *Context) Breadcrumbs() *Breadcrumbs {
	if this.breadcrumbs == nil {
		this.breadcrumbs = &Breadcrumbs{}
	}
	return this.breadcrumbs
}

func (this *Context) PushI18nGroup(group string) func() {
	this.I18nGroupStack = &i18nGroup{this.I18nGroupStack, group}
	return this.PopI18nGroup
}

func (this *Context) PopI18nGroup() {
	this.I18nGroupStack = this.I18nGroupStack.Prev
}

func (this *Context) URLParams() *xroute.OrderedMap {
	return this.RouteContext.URLParams
}

func (this *Context) URLParam(key string) string {
	if this.RouteContext != nil {
		return this.RouteContext.URLParam(key)
	}
	return ""
}

func (this *Context) GetFormOrQuery(key string) (value string) {
	if this.Request.Form != nil {
		if value = this.Request.Form.Get(key); value != "" {
			return
		}
	}
	value = this.Request.URL.Query().Get(key)
	return
}

func (this *Context) SetRequest(r *http.Request) {
	this.Request = r
}

func (this *Context) RequestPath() string {
	return this.Request.URL.Path
}

// PatchCurrentURL is a convinent wrapper for qor/utils.PatchURL
func (this *Context) PatchCurrentURL(params ...interface{}) (patchedURL string, err error) {
	return uurl.PatchURL(this.OriginalURL.String(), params...)
}

// PatchURL is a convinent wrapper for qor/utils.PatchURL
func (this *Context) PatchURL(url string, params ...interface{}) (patchedURL string, err error) {
	return uurl.PatchURL(url, params...)
}

// JoinCurrentURL is a convinent wrapper for qor/utils.JoinURL
func (this *Context) JoinCurrentURL(params ...interface{}) (joinedURL string, err error) {
	return uurl.JoinURL(this.OriginalURL.String(), params...)
}

// JoinURL is a convinent wrapper for qor/utils.JoinURL
func (this *Context) JoinURL(url string, params ...interface{}) (joinedURL string, err error) {
	joinedURL, err = uurl.JoinURL(url, params...)
	return
}

func (this *Context) GetLocale() string {
	if this.Locale != "" {
		return this.Locale
	}

	if this.Request == nil {
		return ""
	}

	var locale string

	if locale = this.Request.Header.Get("Locale"); locale == "" {
		locale = this.Request.URL.Query().Get("locale")
		if locale == "" {
			if local, err := this.Request.Cookie("locale"); err == nil {
				locale = local.Value
			}
		} else if this.Writer != nil {
			this.Request.Header.Set("Locale", locale)
			cookie := http.Cookie{Name: "locale", Value: locale, Expires: time.Now().AddDate(1, 0, 0)}
			cookie.HttpOnly = true

			// set https cookie
			if this.Request != nil && this.Request.URL.Scheme == "https" {
				cookie.Secure = true
			}

			// set default path
			if cookie.Path == "" {
				cookie.Path = "/"
			}

			http.SetCookie(this.Writer, &cookie)
		}
	}

	locale = this.Translator.ValidOrDefaultLocale(locale)
	this.Locale = locale

	return locale
}

func (this *Context) GetI18nContext() i18nmod.Context {
	if this.I18nContext == nil {
		locale := this.GetLocale()
		this.I18nContext = this.Translator.NewContext(locale)
	}
	return this.I18nContext.WithContext(this)
}

func (this *Context) I18nT(key string) *i18nmod.T {
	return this.GetI18nContext().T(key)
}

func (this *Context) T(key string, defaul ...interface{}) template.HTML {
	return template.HTML(this.Ts(key, defaul...))
}

func (this *Context) TT(key string, data interface{}, defaul ...interface{}) template.HTML {
	return template.HTML(this.TTs(key, data, defaul...))
}

func (this *Context) Ts(key string, defaul ...interface{}) string {
	// if stack prefix
	if key[0] == '.' {
		if key[1] == '^' {
			key = "^" + this.I18nGroupStack.Value + "." + key[2:]
		} else {
			key = this.I18nGroupStack.Value + key
		}
	}
	return this.I18nT(key).DefaultArgs(defaul...).Get()
}

func (this *Context) TTs(key string, data interface{}, defaul ...interface{}) string {
	if key[0] == '.' {
		if key[1] == '^' {
			key = "^" + this.I18nGroupStack.Value + "." + key[2:]
		} else {
			key = this.I18nGroupStack.Value + key
		}
	}
	return this.GetI18nContext().TT(key).DefaultArgs(defaul...).Data(data).Get()
}

// Clone clone current context
func (this *Context) Clone() *Context {
	var clone = *this
	clone.Parent = this
	return &clone
}

// Clone clone current context
func (this *Context) CloneBasic() *Context {
	c := &Context{
		LocalContext:       this.LocalContext,
		Request:            this.Request,
		Writer:             this.Writer,
		StaticURL:          this.StaticURL,
		Prefix:             this.Prefix,
		OriginalURL:        this.OriginalURL,
		db:                 this.db,
		Site:               this.Site,
		Parent:             this.Parent,
		currentUser:        this.currentUser,
		Locale:             this.Locale,
		Lang:               this.Lang,
		LangTag:            this.LangTag,
		timeLocation:       this.timeLocation,
		I18nContext:        this.I18nContext,
		Translator:         this.Translator,
		NotFound:           this.NotFound,
		Api:                this.Api,
		logger:             this.logger,
		DefaultLocale:      this.DefaultLocale,
		MetaContextFactory: this.MetaContextFactory,
		MetaTreeStack:      this.MetaTreeStack,
		FormOptions:        this.FormOptions,
	}
	return c
}

func (this *Context) Top() *Context {
	if this.top != nil {
		return this.top
	}
	if this.Parent != nil && !this.isTop {
		this.top = this.Parent.Top()
		return this.top
	}
	return this
}

func (this *Context) AsTop() *Context {
	this.isTop = true
	return this
}

func (this *Context) IsTop() bool {
	return this.isTop || this.Parent == nil
}

func (this *Context) GetStaticURL() string {
	prefix := this.StaticURL

	if prefix == "" {
		v2 := this.Value("STATIC_URL")
		if v3, ok := v2.(string); ok {
			this.StaticURL = v3
			prefix = v3
		}
	}
	return prefix
}

func (this *Context) JoinStaticURL(path ...string) string {
	path = uri.Clean(path)
	prefix := this.GetStaticURL()

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

	return uri.Join(uri.Clean(append([]string{prefix}, path...))...)
}

func (this *Context) Path(path ...string) string {
	path = uri.Clean(path)
	prefix := this.Prefix

	if prefix == "" {
		v2 := this.Value("PREFIX")
		if v3, ok := v2.(string); ok {
			this.Prefix = v3
			prefix = v3
		}
	}

	if len(path) == 0 || path[0] == "" {
		return prefix
	}

	return uri.Join(uri.Clean(append([]string{prefix}, path...))...)
}

func (this *Context) URL(pth ...string) string {
	return httpu.URL(this.Request, append([]string{this.Prefix}, pth...)...)
}

func (this *Context) WsURL(pth ...string) string {
	return httpu.WsURL(this.Request, append([]string{this.Prefix}, pth...)...)
}

func (this *Context) JoinPath(path ...string) string {
	if len(path) == 0 {
		return ""
	}

	var parts []string
	if this.Prefix != "" {
		parts = append(parts, this.Prefix)
	}
	return uri.Join(uri.Clean(append(parts, path...))...)
}

func (this *Context) Root() *Context {
	c := this
	for c.Parent != nil {
		c = c.Parent
	}
	return c
}

func (this *Context) NewChild(r *http.Request, prefix ...string) (*http.Request, *Context) {
	var pth string
	if len(prefix) > 0 {
		pth = prefix[0]
	}
	if pth == "/" {
		pth = ""
	}
	child := this.Clone()
	child.isTop = false
	child.Parent = this

	if pth != "" {
		child.StaticURL = path.Join(child.StaticURL, pth)
		child.Prefix = path.Join(child.Prefix, pth)
		if r == nil {
			r = this.Request
		}
		if r != nil {
			nurl := *r.URL
			nurl2 := &nurl
			nurl2.Path = strings.TrimPrefix(r.URL.Path, pth)
			if nurl2.Path == "" || nurl2.Path[0] != '/' {
				nurl2.Path = "/" + nurl2.Path
			}
			var err error
			r.URL, err = url.Parse(nurl2.String())
			if err != nil {
				panic(err)
			}
		}
	} else if r != nil {
		child.Request = r
	}
	return child.Request, child
}

func (this *Context) DB(db ...*aorm.DB) *aorm.DB {
	for _, db := range db {
		this.SetDB(db)
	}
	return this.db
}

func (this *Context) SetRawDB(db *aorm.DB) *Context {
	this.db = db
	return this
}

func (this *Context) SetDB(db *aorm.DB) *Context {
	if db != nil {
		db.Context = this
		db = db.Set(CONTEXT_KEY, this)
	}
	this.db = db
	return this
}

func (this *Context) Logger() logging.Logger {
	if this.logger == nil {
		this.logger = defaultlogger.GetOrCreateLogger(this.OriginalURL.String())
	}
	return this.logger
}

func (this *Context) Htmlify(value interface{}) template.HTML {
	switch vt := value.(type) {
	case template.HTML:
		return vt
	case string:
		return template.HTML(vt)
	case interface{ Htmlify() template.HTML }:
		return vt.Htmlify()
	case interface{ Htmlify(*Context) template.HTML }:
		return vt.Htmlify(this)
	default:
		return template.HTML(fmt.Sprint(vt))
	}
}

func (this *Context) GetErrorsT() []error {
	return this.Errors.GetErrorsT(this.GetI18nContext())
}

func (this *Context) GetErrorsTS() []string {
	return this.Errors.GetErrorsTS(this.GetI18nContext())
}

func (this *Context) ErrorResult() Result {
	return Result{Messages: this.GetErrorsTS(), Type: "error"}
}

func (this *Context) ErrorT(err error) error {
	return i18nmod.ErrorCtx(this.GetI18nContext(), err)
}

func (this *Context) ErrorTS(err error) string {
	return i18nmod.ErrorCtxS(this.GetI18nContext(), err)
}

type DecoderExclude struct {
	ID   aorm.ID
	Path string
	Data interface{}
}

type DecoderExcludes struct {
	Excludes []DecoderExclude
}

func (this *DecoderExcludes) Add(id aorm.ID, pth string, data interface{}) {
	this.Excludes = append(this.Excludes, DecoderExclude{id, pth, data})
}

func (this *DecoderExcludes) HasItemOf(path string) bool {
	path += "."
	for _, e := range this.Excludes {
		if strings.HasPrefix(e.Path, path) {
			return true
		}
	}
	return false
}
