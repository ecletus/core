package core

type Configor interface {
	ConfigSet(key, value interface{})
	ConfigGet(key interface{}) (value interface{}, ok bool)
}

type Option interface {
	Apply(configor Configor)
}

type OptionFunc func(configor Configor)

func (this OptionFunc) Apply(configor Configor) {
	this(configor)
}

type Configors []Configor

func (c Configors) ConfigSet(key, value interface{}) {
	c[0].ConfigSet(key, value)
}

func (c Configors) ConfigGet(key interface{}) (value interface{}, ok bool) {
	for _, c := range c {
		if value, ok = c.ConfigGet(key); ok {
			return
		}
	}
	return
}
