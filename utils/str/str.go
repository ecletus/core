package str

import (
	"strings"
	"path/filepath"
)

type Vars struct {
	Data   map[string]string
	parent *Vars
}

func (v *Vars) Merge(vs ...map[string]string) *Vars {
	for _, vs_ := range vs {
		if vs_ != nil {
			v = &Vars{vs_, v}
		}
	}
	return v
}

func (v *Vars) Pairs(cb func(k, v string)) {
	for key, value := range v.Data {
		cb(key, value)
	}
}

func (v *Vars) Priority() (ld []*Vars) {
	for v != nil {
		ld = append(ld, v)
		v = v.parent
	}
	return ld
}

func (v Vars) Format(s string) string {
	for _, d := range v.Priority() {
		d.Pairs(func(key, value string) {
			s = strings.Replace(s, "{"+key+"}", value, -1)
		})
	}

	return s
}

func (v Vars) FormatPath(s string) string {
	return filepath.Clean(v.Format(s))
}

func (v *Vars) FormatPtr(sptrs ...*string) *Vars {
	for _, s := range sptrs {
		*s = v.Format(*s)
	}
	return v
}

func (v *Vars) FormatPathPtr(sptrs ...*string) *Vars {
	for _, s := range sptrs {
		*s = filepath.Clean(v.Format(*s))
	}
	return v
}

func (v *Vars) Get(key string) (r string, ok bool) {
	for r, ok = v.Data[key]; !ok; {
		v = v.parent
	}
	return
}

func (v *Vars) GetData() map[string]string {
	if v.parent == nil {
		return v.Data
	}

	d := map[string]string{}
	for v != nil {
		v.Pairs(func(k, v string) {
			if _, ok := d[k]; !ok {
				d[k] = v
			}
		})
		v = v.parent
	}
	return d
}
