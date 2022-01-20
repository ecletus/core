package resource

import (
	"reflect"

	"github.com/moisespsena/template"
)

type SliceValue struct {
	Current    interface{}
	Deleted    interface{}
	DeletedID  []ID
	deletedMap map[string]bool
}

func (s *SliceValue) DeletedMap() map[string]bool {
	if s.deletedMap == nil {
		s.deletedMap = map[string]bool{}
		for _, id := range s.DeletedID {
			s.deletedMap[id.String()] = true
		}
	}
	return s.deletedMap
}

func (s *SliceValue) IsDeleted(id ID) bool {
	if s.deletedMap != nil {
		_, ok := s.deletedMap[id.String()]
		return ok
	}
	for _, del := range s.DeletedID {
		if del.Eq(id) {
			return true
		}
	}
	return false
}

func (s *SliceValue) Iterator() template.Iterator {
	it := &SliceValueIterator{}
	if s.Current != nil {
		it.Slices = append(it.Slices, reflect.ValueOf(s.Current))
	}
	if s.Deleted != nil {
		it.Slices = append(it.Slices, reflect.ValueOf(s.Deleted))
	}
	return it
}

func (s *SliceValue) Len() (l int) {
	if s.Current != nil {
		l = reflect.ValueOf(s.Current).Len()
	}
	if s.Deleted != nil {
		l += reflect.ValueOf(s.Deleted).Len()
	}
	return l
}

type SliceValueIterator struct {
	next, nextItem int
	Slices         []reflect.Value
}

func (s *SliceValueIterator) Start() (state interface{}) {
	if len(s.Slices) == 0 {
		return [3]int{0, 0, 0}
	}
	return [3]int{0, 0, s.Slices[0].Len()}
}

func (s *SliceValueIterator) Done(state interface{}) bool {
	if state == nil {
		return true
	}
	var st = state.([3]int)
	if st[0] == len(s.Slices) {
		return true
	}
	return false
}

func (s *SliceValueIterator) Next(state interface{}) (item, nextState interface{}) {
	var (
		st        = state.([3]int)
		si, ei, l = st[0], st[1], st[2]
	)

	item = s.Slices[si].Index(ei).Interface()
	ei++

	if ei == l {
		si++
		ei = 0
		if si < len(s.Slices) {
			l = s.Slices[si].Len()
		}
	}
	nextState = [3]int{si, ei, l}
	return
}
