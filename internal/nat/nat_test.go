package nat

import (
	"testing"

	"github.com/emiago/sipgo/sip"
)

func TestFixContactPrivateIPUsesSource(t *testing.T) {
	req := sip.NewRequest(sip.REGISTER, sip.Uri{Host: "acme.example"})
	req.SetSource("203.0.113.10:45060")
	req.SetTransport("UDP")
	cont := &sip.ContactHeader{Address: sip.Uri{User: "alice", Host: "192.168.1.5", Port: 5060}}

	hostPort, transport, rewritten := FixContact(cont, req)
	if !rewritten {
		t.Fatal("expected rewrite for private contact")
	}
	if hostPort != "203.0.113.10:45060" {
		t.Fatalf("got %s", hostPort)
	}
	if transport != "udp" {
		t.Fatalf("transport=%s", transport)
	}
}

func TestFixContactUsesViaReceivedRport(t *testing.T) {
	req := sip.NewRequest(sip.REGISTER, sip.Uri{Host: "acme.example"})
	req.SetSource("10.0.0.1:5060")
	params := sip.NewParams()
	params.Add("branch", "z9hG4bK-1")
	params.Add("received", "198.51.100.7")
	params.Add("rport", "5062")
	req.AppendHeader(&sip.ViaHeader{
		ProtocolName: "SIP", ProtocolVersion: "2.0", Transport: "UDP",
		Host: "192.168.0.2", Port: 5060, Params: params,
	})
	cont := &sip.ContactHeader{Address: sip.Uri{User: "bob", Host: "192.168.0.2", Port: 5060}}

	hostPort, _, rewritten := FixContact(cont, req)
	if !rewritten {
		t.Fatal("expected rewrite")
	}
	if hostPort != "198.51.100.7:5062" {
		t.Fatalf("got %s", hostPort)
	}
}

func TestFixContactPublicUnchanged(t *testing.T) {
	req := sip.NewRequest(sip.REGISTER, sip.Uri{Host: "acme.example"})
	req.SetSource("203.0.113.1:5060")
	cont := &sip.ContactHeader{Address: sip.Uri{User: "carol", Host: "203.0.113.50", Port: 5060}}
	hostPort, _, rewritten := FixContact(cont, req)
	if rewritten {
		t.Fatal("public contact should not rewrite")
	}
	if hostPort != "203.0.113.50:5060" {
		t.Fatalf("got %s", hostPort)
	}
}
