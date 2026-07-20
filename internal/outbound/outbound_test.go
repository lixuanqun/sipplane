package outbound

import (
	"testing"

	"github.com/emiago/sipgo/sip"
)

func TestFlowTokenStable(t *testing.T) {
	f := Flow{RemoteHost: "203.0.113.9", RemotePort: 5060, Transport: "udp", LocalHost: "sip.example.com", LocalPort: 5060}
	secret := []byte("test-secret")
	a := f.Token(secret)
	b := f.Token(secret)
	if a != b || a == "" {
		t.Fatalf("token unstable: %q %q", a, b)
	}
	other := Flow{RemoteHost: "203.0.113.10", RemotePort: 5060, Transport: "udp", LocalHost: "sip.example.com", LocalPort: 5060}
	if other.Token(secret) == a {
		t.Fatal("different flows must differ")
	}
}

func TestSupportsOutbound(t *testing.T) {
	req := sip.NewRequest(sip.REGISTER, sip.Uri{Host: "example.com"})
	if SupportsOutbound(req) {
		t.Fatal("expected false")
	}
	req.AppendHeader(sip.NewHeader("Supported", "path, outbound"))
	if !SupportsOutbound(req) {
		t.Fatal("expected true")
	}
}

func TestPathURIAndParse(t *testing.T) {
	uri := PathURI("sip.example.com", 5060, "abcTOKEN")
	if got := ParseFlowToken(uri); got != "abcTOKEN" {
		t.Fatalf("got %q", got)
	}
}
