package utils

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"go4.org/sort"

	"github.com/ecletus/helpers"
	"github.com/gosimple/slug"
	"github.com/jinzhu/now"
	"github.com/microcosm-cc/bluemonday"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"

	"github.com/moisespsena/template/html/template"
)

// AppRoot app root path
var AppRoot, _ = os.Getwd()

// ContextKey defined type used for context's key
type ContextKey string

// DefaultLang Default App Language
var DefaultLocale string

// HTMLSanitizer html sanitizer to avoid XSS
var HTMLSanitizer = bluemonday.UGCPolicy()

func init() {
	HTMLSanitizer.AllowStandardAttributes()
	if path := os.Getenv("WEB_ROOT"); path != "" {
		AppRoot = path
	}

	lang := os.Getenv("LANG")

	if len(lang) >= 5 {
		DefaultLocale = strings.Replace(strings.Split(lang, ".")[0], "_", "-", 1)
	}
}

// HumanizeString Humanize separates string based on capitalizd letters
// e.g. "order_item-data" -> "OrderItemData"
func NamifyString(s string) string {
	var human []rune
	var toUpper bool
	s = "_" + s
	for _, c := range s {
		if c == '_' || c == '-' {
			toUpper = true
			continue
		} else if c == '!' {
			toUpper = true
		} else if toUpper {
			toUpper = false
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
		}
		human = append(human, c)
	}
	return string(human)
}

// HumanizeString Humanize separates string based on capitalizd letters
// e.g. "OrderItem" -> "Order Item"
func HumanizeString(str string) string {
	var human []rune
	for i, l := range str {
		if i > 0 && isUppercase(byte(l)) {
			if (!isUppercase(str[i-1]) && str[i-1] != ' ') || (i+1 < len(str) && !isUppercase(str[i+1]) && str[i+1] != ' ' && str[i-1] != ' ') {
				human = append(human, rune(' '))
			}
		}
		human = append(human, l)
	}
	return strings.Title(string(human))
}

// HumanizeString Humanize separates string based on capitalizd letters
// e.g. "OrderItem" -> "Order Item"
// e.g. "Order_item" -> "Order Item"
func HumanizeStringU(str string) string {
	var human []rune
	for i, l := range str {
		if i > 0 {
			if isUppercase(byte(l)) {
				if (!isUppercase(str[i-1]) && str[i-1] != ' ') || (i+1 < len(str) && !isUppercase(str[i+1]) && str[i+1] != ' ' && str[i-1] != ' ') {
					human = append(human, rune(' '))
				}
			} else if l == '_' {
				human = append(human, rune(' '))
				continue
			}
		}
		human = append(human, l)
	}
	return strings.Title(string(human))
}

func isUppercase(char byte) bool {
	return 'A' <= char && char <= 'Z'
}

var asicsiiRegexp = regexp.MustCompile(`^(\w|\s|-|/!)*$`)

type safeMap struct {
	m map[string]string
	l *sync.RWMutex
}

