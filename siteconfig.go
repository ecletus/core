package core

import "github.com/moisespsena-go/getters"

type ConfigSetter interface {
	Set(key interface{}, value interface{}) (err error)
	Destroy()
}

type ConfigSetterFunc func(key interface{}, value interface{}) (err error)

func (this ConfigSetterFunc) Set(key interface{}, value interface{}) (err error) {
	return this(key, value)
}

type DefaultConfigSetter struct {
	SetterFunc  func(key interface{}, value interface{}) (err error)
	DestroyFunc func()
}

func (this *DefaultConfigSetter) Set(key interface{}, value interface{}) (err error) {
	return this.SetterFunc(key, value)
}

func (this *DefaultConfigSetter) Destroy() {
	this.DestroyFunc()
}

type SiteFactoryCallback struct {
	Setup   func(site *Site, setter ConfigSetter)
	Destroy func(site *Site)
}

type SiteConfigSetterFacotry interface {
	Factory(site *Site) (setter ConfigSetter)
	FactoryCallback(cb ...*SiteFactoryCallback)
}

type SiteConfig interface {
	ConfigGetter
	ConfigSetter
}

type ConfigGetter interface {
	getters.Getter
	GetInterface(key, dest interface{}) (ok bool)
}
