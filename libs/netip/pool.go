package netip

import (
	"fmt"
	"net/netip"
	"sync"
)

// AddrPool manages a pool of IP addresses, including assigned, reserved, and unassigned addresses.
// It ensures thread-safe operations and manages address allocation and deallocation.
type AddrPool struct {
	assigned   map[netip.Addr]bool // Tracks currently allocated IPs.
	reserved   map[netip.Addr]bool // Tracks permanently reserved IPs.
	unassigned map[netip.Addr]bool // Tracks available (released) IPs for quick reuse.

	addr   netip.Addr // Next candidate IP address to try during allocation via Acquire.
	prefix *Prefix    // Network prefix defining the pool range.

	rwm *sync.RWMutex // Mutex to ensure thread-safe access to pool data structures.
}

// NewAddrPool parses a CIDR string into a Prefix and initializes an AddrPool.
// It also reserves the base, network, and (for IPv4) broadcast addresses.
func NewAddrPool(cidr string) (*AddrPool, *Prefix, error) {
	prefix, err := NewPrefix(cidr)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CIDR %q: %w", cidr, err)
	}

	p := &AddrPool{
		assigned:   make(map[netip.Addr]bool),
		reserved:   make(map[netip.Addr]bool),
		unassigned: make(map[netip.Addr]bool),
		addr:       prefix.NetworkAddr(),
		prefix:     prefix,
		rwm:        &sync.RWMutex{},
	}

	// Reserve the prefix's base and network addresses.
	_ = p.Reserve(prefix.Addr())
	_ = p.Reserve(prefix.NetworkAddr())

	// Reserve the broadcast address for IPv4 if applicable.
	if p.addr.Is4() {
		broadcast, err := prefix.BroadcastAddr()
		if err != nil {
			return nil, nil, fmt.Errorf("getting broadcast addr from prefix %q: %w", prefix, err)
		}

		_ = p.Reserve(broadcast)
	}

	return p, prefix, nil
}

// Contains checks if the provided address belongs to the pool's prefix range.
func (p *AddrPool) Contains(addr netip.Addr) bool {
	p.rwm.RLock()
	defer p.rwm.RUnlock()

	return p.prefix.Contains(addr)
}

// Assign marks a specific IP address as assigned if it's valid and unassigned.
func (p *AddrPool) Assign(addr netip.Addr) error {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	if !p.prefix.Contains(addr) {
		return fmt.Errorf("cannot assign %q: outside of prefix", addr)
	}
	if p.reserved[addr] {
		return fmt.Errorf("cannot assign %q: addr is reserved", addr)
	}
	if p.assigned[addr] {
		return fmt.Errorf("cannot assign %q: addr is already assigned", addr)
	}

	delete(p.unassigned, addr)
	p.assigned[addr] = true

	return nil
}

// Reserve marks an address as permanently unavailable for assignment.
func (p *AddrPool) Reserve(addr netip.Addr) error {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	if !p.prefix.Contains(addr) {
		return fmt.Errorf("cannot reserve %q: outside of prefix", addr)
	}
	if p.assigned[addr] {
		return fmt.Errorf("cannot reserve %q: addr is assigned", addr)
	}
	if p.reserved[addr] {
		return fmt.Errorf("cannot reserve %q: addr is already reserved", addr)
	}

	delete(p.unassigned, addr)
	p.reserved[addr] = true

	return nil
}

// Acquire retrieves the next available unassigned IP address.
// Priority is given to any addresses that were previously released (unassigned).
// Otherwise, it linearly scans the prefix from the current candidate (p.addr).
func (p *AddrPool) Acquire() (addr netip.Addr, err error) {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	// First attempt: use any pre-released unassigned addresses.
	for addr := range p.unassigned {
		delete(p.unassigned, addr)
		p.assigned[addr] = true

		return addr, nil
	}

	// Second attempt: scan the prefix sequentially.
	for {
		if !p.prefix.Contains(p.addr) {
			return netip.Addr{}, fmt.Errorf("address pool %s exhausted", p.prefix)
		}

		addr, p.addr = p.addr, p.addr.Next()
		if !p.assigned[addr] && !p.reserved[addr] {
			break
		}
	}

	p.assigned[addr] = true
	return addr, nil
}

