package core

import (
	"context"
	"time"

	"github.com/moisespsena-go/maps"
)

type ContextProxy struct {
	Getter func()context.Context
}

func (this *ContextProxy) Deadline() (deadline time.Time, ok bool) {
	if ctx := this.Getter(); ctx != nil {
		return ctx.Deadline()
	}
	return
}

func (this *ContextProxy) Done() <-chan struct{} {
	if ctx := this.Getter(); ctx != nil {
		return ctx.Done()
	}
	return nil
}

func (this *ContextProxy) Err() error {
	if ctx := this.Getter(); ctx != nil {
		return ctx.Err()
	}
	return nil
}

func (this *ContextProxy) Value(key interface{}) (value interface{}) {
	if ctx := this.Getter(); ctx != nil {
		return ctx.Value(key)
	}
	return
}

func (this *ContextProxy) Get(key interface{}) (value interface{}, ok bool) {
	value = this.Value(key)
	ok = value != nil
	return
}

type LocalContext struct {
	context context.Context
	values  maps.Map
}

func (this *LocalContext) Deadline() (deadline time.Time, ok bool) {
	if this.context != nil {
		return this.context.Deadline()
	}
	return
}

func (this *LocalContext) Done() <-chan struct{} {
	if this.context != nil {
		return this.context.Done()
	}
	return nil
}

func (this *LocalContext) Err() error {
	if this.context != nil {
		return this.context.Err()
	}
	return nil
}

func (this *LocalContext) Value(key interface{}) (value interface{}) {
	var ok bool
	if value, ok = this.values.Get(key); ok {
		return
	}
	if this.context != nil {
		return this.context.Value(key)
	}
	return
}

func (this *LocalContext) Get(key interface{}) (value interface{}, ok bool) {
	if value, ok = this.values.Get(key); ok {
		return
	}
	if this.context != nil {
		if value = this.context.Value(key); value != nil {
			ok = true
		}
	}
	return
}

func (this *LocalContext) SetValue(key, value interface{}) *LocalContext {
	this.values.Set(key, value)
	return this
}

func (this *LocalContext) SetValues(key, value interface{}, pairs ...interface{}) *LocalContext {
	this.values[key] = value
	l := len(pairs)
	for i := 0; i < l; i = i + 2 {
		this.values[pairs[i]] = pairs[i+1]
	}
	return this
}

func (this *LocalContext) GetContext() context.Context {
	return this.context
}

func (this *LocalContext) SetContext(ctx context.Context) {
	this.context = ctx
}

func (this LocalContext) WithContext(ctx context.Context) *LocalContext {
	this.context = ctx
	return &this
}

func (this *LocalContext) BackupValues() func() {
	old := this.context
	this.context = &LocalContext{context: old}
	return func() {
		this.context = old
	}
}
