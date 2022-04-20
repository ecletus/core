package core

import (
	"context"
	"net/http"

	"github.com/go-aorm/aorm"
)

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

func ContextToRequest(req *http.Request, ctx *Context) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), CONTEXT_KEY, ctx))
}

func ContextFromDB(db *aorm.DB) *Context {
	v, _ := db.Get(CONTEXT_KEY)
	if v == nil {
		return nil
	}
	return v.(*Context)
}
