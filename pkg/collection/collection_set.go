package collection

import (
	"goapp/pkg/core"
	"sync"
)

type Set[T comparable] struct {
	underlying map[T]core.Empty
	lock       sync.RWMutex
	once       sync.Once
}

func (s *Set[T]) ensureInit() {
	s.once.Do(func() {
		s.underlying = map[T]core.Empty{}
	})
}

// O(1)~O(n)
func (s *Set[T]) Add(item T) {
	s.ensureInit()
	s.lock.Lock()
	defer s.lock.Unlock()

	s.underlying[item] = core.Empty{}
}

// O(n)
func (s *Set[T]) AddRange(items ...T) {
	s.ensureInit()
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, v := range items {
		s.underlying[v] = core.Empty{}
	}
}

// O(1)~O(n)
func (s *Set[T]) Remove(item T) {
	s.ensureInit()
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.underlying, item)
}

// O(1)
func (s *Set[T]) Clear() {
	s.ensureInit()
	s.lock.Lock()
	defer s.lock.Unlock()

	clear(s.underlying)
}

// O(n)
func (s *Set[T]) Size() int {
	s.ensureInit()
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.underlying)
}

// O(n)
func (s *Set[T]) IsEmpty() bool {
	s.ensureInit()
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.underlying) == 0
}

// O(n)
func (s *Set[T]) ToSlice() []T {
	s.ensureInit()
	s.lock.RLock()
	defer s.lock.RUnlock()

	out := []T{}
	for k := range s.underlying {
		out = append(out, k)
	}
	return out
}

// O(1)~O(n)
func (s *Set[T]) Contains(item T) bool {
	s.ensureInit()
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, ok := s.underlying[item]
	return ok
}
