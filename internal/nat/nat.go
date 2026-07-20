package nat

import (
	"net"
	"strconv"
	"strings"

	"github.com/emiago/sipgo/sip"
)

// FixContact returns the dialable host:port for a REGISTER Contact,
// rewriting private / unmatched contacts using Via received/rport or packet source (RFC 3581).
func FixContact(cont *sip.ContactHeader, req *sip.Request) (hostPort, transport string, rewritten bool) {
	transport = strings.ToLower(req.Transport())
	if cont.Address.UriParams != nil {
		if t, ok := cont.Address.UriParams.Get("transport"); ok {
			transport = strings.ToLower(t)
		}
	}

	host := cont.Address.Host
	port := cont.Address.Port
	if port == 0 {
		port = defaultPort(transport)
	}

	srcHost, srcPort := splitHostPort(req.Source())
	recvHost, recvPort := viaReceived(req)

	useHost, usePort := host, port
	needFix := isPrivateOrLoopback(host) || hostMismatch(host, srcHost, recvHost)

	if needFix {
		if recvHost != "" {
			useHost = recvHost
		} else if srcHost != "" {
			useHost = srcHost
		}
		if recvPort > 0 {
			usePort = recvPort
		} else if srcPort > 0 {
			usePort = srcPort
		}
		rewritten = useHost != host || usePort != port
	}

	return net.JoinHostPort(useHost, strconv.Itoa(usePort)), transport, rewritten
}

// AddPath appends a Path header (RFC 3327) pointing at advertised URI when enabled.
func AddPath(res *sip.Response, advertisedSIPURI string) {
	if advertisedSIPURI == "" {
		return
	}
	// Path uses ;lr like Record-Route
	res.AppendHeader(sip.NewHeader("Path", "<"+advertisedSIPURI+">"))
	res.AppendHeader(sip.NewHeader("Supported", "path"))
}

func viaReceived(req *sip.Request) (host string, port int) {
	via := req.Via()
	if via == nil || via.Params == nil {
		return "", 0
	}
	if v, ok := via.Params.Get("received"); ok && v != "" {
		host = v
	}
	if v, ok := via.Params.Get("rport"); ok && v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}
	return host, port
}

func splitHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	p, _ := strconv.Atoi(portStr)
	return host, p
}

func isPrivateOrLoopback(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		// hostname — treat as public-ish; do not rewrite unless Via received present
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

func hostMismatch(contactHost, srcHost, recvHost string) bool {
	if recvHost != "" && !strings.EqualFold(contactHost, recvHost) {
		return true
	}
	// If transport layer already stamped received via different IP than Contact.
	_ = srcHost
	return false
}

func defaultPort(transport string) int {
	switch strings.ToLower(transport) {
	case "tls", "wss":
		return 5061
	default:
		return 5060
	}
}
