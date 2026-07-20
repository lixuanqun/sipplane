package redirect

import (
	"strconv"
	"strings"

	"github.com/emiago/sipgo/sip"
)

// Policy decides how to handle 3xx responses from upstream.
type Policy string

const (
	// Follow first Contact automatically (simple redirect).
	Follow Policy = "follow"
	// PassThrough returns the 3xx to the UAC unchanged.
	PassThrough Policy = "passthrough"
	// Reject maps redirect to a final failure.
	Reject Policy = "reject"
)

// ContactHostPort extracts host:port from a 3xx Contact header.
func ContactHostPort(res *sip.Response) (string, bool) {
	c := res.Contact()
	if c == nil {
		return "", false
	}
	host := c.Address.Host
	port := c.Address.Port
	if port == 0 {
		port = 5060
	}
	if host == "" {
		return "", false
	}
	return host + ":" + strconv.Itoa(port), true
}

// ShouldFollow reports whether this response should be auto-followed.
func ShouldFollow(res *sip.Response, p Policy) bool {
	if p != Follow {
		return false
	}
	if res.StatusCode < 300 || res.StatusCode >= 400 {
		return false
	}
	_, ok := ContactHostPort(res)
	return ok
}

// NormalizePolicy parses config string.
func NormalizePolicy(s string) Policy {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "follow":
		return Follow
	case "reject":
		return Reject
	default:
		return PassThrough
	}
}
