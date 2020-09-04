package core

import (
	"sync"

	"github.com/moisespsena-go/getters"

	"github.com/go-errors/errors"
)

var (
	ErrDuplicateSiteHost = errors.New("duplicate site host")
	ErrDuplicateSitePath = errors.New("duplicate site path")
	ErrSiteNotFound      = errors.New("site not found")
	ErrSiteFound         = errors.New("site found")
)

type SitesRegister struct {
	Alone                   bool
	ByName                  SitesMap
	ByHost                  SitesMap
	ByPath                  SitesMap
	AddedCallbacks          []func(site *Site)
	PostAddedCallbacks      []func(site *Site)
	DeletedCallbacks        []func(site *Site)
	HostAddedCallbacks      []func(site *Site, host string)
	HostDeletedCallbacks    []func(site *Site, host string)
	PathAddedCallbacks      []func(site *Site, pth string)
	PathDeletedCallbacks    []func(site *Site, pth string)
	SiteConfigGetter        MultipleSiteGetter
	SiteConfigSetterFactory SiteConfigSetterFacotry
	mu                      sync.RWMutex
}

func (this *SitesRegister) OnAdd(f ...func(site *Site)) *SitesRegister {
	this.AddedCallbacks = append(this.AddedCallbacks, f...)
	if this.ByName != nil {
		for _, site := range this.ByName {
			for _, f := range f {
				f(site)
			}
		}
	}
	return this
}

func (this *SitesRegister) OnPostAdd(f ...func(site *Site)) *SitesRegister {
	this.PostAddedCallbacks = append(this.PostAddedCallbacks, f...)
	if this.ByName != nil {
		for _, site := range this.ByName {
			for _, f := range f {
				f(site)
			}
		}
	}
	return this
}

func (this *SitesRegister) Destroy() (err error) {
	return this.DestroySite(this.ByName.Names()...)
}

func (this *SitesRegister) OnSiteDestroy(f ...func(site *Site)) *SitesRegister {
	this.DeletedCallbacks = append(this.DeletedCallbacks, f...)
	return this
}

func (this *SitesRegister) OnHostAdd(f ...func(site *Site, host string)) *SitesRegister {
	this.HostAddedCallbacks = append(this.HostAddedCallbacks, f...)
	if this.ByHost != nil {
		for host, site := range this.ByHost {
			for _, f := range f {
				f(site, host)
			}
		}
	}
	return this
}

func (this *SitesRegister) OnHostDel(f ...func(site *Site, host string)) *SitesRegister {
	this.HostDeletedCallbacks = append(this.HostDeletedCallbacks, f...)
	return this
}

func (this *SitesRegister) OnPathAdd(f ...func(site *Site, path string)) *SitesRegister {
	this.PathAddedCallbacks = append(this.PathAddedCallbacks, f...)
	if this.ByPath != nil {
		for path, site := range this.ByPath {
			for _, f := range f {
				f(site, path)
			}
		}
	}
	return this
}

func (this *SitesRegister) OnPathDel(f ...func(site *Site, path string)) *SitesRegister {
	this.PathDeletedCallbacks = append(this.PathDeletedCallbacks, f...)
	return this
}

func (this *SitesRegister) Get(name string) (site *Site, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.ByName.Get(name)
}

func (this *SitesRegister) MustGet(name string) (site *Site) {
	site, _ = this.Get(name)
	return
}

func (this *SitesRegister) Has(name string) (ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.ByName.Has(name)
}

func (this *SitesRegister) GetByHost(host string) (site *Site, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.ByHost.Get(host)
}

func (this *SitesRegister) GetByPath(path string) (site *Site, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.ByPath.Get(path)
}

func (this *SitesRegister) AddHost(siteName, host string) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.ByHost.Has(host) {
		return ErrDuplicateSiteHost
	}
	var (
		site *Site
		ok   bool
	)
	if site, ok = this.ByName.Get(siteName); !ok {
		return ErrSiteNotFound
	}
	this.ByHost.Set(host, site)
	for _, f := range this.HostAddedCallbacks {
		f(site, host)
	}
	return nil
}

