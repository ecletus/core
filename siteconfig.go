package core

import "github.com/moisespsena-go/getters"

type ConfigSetter interface {
	Set(key interface{}, value interface{}) (err error)
}

type ConfigSetterFunc func(key interface{}, value interface{}) (err error)

func (this ConfigSetterFunc) Set(key interface{}, value interface{}) (err error) {
	return this(key, value)
}

type SiteConfigSetterFacotry interface {
	Factory(site *Site) (setter ConfigSetter)
}


type SiteConfig interface {
	getters.Getter
	ConfigSetter
}
