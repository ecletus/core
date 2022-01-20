package core

import (
	"github.com/mitchellh/mapstructure"
	"github.com/moisespsena-go/getters"
)

type SiteGetter interface {
	Get(site *Site, key interface{}) (value interface{}, ok bool)
	GetInterface(site *Site, key, dest interface{}) (ok bool)
}

type MultipleSiteGetter []SiteGetter

func (this MultipleSiteGetter) Get(site *Site, key interface{}) (value interface{}, ok bool) {
	for _, g := range this {
		if value, ok = g.Get(site, key); ok {
			return
		}
	}
	return
}
func (this MultipleSiteGetter) GetInterface(site *Site, key, dest interface{}) (ok bool) {
	for _, g := range this {
		if g.GetInterface(site, key, dest) {
			return
		}
	}
	return
}

func (this *MultipleSiteGetter) Append(g ...SiteGetter) *MultipleSiteGetter {
	(*this) = append((*this), g...)
	return this
}

func (this *MultipleSiteGetter) Prepend(g ...SiteGetter) *MultipleSiteGetter {
	(*this) = append(g[:], (*this)...)
	return this
}

type SiteGetterImpl struct {
	GetFunc          func(site *Site, key interface{}) (value interface{}, ok bool)
	GetInterfaceFunc func(site *Site, key, dest interface{}) (ok bool)
}

func (f *SiteGetterImpl) Get(site *Site, key interface{}) (value interface{}, ok bool) {
	return f.GetFunc(site, key)
}

func (f *SiteGetterImpl) GetInterface(site *Site, key, dest interface{}) (ok bool) {
	return f.GetInterfaceFunc(site, key, dest)
}

func NewSiteGetter(getter func(site *Site, key interface{}) (value interface{}, ok bool), ifGetter func(site *Site, key, dest interface{}) (ok bool)) SiteGetter {
	return &SiteGetterImpl{getter, ifGetter}
}

func MapstructureRawGetter2InterfaceGetter(getter getters.Getter, errorcb func(key, value, dest interface{}, err error)) *getters.InterfaceGetterImpl {
	return &getters.InterfaceGetterImpl{
		getter,
		func(key, dest interface{}) (ok bool) {
			var v interface{}
			if v, ok = getter.Get(key); !ok {
				return
			}

			err := mapstructure.Decode(v, dest)
			if err != nil {
				errorcb(key, v, dest, err)
			}
			return
		},
	}
}