func (this *SitesRegister) DelHost(host string) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	site, err := this.ByHost.Del(host)
	if err == nil {
		for _, f := range this.HostDeletedCallbacks {
			f(site, host)
		}
	}
	return err
}

func (this *SitesRegister) AddPath(siteName, path string) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.ByPath.Has(path) {
		return ErrDuplicateSitePath
	}
	var (
		site *Site
		ok   bool
	)
	if site, ok = this.ByName.Get(siteName); !ok {
		return ErrSiteNotFound
	}
	this.ByPath.Set(path, site)
	for _, f := range this.PathAddedCallbacks {
		f(site, path)
	}
	return nil
}

func (this *SitesRegister) DelPath(siteName, path string) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	site, err := this.ByPath.Del(path)
	if err == nil {
		for _, f := range this.PathDeletedCallbacks {
			f(site, path)
		}
	}
	return err
}

func (this *SitesRegister) Rename(oldName, newName string) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	site, err := this.ByName.Del(oldName)
	if err != nil {
		return err
	}
	this.ByName.Set(newName, site)
	return nil
}

func (this *SitesRegister) HasSites() bool {
	return len(this.ByName) > 0
}

func (this *SitesRegister) Site() *Site {
	if this.Alone {
		for _, site := range this.ByName {
			return site
		}
	}
	return nil
}

func (this *SitesRegister) Add(site *Site) (err error) {
	if this.Alone && this.HasSites() {
		return errors.New("register site: alone mode accept only one site")
	}
	defer func() {
		if err != nil {
			return
		}

		for _, f := range this.PostAddedCallbacks {
			f(site)
		}
	}()
	this.mu.Lock()
	defer this.mu.Unlock()

	if !site.IsRegistered() {
		var configGetter getters.MultipleGetter
		configGetter.Append(getters.New(func(key interface{}) (value interface{}, ok bool) {
			return this.SiteConfigGetter.Get(site, key)
		}))
		if site.configGetter != nil {
			configGetter.Append(site.configGetter)
		}
		site.configGetter = configGetter

		if site.ConfigSetter == nil && this.SiteConfigSetterFactory != nil {
			site.ConfigSetter = this.SiteConfigSetterFactory.Factory(site)
		}

		defer func() {
			if err == nil {
				site.initLogger()
				site.registered = true
			}
		}()
	}

	if this.ByName.Has(site.name) {
		err = ErrSiteFound
		return
	}

	this.ByName.Set(site.name, site)
	for _, f := range this.AddedCallbacks {
		f(site)
	}
	return
}

func (this *SitesRegister) Only(name string, f func(site *Site) error) error {
	sites := this.ByName.Copy()

	for sname := range sites {
		if name != sname {
			if err := this.DestroySite(sname); err != nil {
				return err
			}
		}
	}

	if err := f(sites[name]); err != nil {
		return err
	}

	for sname, site := range sites {
		if name != sname {
			if err := this.Add(site); err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *SitesRegister) DestroySite(name ...string) (err error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	var site *Site
	for _, name := range name {
		site, err = this.ByName.Del(name)
		if err != nil {
			return
		}

		for _, onDel := range site.onDestroyCallbacks {
			onDel()
		}

		site.onDestroyCallbacks = nil

		if this.ByHost != nil {
			for host, s := range this.ByHost {
				if s == site {
					this.ByHost.Del(host)
					for _, f := range this.HostDeletedCallbacks {
						f(site, host)
					}
				}
			}
		}

		if this.ByPath != nil {
			for pth, s := range this.ByPath {
				if s == site {
					this.ByPath.Del(pth)
					for _, f := range this.PathDeletedCallbacks {
						f(site, pth)
					}
				}
			}
		}

		for _, f := range this.DeletedCallbacks {
			f(site)
		}
	}
	return
}

func (this *SitesRegister) Reader() SitesMap {
	return this.ByName
}