func (s *safeMap) Set(key string, value string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeMap) Get(key string) string {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newSafeMap() *safeMap {
	return &safeMap{l: new(sync.RWMutex), m: make(map[string]string)}
}

var paramSnb = aorm.NewSafeNameBuilder(newSafeMap())

func init() {
	paramSnb.PreFormat = func(v string) string {
		return strings.Replace(v, " ", "_", -1)
	}
}

func ToUri(value string) string {
	value = path.Clean(value)
	return paramSnb.BuildParts(value, "/")
}

// ToParamString replaces spaces and separates words (by uppercase letters) with
// underscores in a string, also downcase it
// e.g. ToParamString -> to_param_string, To ParamString -> to_param_string
func ToParamString(str string) (r string) {
	if asicsiiRegexp.MatchString(str) {
		return paramSnb.BuildParts(str, "/")
	}
	parts := strings.Split(str, "/")
	for i := range parts {
		parts[i] = slug.Make(parts[i])
	}
	return strings.Join(parts, "/")
}

// SetCookie set cookie for context
func SetCookie(cookie http.Cookie, context *core.Context) {
	cookie.HttpOnly = true

	// set https cookie
	if context.Request != nil && context.Request.URL.Scheme == "https" {
		cookie.Secure = true
	}

	// set default path
	if cookie.Path == "" {
		cookie.Path = context.Root().Path()
	}

	http.SetCookie(context.Writer, &cookie)
}

// Stringify stringify any data, if it is a struct, will try to use its Name, Title, Code field, else will use its primary key
func Stringify(object interface{}) string {
	if helpers.IsNilInterface(object) {
		return ""
	}
	if obj, ok := object.(interface {
		Stringify() string
	}); ok {
		return obj.Stringify()
	}
	if obj, ok := object.(fmt.Stringer); ok {
		return obj.String()
	}
	if reflect.Indirect(reflect.ValueOf(object)).Kind() == reflect.Struct {
		instance := aorm.InstanceOf(object)
		for _, column := range []string{"Name", "Title", "Code"} {
			if field, ok := instance.FieldByName(column); ok {
				if field.Field.IsValid() {
					result := field.Field.Interface()
					if valuer, ok := result.(driver.Valuer); ok {
						if result, err := valuer.Value(); err == nil {
							return fmt.Sprint(result)
						}
					}
					return fmt.Sprint(result)
				}
				return ""
			}
		}

		if pk := instance.ID(); pk != nil {
			if pk.IsZero() {
				return ""
			}
			return fmt.Sprintf("%v #%v", HumanizeStringU(instance.Model.Type.Name()), pk)
		}
	}
	return fmt.Sprint(object)
}

// StringifyContext stringify any data, if it is a struct, will try to use its Name, Title, Code field, else will use its primary key
func StringifyContext(object interface{}, ctx *core.Context) string {
	if helpers.IsNilInterface(object) {
		return ""
	}
	if obj, ok := object.(core.ContextStringer); ok {
		return obj.ContextString(ctx)
	}
	return Stringify(object)
}

func HtmlifyContext(value interface{}, ctx *core.Context) template.HTML {
	switch vt := value.(type) {
	case interface{ Htmlify() template.HTML }:
		return vt.Htmlify()
	case interface {
		Htmlify(*core.Context) template.HTML
	}:
		return vt.Htmlify(ctx)
	default:
		return template.HTML(StringifyContext(value, ctx))
	}
}

// Type get value's model type
func ModelType(value interface{}) reflect.Type {
	reflectType := reflect.Indirect(reflect.ValueOf(value)).Type()

	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}

	return reflectType
}

// ParseTagOption parse tag options to hash
func ParseTagOption(str string) map[string]string {
	tags := strings.Split(str, ";")
	setting := map[string]string{}
	for _, value := range tags {
		v := strings.Split(value, ":")
		k := strings.TrimSpace(strings.ToUpper(v[0]))
		if len(v) == 2 {
			setting[k] = v[1]
		} else {
			setting[k] = k
		}
	}
	return setting
}

// FileServer file server that disabled file listing
func FileServer(dir http.Dir) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Join(string(dir), r.URL.Path)
		if f, err := os.Stat(p); err == nil && !f.IsDir() {
			http.ServeFile(w, r, p)
			return
		}

		http.NotFound(w, r)
	})
}

func filenameWithLineNum() string {
	var total = 10
	var results []string
	for i := 2; i < 15; i++ {
		if _, file, line, ok := runtime.Caller(i); ok {
			total--
			results = append(results[:0],
				append(
					[]string{fmt.Sprintf("%v:%v", strings.TrimPrefix(file, os.Getenv("GOPATH")+"src/"), line)},
					results[0:]...)...)

			if total == 0 {
				return strings.Join(results, "\n")
			}
		}
	}
	return ""
}

// GetLocale get locale from request, cookie, after get the locale, will write the locale to the cookie if possible
// Overwrite the default logic with
//     utils.GetLocale = func(context *qor.Context) string {
//         // ....
//     }
var GetLocale = func(context *core.Context) string {
	if locale := context.Request.Header.Get("Locale"); locale != "" {
		return locale
	}

	if locale := context.Request.URL.Query().Get("locale"); locale != "" {
		if context.Writer != nil {
			context.Request.Header.Set("Locale", locale)
			SetCookie(http.Cookie{Name: "locale", Value: locale, Expires: time.Now().AddDate(1, 0, 0)}, context)
		}
		return locale
	}

	if locale, err := context.Request.Cookie("locale"); err == nil {
		return locale.Value
	}

	return DefaultLocale
}

// ParseTime parse time from string
// Overwrite the default logic with
//     utils.ParseTime = func(timeStr string, context *qor.Context) (time.Time, error) {
//         // ....
//     }
var ParseTime = func(timeStr string, context *core.Context) (time.Time, error) {
	return now.Parse(timeStr)
}

