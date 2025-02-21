package wireguard

import (
	"errors"
	"fmt"
	"net/netip"
	"sync"

	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// Peer represents a network peer with identity and IP addresses.
type Peer struct {
	ID    string // ID of the peer
	Addrs []netip.Prefix
}

// Key returns the identity of the peer as the key.
func (p *Peer) Key() string {
	return p.ID
}

// PeerManager manages a collection of Peers and their associated IP addresses.
type PeerManager struct {
	m     map[string]*Peer
	pools []*types.IPPool
	rwm   *sync.RWMutex
}

// NewPeerManager creates a new instance of PeerManager.
func NewPeerManager(pools ...*types.IPPool) *PeerManager {
	return &PeerManager{
		m:     make(map[string]*Peer),
		pools: pools,
		rwm:   &sync.RWMutex{},
	}
}

// Get retrieves a Peer from the PeerManager by its identity.
func (m *PeerManager) Get(v string) *Peer {
	m.rwm.RLock()
	defer m.rwm.RUnlock()

	return m.m[v]
}

// Put adds a new Peer with the given identity to the PeerManager.
// It assigns available IPv4 and IPv6 addresses to the Peer.
func (m *PeerManager) Put(id string) (addrs []netip.Prefix, err error) {
	m.rwm.Lock()
	defer m.rwm.Unlock()

	if id == "" {
		return nil, errors.New("peer id is empty")
	}

	// Check if the Peer already exists
	if _, ok := m.m[id]; ok {
		return nil, fmt.Errorf("peer %s already exists", id)
	}

	defer func() {
		if len(addrs) != len(m.pools) {
			for i := 0; i < len(addrs); i++ {
				addr := addrs[i].Addr()
				if err := m.pools[i].Put(addr); err != nil {
					panic(fmt.Errorf("failed to put addr %s to pool: %w", addr, err))
				}
			}
		}
	}()

	for _, pool := range m.pools {
		addr, err := pool.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to get addr from pool: %w", err)
		}

		b := 32
		if addr.Is6() {
			b = 128
		}

		prefixAddr, err := addr.Prefix(b)
		if err != nil {
			return nil, fmt.Errorf("failed to get prefix addr: %w", err)
		}

		addrs = append(addrs, prefixAddr)
	}

	// Create and store the new Peer
	m.m[id] = &Peer{
		ID:    id,
		Addrs: addrs,
	}

	return addrs, nil
}

// Delete removes a Peer from the PeerManager by its identity.
func (m *PeerManager) Delete(v string) {
	m.rwm.Lock()
	defer m.rwm.Unlock()

	// Retrieve the Peer and its IP addresses
	item, ok := m.m[v]
	if !ok {
		return
	}

	for i := 0; i < len(item.Addrs); i++ {
		addr := item.Addrs[i].Addr()
		if err := m.pools[i].Put(addr); err != nil {
			panic(fmt.Errorf("failed to put addr %s to pool: %w", addr, err))
		}
	}

	// Remove the Peer from the PeerManager
	delete(m.m, v)
}

// Len returns the number of Peers in the PeerManager.
func (m *PeerManager) Len() int {
	m.rwm.RLock()
	defer m.rwm.RUnlock()

	return len(m.m)
}

// Iterate iterates over each Peer in the PeerManager and applies the provided function.
// If the function returns true, the iteration stops.
// If the function returns an error, the iteration stops and the error is returned.
func (m *PeerManager) Iterate(fn func(key string, value *Peer) (bool, error)) error {
	m.rwm.RLock()
	defer m.rwm.RUnlock()

	for key, value := range m.m {
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
