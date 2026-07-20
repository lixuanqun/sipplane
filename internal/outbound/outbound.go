package outbound

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/emiago/sipgo/sip"
)

// Flow identifies a UA network flow (RFC 5626).
type Flow struct {
	RemoteHost string
	RemotePort int
	Transport  string
	LocalHost  string
	LocalPort  int
}

// FromRequest builds a flow from the packet source and advertised local edge.
func FromRequest(req *sip.Request, localHost string, localPort int) Flow {
	host, portStr, err := net.SplitHostPort(req.Source())
	port := 0
	if err == nil {
		port, _ = strconv.Atoi(portStr)
	} else {
		host = req.Source()
	}
	return Flow{
		RemoteHost: host,
		RemotePort: port,
		Transport:  strings.ToLower(req.Transport()),
		LocalHost:  localHost,
		LocalPort:  localPort,
	}
}

// Token returns a stable opaque flow-token (HMAC) for embedding in Path.
func (f Flow) Token(secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "%s|%d|%s|%s|%d", f.RemoteHost, f.RemotePort, f.Transport, f.LocalHost, f.LocalPort)
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// HostPort is the dial target for this flow.
func (f Flow) HostPort() string {
	if f.RemotePort <= 0 {
		return f.RemoteHost
	}
	return net.JoinHostPort(f.RemoteHost, strconv.Itoa(f.RemotePort))
}

// SupportsOutbound reports whether the request advertises outbound support.
func SupportsOutbound(req *sip.Request) bool {
	for _, name := range []string{"Supported", "Require"} {
		h := req.GetHeader(name)
		if h == nil {
			continue
		}
		for _, part := range strings.Split(h.Value(), ",") {
			if strings.EqualFold(strings.TrimSpace(part), "outbound") {
				return true
			}
		}
	}
	return false
}

// PathURI builds a Path URI with ;lr;ob and opaque flow token user-part.
// Example: sip:TOKEN@edge.example:5060;lr;ob
func PathURI(advertisedHost string, advertisedPort int, token string) string {
	if advertisedPort == 0 {
		advertisedPort = 5060
	}
	return fmt.Sprintf("sip:%s@%s:%d;lr;ob", token, advertisedHost, advertisedPort)
}

// AddPathWithFlow appends Path + Supported: outbound,path when outbound is in use.
func AddPathWithFlow(res *sip.Response, advertisedHost string, advertisedPort int, token string, outbound bool) {
	uri := PathURI(advertisedHost, advertisedPort, token)
	res.AppendHeader(sip.NewHeader("Path", "<"+uri+">"))
	if outbound {
		res.AppendHeader(sip.NewHeader("Supported", "path,outbound"))
	} else {
		res.AppendHeader(sip.NewHeader("Supported", "path"))
	}
}

// ParseFlowToken extracts the user-part token from a Path/Route URI if present.
func ParseFlowToken(uri string) string {
	uri = strings.TrimPrefix(strings.TrimSpace(uri), "<")
	uri = strings.TrimSuffix(uri, ">")
	u := sip.Uri{}
	if err := sip.ParseUri(uri, &u); err != nil {
		return ""
	}
	return u.User
}
