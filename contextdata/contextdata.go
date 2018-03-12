package contextdata

import (
	"fmt"
	"context"
	"net/http"

	"github.com/jinzhu/gorm"
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

func (d *ContextData) Set(key, value interface{}, pairs ... interface{}) *ContextData {
	d.Current.Data[key] = value
	l := len(pairs)
	for i := 0; i < l; i = i + 2 {
		d.Current.Data[pairs[i]] = pairs[i+1]
	}
	return d
}

func (d *ContextData) Get(key interface{}) interface{} {
	node := d.Current
	for node != nil {
		if v, ok := node.Data[key]; ok {
			return v
		}
		node = node.Parent
	}
	return nil
}

func (d *ContextData) SetDB(dbname string, db *gorm.DB) *ContextData {
	return d.Set("db:"+dbname, db)
}

func (d *ContextData) GetDB(dbname string) *gorm.DB {
	return d.Get("db:" + dbname).(*gorm.DB)
}

func (d *ContextData) RequireDB(dbname string) *gorm.DB {
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
