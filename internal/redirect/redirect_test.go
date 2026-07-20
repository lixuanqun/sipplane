package redirect

import (
	"testing"

	"github.com/emiago/sipgo/sip"
)

func TestContactHostPort(t *testing.T) {
	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "a", Host: "b"})
	res := sip.NewResponseFromRequest(req, 302, "Moved Temporarily", nil)
	res.AppendHeader(&sip.ContactHeader{Address: sip.Uri{User: "a", Host: "10.0.0.9", Port: 5080}})
	hp, ok := ContactHostPort(res)
	if !ok || hp != "10.0.0.9:5080" {
		t.Fatalf("got %q ok=%v", hp, ok)
	}
	if !ShouldFollow(res, Follow) {
		t.Fatal("expected follow")
	}
	if ShouldFollow(res, PassThrough) {
		t.Fatal("passthrough should not follow")
	}
}

func TestNormalizePolicy(t *testing.T) {
	if NormalizePolicy("follow") != Follow {
		t.Fatal("follow")
	}
	if NormalizePolicy("reject") != Reject {
		t.Fatal("reject")
	}
	if NormalizePolicy("") != PassThrough {
		t.Fatal("default passthrough")
	}
}
