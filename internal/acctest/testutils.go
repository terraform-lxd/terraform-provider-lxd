package acctest

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	petname "github.com/dustinkirkland/golang-petname"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// generateString generates a random string of the given length.
func generateString(length int) string {
	s := make([]byte, length)
	for i := range s {
		s[i] = charset[rand.IntN(len(charset))]
	}
	return string(s)
}

// GenerateName generates a petname with a random string suffix.
// If requested number of words is 1 or less, just petname is returned.
func GenerateName(words int, separator string) string {
	if words <= 1 {
		return petname.Name()
	}

	return petname.Generate(words-1, separator) + separator + generateString(6)
}

// QuoteStrings converts slice of strings into a single string where each slice
// element is quoted and delimited with a comma and whitespace.
func QuoteStrings(slice []string) string {
	quoted := make([]string, len(slice))
	for i, s := range slice {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}

var usedAddrsLock = sync.Mutex{}
var usedAddrs = make(map[string]struct{})

// Subnet represents a randomly generated private subnet, with both an IPv4
// and IPv6 range, used to avoid address collisions between acceptance tests
// that create networks.
type Subnet struct {
	ipv4 string
	ipv6 string
}

// GenerateSubnet generates a random private /24 IPv4 subnet and a random
// private /64 IPv6 subnet.
func GenerateSubnet() Subnet {
	randomIPv4 := func() string {
		// Use private IPv4 range 10.11.0.0 - 10.255.255.0.
		// This avoids conflicts with the private IPs used by GitHub runners.
		return fmt.Sprintf("10.%d.%d.0", rand.IntN(245)+11, rand.IntN(256))
	}

	randomIPv6 := func() string {
		return fmt.Sprintf("fd42:%x:%x:%x::", rand.IntN(0x10000), rand.IntN(0x10000), rand.IntN(0x10000))
	}

	findUniqueIP := func(newIP func() string) string {
		usedAddrsLock.Lock()
		defer usedAddrsLock.Unlock()

		for {
			ip := newIP()
			_, used := usedAddrs[ip]
			if !used {
				usedAddrs[ip] = struct{}{}
				return ip
			}
		}
	}

	return Subnet{
		ipv4: findUniqueIP(randomIPv4),
		ipv6: findUniqueIP(randomIPv6),
	}
}

// GatewayCIDRv4 returns the IPv4 gateway address in CIDR notation.
// Example: "10.123.45.1/24".
func (s Subnet) GatewayCIDRv4() string {
	return s.HostIPv4(1) + "/24"
}

// GatewayCIDRv6 returns the IPv6 gateway address in CIDR notation.
// Example: "fd42:1:2:3::1/64".
func (s Subnet) GatewayCIDRv6() string {
	return s.HostIPv6(1) + "/64"
}

// HostIPv4 returns the address of the given host within the IPv4 subnet.
// Example: HostIPv4(200) returns "10.123.45.200".
func (s Subnet) HostIPv4(host int) string {
	return fmt.Sprintf("%s.%d", strings.TrimSuffix(s.ipv4, ".0"), host)
}

// HostIPv6 returns the address of the given host within the IPv6 subnet.
// Example: HostIPv6(200) returns "fd42:1:2:3::c8".
func (s Subnet) HostIPv6(host int) string {
	return fmt.Sprintf("%s%x", s.ipv6, host)
}

// RangeV4 returns the IPv4 subnet range.
// Example: "10.123.45.0-10.123.45.255".
func (s Subnet) RangeV4() string {
	return fmt.Sprintf("%s-%s", s.HostIPv4(0), s.HostIPv4(255))
}

// RangeV6 returns the IPv6 subnet range.
// Example: "fd42:1:2:3::-fd42:1:2:3::ffff:ffff:ffff:ffff".
func (s Subnet) RangeV6() string {
	return fmt.Sprintf("%s-%s", s.HostIPv6(0), s.ipv6+"ffff:ffff:ffff:ffff")
}

// SubRangeV4 returns the address range between the two given hosts within the IPv4 subnet.
// Example: SubRangeV4(100, 150) returns "10.123.45.100-10.123.45.150".
func (s Subnet) SubRangeV4(fromHost int, toHost int) string {
	return fmt.Sprintf("%s-%s", s.HostIPv4(fromHost), s.HostIPv4(toHost))
}

// SubRangeV6 returns the address range of a sub-block within the /64 subnet, identified
// by the given sub-block number (encoded in the next hextet).
// Example: SubRangeV6(10) returns "fd42:1:2:3:a::-fd42:1:2:3:a::ffff".
func (s Subnet) SubRangeV6(sub int) string {
	prefix := fmt.Sprintf("%s:%x::", strings.TrimSuffix(s.ipv6, "::"), sub)
	return fmt.Sprintf("%s-%sffff", prefix, prefix)
}

// GenerateMACAddress generates a random locally-administered MAC address.
func GenerateMACAddress() string {
	usedAddrsLock.Lock()
	defer usedAddrsLock.Unlock()

	for {
		mac := fmt.Sprintf("02:16:3e:%02x:%02x:%02x", rand.IntN(256), rand.IntN(256), rand.IntN(256))
		_, ok := usedAddrs[mac]
		if !ok {
			usedAddrs[mac] = struct{}{}
			return mac
		}
	}
}
