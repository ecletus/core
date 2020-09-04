package core

type SiteGetter interface {
	Get(site *Site, key interface{}) (value interface{}, ok bool)
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

func (this *MultipleSiteGetter) Append(g ...SiteGetter) *MultipleSiteGetter {
	(*this) = append((*this), g...)
	return this
}

func (this *MultipleSiteGetter) Prepend(g ...SiteGetter) *MultipleSiteGetter {
	(*this) = append(g[:], (*this)...)
	return this
}

type SiteGetterFunc func(site *Site, key interface{}) (value interface{}, ok bool)

func (f SiteGetterFunc) Get(site *Site, key interface{}) (value interface{}, ok bool) {
	return f(site, key)
}

func NewSiteGetter(getter func(site *Site, key interface{}) (value interface{}, ok bool)) SiteGetter {
	return SiteGetterFunc(getter)
}
