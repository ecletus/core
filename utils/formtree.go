package utils

import (
	"mime/multipart"
	"net/url"
	"sort"
)

type FormTreeItemSlice struct {
	Items []*FormTreeItem
}

type FormTreeItem struct {
	Tree   *FormTree               `json:",omitempty"`
	Values []string                `json:",omitempty"`
	Files  []*multipart.FileHeader `json:",omitempty"`
	Index  int                     `json:",omitempty"`
}

type FormTree struct {
	Map      FormTreeMap           `json:",omitempty"`
	Keys     []string              `json:",omitempty"`
	SliceMap map[int]*FormTreeItem `json:"-"`
	Slice    FormTreeSlice         `json:",omitempty"`
}

func NewFormTree(form url.Values) (*FormTree, error) {
	return NewMultipartFormTree(&multipart.Form{Value: form})
}

func NewMultipartFormTree(mform *multipart.Form) (_ *FormTree, err error) {
	var (
		keyParts []interface{}
		tree     = &FormTree{}
	)

	for key, values := range mform.Value {
		if keyParts, err = ParseFormKey(key); err != nil {
			return
		}
		tree.AddValue(keyParts, values)
	}

	if mform.File != nil {
		for key, files := range mform.File {
			if keyParts, err = ParseFormKey(key); err != nil {
				return
			}
			tree.AddFile(keyParts, files)
		}
	}

	tree.Sort()
	return tree, nil
}

func (this *FormTree) Sort() {
	sort.Strings(this.Keys)

	if this.SliceMap != nil {
		var sliceKeys []int
		for key := range this.SliceMap {
			sliceKeys = append(sliceKeys, key)
		}
		sort.Ints(sliceKeys)
		this.Slice = make([]*FormTreeItem, len(sliceKeys))
		for i, key := range sliceKeys {
			this.Slice[i] = this.SliceMap[key]
			if this.Slice[i].Tree != nil {
				this.Slice[i].Tree.Sort()
			}
		}
	}

	if this.Map != nil {
		for _, item := range this.Map {
			if item.Tree != nil {
				item.Tree.Sort()
			}
		}
	}
}

func (this *FormTree) AddValue(key []interface{}, values []string) {
	this.add(key, func(item *FormTreeItem) {
		item.Values = values
	})
}

func (this *FormTree) AddFile(key []interface{}, files []*multipart.FileHeader) {
	this.add(key, func(item *FormTreeItem) {
		item.Files = files
	})
}

func (this *FormTree) add(key []interface{}, cb func(item *FormTreeItem)) {
	var cur, sub = key[0], key[1:]
	switch k := cur.(type) {
	case int:
		if this.SliceMap == nil {
			this.SliceMap = map[int]*FormTreeItem{}
		}
		if len(sub) > 0 {
			item, ok := this.SliceMap[k]
			if ok {
				if item.Tree == nil {
					item.Tree = &FormTree{}
				}
			} else {
				item = &FormTreeItem{Tree: &FormTree{}}
				this.SliceMap[k] = item
			}
			item.Tree.add(sub, cb)
		} else {
			item := &FormTreeItem{}
			cb(item)
			if k == -1 {
				k = len(this.SliceMap)
			}
			this.SliceMap[k] = item
		}
	case string:
		if this.Map == nil {
			this.Map = map[string]*FormTreeItem{}
		}
		if len(sub) > 0 {
			item, ok := this.Map[k]
			if ok {
				if item.Tree == nil {
					item.Tree = &FormTree{}
				}
			} else {
				item = &FormTreeItem{Tree: &FormTree{}}
				this.Map[k] = item
			}
			item.Tree.add(sub, cb)
		} else {
			item := &FormTreeItem{}
			cb(item)
			this.Map[k] = item
		}
	}
}

func (this *FormTree) Each(slicePriority bool, cb func(data interface{}) error) error {
	if this.Slice != nil {
		if slicePriority {
			return cb(this.Slice)
		}
	}
	if this.Map != nil {
		return cb(this.Map)
	}
	if this.Slice != nil {
		return cb(this.Slice)
	}
	return nil
}

type FormTreeSlice []*FormTreeItem
type FormTreeMap map[string]*FormTreeItem