// FormatTime format time to string
// Overwrite the default logic with
//     utils.FormatTime = func(time time.Time, layout string, context *qor.Context) string {
//         // ....
//     }
var FormatTime = func(date time.Time, layout string, context *core.Context) string {
	return date.Format(layout)
}

var replaceIdxRegexp = regexp.MustCompile(`\.\d+`)

// SortFormKeys sort form keys
func SortFormKeys(strs []string) {
	sort.Slice(strs, func(i, j int) bool { // true for first
		str1 := strs[i]
		str2 := strs[j]
		matched1 := replaceIdxRegexp.FindAllStringIndex(str1, -1)
		matched2 := replaceIdxRegexp.FindAllStringIndex(str2, -1)

		for x := 0; x < len(matched1); x++ {
			prefix1 := str1[:matched1[x][0]]
			prefix2 := str2

			if len(matched2) >= x+1 {
				prefix2 = str2[:matched2[x][0]]
			}

			if prefix1 != prefix2 {
				return strings.Compare(prefix1, prefix2) < 0
			}

			if len(matched2) < x+1 {
				return false
			}

			number1 := str1[matched1[x][0]:matched1[x][1]]
			number2 := str2[matched2[x][0]:matched2[x][1]]

			if number1 != number2 {
				if len(number1) != len(number2) {
					return len(number1) < len(number2)
				}
				return strings.Compare(number1, number2) < 0
			}
		}

		return strings.Compare(str1, str2) < 0
	})
}

// GetAbsURL get absolute URL from request, refer: https://stackoverflow.com/questions/6899069/why-are-request-url-host-and-scheme-blank-in-the-development-server
func GetAbsURL(req *http.Request) url.URL {
	var result url.URL

	if req.URL.IsAbs() {
		return *req.URL
	}

	if domain := req.Host; domain != "" {
		parseResult, _ := url.Parse(domain)
		result = *parseResult
	}

	result.Parse(req.RequestURI)
	return result
}

func RenderHtmlTemplate(tpl string, data interface{}) template.HTML {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return template.HTML(fmt.Sprint(err))
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return template.HTML(fmt.Sprint(err))
	}
	return template.HTML(buf.String())
}

func TypeId(tp interface{}) string {
	typ := IndirectType(tp)
	return typ.PkgPath() + "." + typ.Name()
}

func IndirectItemType(v interface{}) (typ reflect.Type) {
	typ = IndirectType(v)
	switch typ.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		typ = IndirectType(typ.Elem())
	}
	return
}

func IndirectType(v interface{}) (typ reflect.Type) {
	var isType bool
	switch vt := v.(type) {
	case reflect.Value:
		typ = vt.Type()
	case *reflect.Value:
		typ = vt.Type()
	case reflect.Type:
		typ = vt
		isType = true
	default:
		typ = reflect.ValueOf(vt).Type()
	}

	if isType {
		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
	} else {
		for {
			switch typ.Kind() {
			case reflect.Ptr, reflect.Interface:
				typ = typ.Elem()
			default:
				return
			}
		}
	}
	return
}

func StructType(v interface{}) (typ reflect.Type) {
	switch vt := v.(type) {
	case reflect.Value:
		typ = vt.Type()
	case *reflect.Value:
		typ = vt.Type()
	case reflect.Type:
		typ = vt
	default:
		typ = reflect.ValueOf(vt).Type()
	}
	for typ.Kind() != reflect.Struct {
		typ = typ.Elem()
	}
	return
}

func StringsFilter(accept func(s string) bool, dst *[]string, src []string) {
	for _, v := range src {
		if accept(v) {
			*dst = append(*dst, v)
		}
	}
}

func NoEmptyStrings(values ...string) (res []string) {
	StringsFilter(func(s string) bool {
		return s != ""
	}, &res, values)
	return
}

func StringsJoin(sep, lastSep string, values ...string) string {
	var (
		b   bytes.Buffer
		wc  bool
		end = len(values) - 1
	)
	if end < 0 {
		return ""
	}
	for _, v := range values[0:end] {
		if wc {
			b.WriteString(sep)
		}
		wc = true
		b.WriteString(v)
	}

	if wc {
		b.WriteString(lastSep)
	}
	b.WriteString(values[end])
	return b.String()
}
