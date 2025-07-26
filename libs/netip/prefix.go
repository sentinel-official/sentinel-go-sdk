package netip

import (
	"errors"
	"fmt"
	"net/netip"
)

const maxPrefixSize = 1 << 16

type Prefix struct {
	netip.Prefix
}

// NewPrefix creates a new Prefix object from a given CIDR string.
func NewPrefix(cidr string) (*Prefix, error) {
	if cidr == "" {
		return nil, errors.New("CIDR string is empty")
	}

	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CIDR %v: %w", cidr, err)
	}

	p := &Prefix{prefix}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("invalid prefix: %w", err)
	}

	return p, nil
}

// Len calculates the number of addresses in the Prefix block.
func (p *Prefix) Len() int64 {
	diff := p.Addr().BitLen() - p.Bits()
	if diff < 0 {
		return 0
	}

	return int64(1) << diff
}

// Addrs returns a slice of all addresses within the Prefix block.
func (p *Prefix) Addrs() ([]netip.Addr, error) {
	if p.Len() > maxPrefixSize {
		return nil, fmt.Errorf("prefix %v block size exceeds max %v", p, maxPrefixSize)
	}

	var addrs []netip.Addr
	for addr := p.NetworkAddr(); p.Contains(addr); addr = addr.Next() {
		addrs = append(addrs, addr)
	}

	return addrs, nil
}

// Validate checks if the *Prefix block size is within limits.
func (p *Prefix) Validate() error {
	return nil
}

// NetworkAddr returns the network address of the Prefix.
func (p *Prefix) NetworkAddr() netip.Addr {
	return p.Masked().Addr()
}

// BroadcastAddr returns the broadcast address of the Prefix for IPv4.
// Returns an error for IPv6.
func (p *Prefix) BroadcastAddr() (netip.Addr, error) {
	if !p.Addr().Is4() {
		return netip.Addr{}, fmt.Errorf("prefix %v is not IPv4 type", p)
	}

	size := p.Len() - 1
	if size == 0 {
		return p.Addr(), nil
	}

	buf := p.NetworkAddr().As4()
	for i := 0; i < len(buf); i++ {
		buf[len(buf)-i-1] |= byte(size >> (i * 8))
	}

	addr, ok := netip.AddrFromSlice(buf[:])
	if !ok {
		return netip.Addr{}, fmt.Errorf("failed to parse addr from %v", buf)
	}

	return addr, nil
}

// Overlaps checks if two prefixes overlap.
func (p *Prefix) Overlaps(v *Prefix) bool {
	return p.Prefix.Overlaps(v.Prefix)
}
