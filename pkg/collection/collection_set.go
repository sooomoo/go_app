package collection

import (
	"goapp/pkg/core"
	"sync"
)

// 并发安全的 Set
type Set[T comparable] struct {
	underlying map[T]core.Empty
	lock       sync.RWMutex
}

// O(n)
func (s *Set[T]) Add(items ...T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.underlying == nil {
		s.underlying = map[T]core.Empty{}
	}
	for _, v := range items {
		s.underlying[v] = core.Empty{}
	}
}

// O(n)
func (s *Set[T]) Remove(items ...T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.underlying) == 0 {
		return
	}

	for _, v := range items {
		delete(s.underlying, v)
	}
}

// O(1)
func (s *Set[T]) Clear() {
	s.lock.Lock()
	defer s.lock.Unlock()

	clear(s.underlying)
}

// O(n)
func (s *Set[T]) Size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.underlying)
}

// O(n)
func (s *Set[T]) IsEmpty() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.underlying) == 0
}

// O(n)
func (s *Set[T]) ToSlice() []T {
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.underlying) == 0 {
		return false
	}

	_, ok := s.underlying[item]
	return ok
}
