package safe

import (
	"sync"
)

// Map is a thread-safe generic map for concurrent reads/writes.
type Map[K comparable, V any] struct {
	m  map[K]V      // Underlying Go map
	mu sync.RWMutex // Protects access to the map
}

// NewMap creates and returns a new thread-safe Map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{m: make(map[K]V)}
}

// Set stores or replaces a key-value pair (exclusive lock).
func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[key] = value
}

// Get retrieves a value and existence flag (read lock).
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, found := m.m[key]

	return value, found
}

// Update atomically reads, modifies, and writes a value.
func (m *Map[K, V]) Update(key K, fn func(value V, found bool) (newValue V, do bool)) (newValue V, done bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.m[key]

	newValue, do := fn(value, found)
	if do {
		m.m[key] = newValue
	}

	return newValue, do
}

// Delete removes a key if fn allows, or unconditionally if fn is nil.
func (m *Map[K, V]) Delete(key K, fn func(value V, found bool) (do bool)) (done bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if fn == nil {
		delete(m.m, key)

		return true
	}

	value, found := m.m[key]

	do := fn(value, found)
	if do {
		delete(m.m, key)
	}

	return do
}

// Exists returns true if the key exists (read lock).
func (m *Map[K, V]) Exists(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, found := m.m[key]

	return found
}

// Len returns the number of items in the map (read lock).
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.m)
}

// RangeGet iterates with read lock; stops if fn returns true.
func (m *Map[K, V]) RangeGet(fn func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for key, value := range m.m {
		stop := fn(key, value)
		if stop {
			break
		}
	}
}

// RangeUpdate iterates and updates values (exclusive lock).
func (m *Map[K, V]) RangeUpdate(fn func(key K, value V) (newValue V, do, stop bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, value := range m.m {
		newValue, do, stop := fn(key, value)
		if do {
			m.m[key] = newValue
		}

		if stop {
			break
		}
	}
}

// RangeDelete iterates and conditionally deletes keys (exclusive lock).
func (m *Map[K, V]) RangeDelete(fn func(key K, value V) (do, stop bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, value := range m.m {
		do, stop := fn(key, value)
		if do {
			delete(m.m, key)
		}

		if stop {
			break
		}
	}
}
