package policy

import (
	"context"
	"testing"

	"github.com/emiago/sipgo/sip"
)

func TestACLDenyCIDR(t *testing.T) {
	acl := &ACL{DenyCIDRs: []string{"10.0.0.0/8"}}
	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "a", Host: "b"})
	req.SetSource("10.1.2.3:5060")
	res := acl.Check(req)
	if res.Decision != Deny {
		t.Fatalf("want deny, got %+v", res)
	}
}

func TestRateLimit(t *testing.T) {
	rl := &RateLimit{CPS: 1, Burst: 1}
	if !rl.Allow(context.Background(), "") {
		t.Fatal("first should allow")
	}
	if rl.Allow(context.Background(), "") {
		t.Fatal("second should deny")
	}
}