// Release moves an assigned address back into the unassigned pool.
func (p *AddrPool) Release(addr netip.Addr) error {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	if !p.prefix.Contains(addr) {
		return fmt.Errorf("cannot release %q: outside of prefix", addr)
	}
	if p.reserved[addr] {
		return fmt.Errorf("cannot release %q: addr is reserved", addr)
	}
	if !p.assigned[addr] {
		return fmt.Errorf("cannot release %q: addr was not assigned", addr)
	}

	delete(p.assigned, addr)
	p.unassigned[addr] = true

	return nil
}

// AddrPoolSet manages multiple AddrPools and prevents prefix overlaps.
type AddrPoolSet struct {
	pools []*AddrPool
	rwm   *sync.RWMutex
}

// NewAddrPoolSet constructs an AddrPoolSet from a list of CIDR strings.
// It fails if any of the prefixes overlap.
func NewAddrPoolSet(cidrs ...string) (*AddrPoolSet, error) {
	p := &AddrPoolSet{
		pools: []*AddrPool{},
		rwm:   &sync.RWMutex{},
	}

	var items []*Prefix
	for _, cidr := range cidrs {
		pool, prefix, err := NewAddrPool(cidr)
		if err != nil {
			return nil, fmt.Errorf("creating add pool for CIDR %q: %w", cidr, err)
		}

		// Ensure no overlaps between prefixes.
		for _, item := range items {
			if prefix.Overlaps(item) {
				return nil, fmt.Errorf("prefix %q overlaps with %q", prefix, item)
			}
		}

		items = append(items, prefix)
		p.pools = append(p.pools, pool)
	}

	return p, nil
}

// Contains returns true if any pool contains the given address.
func (p *AddrPoolSet) Contains(addr netip.Addr) bool {
	p.rwm.RLock()
	defer p.rwm.RUnlock()

	for _, pool := range p.pools {
		if pool.Contains(addr) {
			return true
		}
	}

	return false
}

// Assign marks multiple addresses as assigned in their respective pools.
func (p *AddrPoolSet) Assign(addrs ...netip.Addr) error {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	if len(addrs) != len(p.pools) {
		return fmt.Errorf("addr count mismatch: got %d, expected %d", len(addrs), len(p.pools))
	}

	for _, addr := range addrs {
		assigned := false
		for _, pool := range p.pools {
			if pool.Contains(addr) {
				if err := pool.Assign(addr); err != nil {
					return fmt.Errorf("assigning addr %q: %w", addr, err)
				}

				assigned = true
				break
			}
		}

		if !assigned {
			return fmt.Errorf("cannot assign %q: pool not found", addr)
		}
	}

	return nil
}

// Reserve marks a specific address as reserved in the correct pool.
func (p *AddrPoolSet) Reserve(addr netip.Addr) error {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	for _, pool := range p.pools {
		if pool.Contains(addr) {
			return pool.Reserve(addr)
		}
	}

	return fmt.Errorf("cannot reserve %q: pool not found", addr)
}

// Acquire acquires one address from each pool.
// If any pool fails, previously acquired addresses are rolled back.
func (p *AddrPoolSet) Acquire() (addrs []netip.Addr, err error) {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	// Rollback logic: release any acquired addresses on error.
	defer func() {
		if len(addrs) != len(p.pools) {
			for i := 0; i < len(addrs); i++ {
				if err := p.pools[i].Release(addrs[i]); err != nil {
					panic(fmt.Errorf("rollback failed for addr %q: %w", addrs[i], err))
				}
			}
		}
	}()

	for _, pool := range p.pools {
		addr, err := pool.Acquire()
		if err != nil {
			return nil, fmt.Errorf("acquiring addr from pool: %w", err)
		}

		addrs = append(addrs, addr)
	}

	return addrs, nil
}

// Release returns the acquired addresses to their corresponding pools.
func (p *AddrPoolSet) Release(addrs []netip.Addr) error {
	p.rwm.Lock()
	defer p.rwm.Unlock()

	if len(addrs) != len(p.pools) {
		return fmt.Errorf("addr count mismatch: got %d, expected %d", len(addrs), len(p.pools))
	}

	for _, addr := range addrs {
		released := false
		for _, pool := range p.pools {
			if pool.Contains(addr) {
				if err := pool.Release(addr); err != nil {
					return fmt.Errorf("releasing addr %q: %w", addr, err)
				}

				released = true
				break
			}
		}

		if !released {
			return fmt.Errorf("cannot release %q: pool not found", addr)
		}
	}

	return nil
}
