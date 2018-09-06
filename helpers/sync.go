package helpers

import (
	"sync"
)

type SyncMap struct {
	sync.Map
}

func (m *SyncMap) LoadOrFactory(key interface{}, factory func() interface{}) interface{} {
	if v, ok := m.Load(key); ok {
		return v
	}
	v := factory()
	m.Store(key, v)
	return v
}
