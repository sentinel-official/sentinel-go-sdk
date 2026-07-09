package netip

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

const maxPrefixSize = 1 << 16

type Prefix struct {
	netip.Prefix
}

// NewPrefix creates a new Prefix object from a given CIDR string.
func NewPrefix(cidr string) (*Prefix, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return nil, errors.New("empty CIDR string")
	}

	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("parsing CIDR %q: %w", cidr, err)
	}

	p := &Prefix{prefix}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validating CIDR %q: %w", cidr, err)
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
		return nil, fmt.Errorf("prefix %q block size %d exceeds max size %d", p, p.Len(), maxPrefixSize)
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
		return netip.Addr{}, fmt.Errorf("broadcast addr not applicable for IPv6 prefix %q", p)
	}

	size := p.Len() - 1
	if size == 0 {
		return p.Addr(), nil
	}

	buf := p.NetworkAddr().As4()
	for i := range buf {
		buf[len(buf)-i-1] |= byte(size >> (i * 8))
	}

	addr, ok := netip.AddrFromSlice(buf[:])
	if !ok {
		return netip.Addr{}, errors.New("creating broadcast addr from slice")
	}

	return addr, nil
}

// Overlaps checks if two prefixes overlap.
func (p *Prefix) Overlaps(v *Prefix) bool {
	return p.Prefix.Overlaps(v.Prefix)
}
