package core

import "github.com/pkg/errors"

var ErrStopIteration = errors.New("stop iteration")

type NamedContextHandler struct {
	Name    string
	Handler func(ctx *Context)
}

type NamedContextHandlersRegistrator interface {
	InheritsFrom(registrator ...NamedContextHandlersRegistrator)
	Add(handler ...*NamedContextHandler)
	Get(name string) *NamedContextHandler
	Each(cb func(handler *NamedContextHandler) (err error), state ...map[string]*NamedContextHandler) (err error)
}

type NamedContextHandlersRegistry struct {
	handlers     []*NamedContextHandler
	byName       map[string]*NamedContextHandler
	inheritsFrom []NamedContextHandlersRegistrator
}

func (this *NamedContextHandlersRegistry) InheritsFrom(registrator ...NamedContextHandlersRegistrator) {
	this.inheritsFrom = append(this.inheritsFrom, registrator...)
}

func (this *NamedContextHandlersRegistry) Add(handler ...*NamedContextHandler) {
	this.handlers = append(this.handlers, handler...)
	if this.byName == nil {
		this.byName = make(map[string]*NamedContextHandler, len(handler))
	}
	for _, s := range handler {
		this.byName[s.Name] = s
	}
}

func (this *NamedContextHandlersRegistry) Each(cb func(handler *NamedContextHandler) (err error), state ...map[string]*NamedContextHandler) (err error) {
	var s map[string]*NamedContextHandler
	for _, s = range state {
	}
	if s == nil {
		s = map[string]*NamedContextHandler{}
	}

	if this.byName != nil {
		var ok bool
		for name, f := range this.byName {
			if _, ok = s[name]; !ok {
				s[name] = f
				if err = cb(f); err == ErrStopIteration {
					return nil
				} else if err != nil {
					return
				}
			}
		}
	}
	for _, parent := range this.inheritsFrom {
		if err = parent.Each(cb, s); err == ErrStopIteration {
			return nil
		} else if err != nil {
			return
		}
	}
	return nil
}

func (this *NamedContextHandlersRegistry) Get(name string) (f *NamedContextHandler) {
	if this.byName != nil {
		if f = this.byName[name]; f != nil {
			return
		}
	}
	for _, parent := range this.inheritsFrom {
		if f = parent.Get(name); f != nil {
			return
		}
	}
	return
}

func GetNamedContextHandlers(this NamedContextHandlersRegistrator) (handlers []*NamedContextHandler, err error) {
	ok := func(f *NamedContextHandler) bool {
		return true
	}
	err = this.Each(func(f *NamedContextHandler) (err error) {
		if ok(f) {
			handlers = append(handlers, f)
		}
		return nil
	})
	return
}

func MustGetNamedContextHandlers(this NamedContextHandlersRegistrator) (handlers []*NamedContextHandler, err error) {
	if handlers, err = GetNamedContextHandlers(this); err != nil {
		panic(err)
	}
	return
}
