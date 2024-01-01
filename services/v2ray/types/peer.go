package types

import (
	"sync"
)

// Peer represents an entity with an Email field.
type Peer struct {
	Email string
}

// Key returns the unique identifier (email) associated with the Peer.
func (p *Peer) Key() string {
	return p.Email
}

// Peers is a thread-safe map-like structure that stores Peer objects.
type Peers struct {
	*sync.RWMutex
	m map[string]*Peer
}

// NewPeers creates and returns a new instance of Peers.
func NewPeers() *Peers {
	return &Peers{
		RWMutex: &sync.RWMutex{},
		m:       make(map[string]*Peer),
	}
}

// Get retrieves a Peer from Peers based on the provided key.
func (p *Peers) Get(v string) *Peer {
	p.RLock()
	defer p.RUnlock()

	value, ok := p.m[v]
	if !ok {
		return nil
	}

	return value
}

// Put adds a Peer to Peers.
func (p *Peers) Put(v *Peer) {
	p.Lock()
	defer p.Unlock()

	_, ok := p.m[v.Key()]
	if ok {
		return
	}

	p.m[v.Key()] = v
}

// Delete removes a Peer from Peers based on the provided key.
func (p *Peers) Delete(v string) {
	p.Lock()
	defer p.Unlock()

	delete(p.m, v)
}

// Len returns the number of elements in Peers.
func (p *Peers) Len() int {
	p.RLock()
	defer p.RUnlock()

	return len(p.m)
}

// Iterate iterates over each element in Peers and applies the provided function.
// If the function returns true, the iteration stops.
func (p *Peers) Iterate(fn func(key string, value *Peer) (bool, error)) error {
	p.RLock()
	defer p.RUnlock()

	for key, value := range p.m {
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
