package core

import (
	"fmt"
	"sort"

	errwrap "github.com/moisespsena-go/error-wrap"
)

type SitesMap map[string]*Site

func (m *SitesMap) Set(key string, value *Site) {
	if *m == nil {
		*m = map[string]*Site{
			key: value,
		}
	} else {
		(*m)[key] = value
	}
}

func (m SitesMap) Copy() (copy SitesMap) {
	copy = SitesMap{}
	if m == nil {
		return
	}
	for k, v := range m {
		copy[k] = v
	}
	return
}

func (m SitesMap) Has(key ...string) (ok bool) {
	if m == nil {
		return
	}

	for _, key := range key {
		if _, ok = m[key]; !ok {
			return false
		}
	}
	return
}

func (m SitesMap) MustGet(key string) (site *Site) {
	if m == nil {
		return
	}
	return m[key]
}

func (m SitesMap) Get(key string) (site *Site, ok bool) {
	if m == nil {
		return
	}
	site, ok = m[key]
	return
}

func (m *SitesMap) Del(key string) (site *Site, err error) {
	if *m == nil {
		return nil, ErrSiteNotFound
	}
	var ok bool
	if site, ok = (*m)[key]; !ok {
		return nil, ErrSiteNotFound
	}
	delete(*m, key)
	return
}

func (r SitesMap) GetOrError(siteName string) (*Site, error) {
	s, ok := r[siteName]
	if !ok {
		return nil, fmt.Errorf("Site %q does not exists.", siteName)
	}
	return s, nil
}

func (r SitesMap) All() (sites []*Site) {
	for _, s := range r {
		sites = append(sites, s)
	}
	return
}

func (r SitesMap) Keys() (keys []string) {
	for k := range r {
		keys = append(keys, k)
	}
	return
}

func (r SitesMap) Names() (names []string) {
	for _, site := range r {
		names = append(names, site.Name())
	}
	return
}

func (r SitesMap) Sorted() []*Site {
	sites := r.All()
	sort.Slice(sites, func(a, b int) bool {
		return sites[a].Name() < sites[b].Name()
	})
	return sites
}

func (r SitesMap) Each(cb func(site *Site) (err error)) (err error) {
	for _, s := range r {
		if err = cb(s); err != nil {
			if err == StopSiteIteration {
				return nil
			}
			return errwrap.Wrap(err, "Site %q", s.Name())
		}
	}
	return nil
}

func (r SitesMap) EachOrAll(siteName string, cb func(site *Site) (err error)) error {
	if siteName == "" || siteName == "*" {
		return r.Each(cb)
	}

	site, err := r.GetOrError(siteName)

	if err != nil {
		return err
	}

	return cb(site)
}
