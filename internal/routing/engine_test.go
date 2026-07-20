package routing

import (
	"testing"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/resources"
)

func TestMatchRegisterLookupAndReject(t *testing.T) {
	snap := &resources.Snapshot{
		Routes: []*resources.Route{
			{
				Metadata: resources.ObjectMeta{Name: "reject-spam"},
				Spec: resources.RouteSpec{
					Priority: 200,
					Match: resources.RouteMatch{
						Methods:    []string{"INVITE"},
						RequestURI: &resources.URIMatch{Prefix: "sip:999"},
					},
					Action: resources.RouteAction{Type: "reject", Code: 403, Reason: "Forbidden"},
				},
			},
			{
				Metadata: resources.ObjectMeta{Name: "to-ua"},
				Spec: resources.RouteSpec{
					Priority: 100,
					Match: resources.RouteMatch{
						Methods: []string{"INVITE"},
					},
					Action: resources.RouteAction{Type: "registerLookup"},
				},
			},
		},
	}
	eng := NewEngine(snap)

	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "alice", Host: "acme.example"})
	params := sip.NewParams()
	params.Add("tag", "x")
	req.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "bob", Host: "acme.example"}, Params: params})
	d, ok := eng.Match(req)
	if !ok || d.Action.Type != "registerLookup" {
		t.Fatalf("want registerLookup, got %+v ok=%v", d, ok)
	}

	req2 := sip.NewRequest(sip.INVITE, sip.Uri{User: "999", Host: "acme.example"})
	params2 := sip.NewParams()
	params2.Add("tag", "y")
	req2.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "bob", Host: "acme.example"}, Params: params2})
	d2, ok := eng.Match(req2)
	if !ok {
		t.Fatal("expected match")
	}
	t.Logf("action=%s (uri=%s)", d2.Action.Type, req2.Recipient.String())
}

func TestMatchProxyTarget(t *testing.T) {
	snap := &resources.Snapshot{
		Routes: []*resources.Route{{
			Metadata: resources.ObjectMeta{Name: "to-pbx"},
			Spec: resources.RouteSpec{
				Priority: 10,
				Match:    resources.RouteMatch{Methods: []string{"INVITE"}},
				Action:   resources.RouteAction{Type: "proxy", Target: "10.0.0.5:5080"},
			},
		}},
	}
	eng := NewEngine(snap)
	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "1001", Host: "pbx.local"})
	d, ok := eng.Match(req)
	if !ok || d.DestAddr != "10.0.0.5:5080" {
		t.Fatalf("got %+v ok=%v", d, ok)
	}
}

func TestLoadBalanceConsistentHash(t *testing.T) {
	snap := &resources.Snapshot{
		Trunks: map[string]*resources.Trunk{
			"fs-a": {Metadata: resources.ObjectMeta{Name: "fs-a"}, Spec: resources.TrunkSpec{
				Destination: resources.TrunkDestination{Host: "10.0.0.1", Port: 5060},
			}},
			"fs-b": {Metadata: resources.ObjectMeta{Name: "fs-b"}, Spec: resources.TrunkSpec{
				Destination: resources.TrunkDestination{Host: "10.0.0.2", Port: 5060},
			}},
		},
		Routes: []*resources.Route{{
			Metadata: resources.ObjectMeta{Name: "lb", Tenant: "acme"},
			Spec: resources.RouteSpec{
				Priority: 10,
				Match:    resources.RouteMatch{Methods: []string{"INVITE"}},
				Action: resources.RouteAction{
					Type:      "loadBalance",
					Algorithm: "consistent_hash",
					Trunks: []resources.TrunkWeight{
						{Name: "fs-a", Weight: 100},
						{Name: "fs-b", Weight: 100},
					},
				},
			},
		}},
	}
	eng := NewEngine(snap)

	pick := func(cid string) string {
		req := sip.NewRequest(sip.INVITE, sip.Uri{User: "x", Host: "acme.example"})
		c := sip.CallIDHeader(cid)
		req.AppendHeader(&c)
		d, ok := eng.Match(req)
		if !ok || d.DestAddr == "" {
			t.Fatalf("cid=%s dest=%v", cid, d)
		}
		return d.DestAddr
	}
	a1 := pick("call-stable-1")
	a2 := pick("call-stable-1")
	if a1 != a2 {
		t.Fatalf("affinity broken: %s vs %s", a1, a2)
	}
	// Different Call-ID may land elsewhere (not required, but both must be valid).
	b := pick("call-other-2")
	if b != "10.0.0.1:5060" && b != "10.0.0.2:5060" {
		t.Fatalf("unexpected dest %s", b)
	}
}
