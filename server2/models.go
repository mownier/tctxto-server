package server2

import (
	"sync"
)

type safeMap[K comparable, V any] struct {
	data map[K]V
	mu   sync.RWMutex
}

func (m *safeMap[K, V]) get(k K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, e := m.data[k]
	return v, e
}

func (m *safeMap[K, V]) set(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[k] = v
}

func (m *safeMap[K, V]) delete(k K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, k)
}

func (m *safeMap[K, V]) forEach(f func(k K, v V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.data {
		if !f(k, v) {
			break
		}
	}
}

func newSafeMapWith[K comparable, V any](data map[K]V) *safeMap[K, V] {
	return &safeMap[K, V]{
		data: data,
	}
}

func newSafeMap[K comparable, V any]() *safeMap[K, V] {
	return &safeMap[K, V]{
		data: make(map[K]V),
	}
}
