package safe

import (
	"sync"
)

// Map is a thread-safe generic map with concurrent read and write support.
type Map[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewMap returns a new instance of a thread-safe Map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{m: make(map[K]V)}
}

// Set stores the given key-value pair in the map, replacing any existing value.
func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[key] = value
}

// Get retrieves the value associated with the given key.
// It returns the value and a boolean indicating whether the key was found.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.m[key]
	return val, ok
}

// Delete removes the key-value pair for the given key, if it exists.
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.m, key)
}

// Exists returns true if the key exists in the map, otherwise false.
func (m *Map[K, V]) Exists(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.m[key]
	return ok
}

// Len returns the current number of key-value pairs in the map.
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.m)
}

// Range iterates over all key-value pairs in the map and calls fn for each.
// If fn returns (true, nil), the iteration stops early.
// If fn returns an error, the iteration stops and the error is returned.
func (m *Map[K, V]) Range(fn func(key K, value V) (bool, error)) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for key, val := range m.m {
		stop, err := fn(key, val)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// Update performs an atomic read-modify-write on the given key.
// The callback fn is called with the current value (if present) and a boolean indicating
// whether the key exists. It must return the new value, which will be stored in the map.
// The method returns the updated value.
func (m *Map[K, V]) Update(key K, fn func(val V, ok bool) V) V {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.m[key]
	val = fn(val, ok)
	m.m[key] = val

	return val
}
