package contextdata

import (
	"context"
	"fmt"
	"net/http"

	"github.com/moisespsena-go/aorm"
)

type ContextDataNode struct {
	Parent *ContextDataNode
	Data   map[interface{}]interface{}
}

type ContextData struct {
	Current *ContextDataNode
}

func NewRequestContextData() *ContextData {
	return (&ContextData{}).Inside()
}

func (d *ContextData) Set(key, value interface{}, pairs ...interface{}) *ContextData {
	d.Current.Data[key] = value
	l := len(pairs)
	for i := 0; i < l; i = i + 2 {
		d.Current.Data[pairs[i]] = pairs[i+1]
	}
	return d
}

func (d *ContextData) Get(key interface{}) interface{} {
	v, _ := d.GetOk(key)
	return v
}

func (d *ContextData) GetInterface(key interface{}) interface{} {
	v, _ := d.GetOk(key)
	return v
}

func (d *ContextData) GetString(key interface{}) string {
	if v, ok := d.GetOk(key); ok {
		return v.(string)
	}
	return ""
}

func (d *ContextData) GetOk(key interface{}) (interface{}, bool) {
	node := d.Current
	for node != nil {
		if v, ok := node.Data[key]; ok {
			return v, true
		}
		node = node.Parent
	}
	return nil, false
}

func (d *ContextData) GetCallback(key interface{}, cb func(v interface{}, ok bool) interface{}) interface{} {
	v, ok := d.GetOk(key)
	return cb(v, ok)
}

func (d *ContextData) SetDB(dbname string, db *aorm.DB) *ContextData {
	return d.Set("db:"+dbname, db)
}

func (d *ContextData) GetDB(dbname string) *aorm.DB {
	return d.Get("db:" + dbname).(*aorm.DB)
}

func (d *ContextData) RequireDB(dbname string) *aorm.DB {
	db := d.GetDB(dbname)
	if db == nil {
		panic(fmt.Sprint("Database %q isn't set in context data.", dbname))
	}
	return db
}

func (d *ContextData) GetLocal(key interface{}) interface{} {
	return d.Current.Data[key]
}

func (d *ContextData) Inside() *ContextData {
	d.Current = &ContextDataNode{d.Current, map[interface{}]interface{}{}}
	return d
}

func (d *ContextData) With() func() {
	d.Inside()
	return func() {
		d.Outside()
	}
}

func (d *ContextData) Outside() *ContextData {
	d.Current = d.Current.Parent
	return d
}

const requestContextDataKey = "qor:request_context_data"

func GetOrSetRequestContextData(req *http.Request) (*http.Request, *ContextData) {
	v := req.Context().Value(requestContextDataKey)
	if v != nil {
		return req, v.(*ContextData)
	}
	rcd := NewRequestContextData()
	return req.WithContext(context.WithValue(req.Context(), requestContextDataKey, rcd)), rcd
}
