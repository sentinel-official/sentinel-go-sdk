package v2ray

import (
	"sync"
)

// Peer represents an entity with an Email field.
type Peer struct {
	Email string // Email uniquely identifies the Peer
}

// Key returns the unique identifier (email) associated with the Peer.
func (p *Peer) Key() string {
	return p.Email
}

// PeerManager is a thread-safe map-like structure that stores Peer objects.
type PeerManager struct {
	*sync.RWMutex                  // Read-write mutex for safe concurrent access
	m             map[string]*Peer // Map storing Peers indexed by their keys
}

// NewPeerManager creates and returns a new instance of PeerManager.
func NewPeerManager() *PeerManager {
	return &PeerManager{
		RWMutex: &sync.RWMutex{},
		m:       make(map[string]*Peer),
	}
}

// Get retrieves a Peer from the PeerManager based on the provided key.
// It returns the corresponding Peer if found, or nil otherwise.
func (pm *PeerManager) Get(v string) *Peer {
	pm.RLock()
	defer pm.RUnlock()

	value, ok := pm.m[v]
	if !ok {
		return nil
	}

	return value
}

// Put adds a Peer to the PeerManager.
// If a Peer with the same key already exists, it does nothing.
func (pm *PeerManager) Put(id string) {
	pm.Lock()
	defer pm.Unlock()

	_, ok := pm.m[id]
	if ok {
		return
	}

	pm.m[id] = &Peer{
		Email: id,
	}
}

// Delete removes a Peer from the PeerManager based on the provided key.
func (pm *PeerManager) Delete(v string) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.m, v)
}

// Len returns the number of Peers in the PeerManager.
func (pm *PeerManager) Len() int {
	pm.RLock()
	defer pm.RUnlock()

	return len(pm.m)
}

// Iterate iterates over each element in the PeerManager and applies the provided function.
// If the function returns true, the iteration stops.
// If the function returns an error, the iteration stops and the error is returned.
func (pm *PeerManager) Iterate(fn func(key string, value *Peer) (bool, error)) error {
	pm.RLock()
	defer pm.RUnlock()

	for key, value := range pm.m {
		stop, err := fn(key, value)
		if err != nil {
			return err
		}

		if stop {
			return nil
		}
	}

	return nil
}
