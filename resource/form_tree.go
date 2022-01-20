package resource

import (
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type FormTreeMap struct {
	Keys      []string
	Map       map[string]*FormTree
	Slice     []*FormTree
	NextIndex int
}

func (this *FormTreeMap) Add(key string, t *FormTree) {
	if this.Map == nil {
		this.Map = map[string]*FormTree{}
	}
	this.Map[key] = t
	this.Keys = append(this.Keys, key)

	if _, err := strconv.ParseUint(key, 10, 32); err == nil {
		t.Index = this.NextIndex
		this.NextIndex++
		this.Slice = append(this.Slice, t)
	}
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns
// the empty string. To access multiple values, use the map
// directly.
func (this *FormTreeMap) GetStringValue(key string) string {
	if this.Map == nil {
		return ""
	}
	child := this.Map[key]
	if child == nil || child.Value == nil {
		return ""
	}
	switch t := child.Value.(type) {
	case string:
		return t
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	}
	return ""
}

func (this *FormTreeMap) Get(key string) (t *FormTree, ok bool) {
	if this.Map == nil {
		return
	}
	t, ok = this.Map[key]
	return
}

type FormTree struct {
	Key      string
	Index    int
	Value    interface{}
	Children FormTreeMap
	Parent   *FormTree
	Data     interface{}
}

func NewFormTree() *FormTree {
	return &FormTree{Index: -1}
}

func (this *FormTree) Of(key string) *FormTree {
	parts := strings.Split(strings.ReplaceAll(strings.ReplaceAll(key, "]", ""), "[", "."), ".")

	var (
		el    = this
		child *FormTree
		ok    bool
	)

	for _, key := range parts {
		if key[0] == '[' {
			key = key[1 : len(key)-1]
		}

		if child, ok = el.Children.Get(key); !ok {
			child = &FormTree{Parent: el, Key: key, Index: -1}
			el.Children.Add(key, child)
		}

		el = child
	}

	return child
}

func (this *FormTree) ParseFormTreeValues(m map[string][]string, prefix string) {
	for key, values := range m {
		key = strings.TrimPrefix(key, prefix)
		this.Of(key).Value = values
	}
}

func (this *FormTree) ParseFormTreeFiles(m map[string][]*multipart.FileHeader, prefix string) {
	for key, values := range m {
		key = strings.TrimPrefix(key, prefix)
		this.Of(key).Value = values
	}
}

func (this *FormTree) Walk(cb func(t *FormTree, data interface{}) (childData interface{}, err error), data interface{}) (err error) {
	for _, key := range this.Children.Keys {
		child := this.Children.Map[key]
		var childData interface{}
		if childData, err = cb(child, data); err != nil {
			if err == FormTreeSkipDir {
				continue
			}
			return
		}
		if len(child.Children.Keys) > 0 {
			if err = child.Walk(cb, childData); err != nil {
				if err != FormTreeSkipDir {
					return
				}
			}
		}
	}
	return
}

var FormTreeSkipDir = errors.New("skip dir")
