package proxy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/location"
	"github.com/sipplane/sipplane/internal/proxy"
	"github.com/sipplane/sipplane/internal/resources"
	"github.com/sipplane/sipplane/internal/routing"
	"github.com/sipplane/sipplane/internal/webhook"
)

func TestProxyRejectAndRegisterLookup(t *testing.T) {
	store := location.NewMemoryStore()
	_ = store.Put("sip:bob@acme.example", []location.Contact{{
		URI: "sip:bob@127.0.0.1:5099", HostPort: "127.0.0.1:5099", Transport: "udp",
	}}, time.Hour)

	snap := &resources.Snapshot{
		Revision: 1,
		Routes: []*resources.Route{{
			Metadata: resources.ObjectMeta{Name: "deny"},
			Spec: resources.RouteSpec{
				Priority: 200,
				Match: resources.RouteMatch{
					Methods:    []string{"INVITE"},
					RequestURI: &resources.URIMatch{Prefix: "sip:blocked@"},
				},
				Action: resources.RouteAction{Type: "reject", Code: 403},
			},
		}, {
			Metadata: resources.ObjectMeta{Name: "lookup"},
			Spec: resources.RouteSpec{
				Priority: 100,
				Match:    resources.RouteMatch{Methods: []string{"INVITE"}},
				Action:   resources.RouteAction{Type: "registerLookup"},
			},
		}},
		Tenants: map[string]*resources.Tenant{},
		Trunks:  map[string]*resources.Trunk{},
		Secrets: map[string]string{},
	}
	p := &proxy.Proxy{
		Engine: routing.NewEngine(snap),
		Store:  store,
	}

	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "blocked", Host: "acme.example"})
	dst, route, _, _, code := p.ResolveDestination(req)
	if code != 403 || route != "deny" || dst != "" {
		t.Fatalf("reject: dst=%s route=%s code=%d", dst, route, code)
	}

	req2 := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})
	dst, route, _, _, code = p.ResolveDestination(req2)
	if code != 0 || dst != "127.0.0.1:5099" || route != "lookup" {
		t.Fatalf("lookup: dst=%s route=%s code=%d", dst, route, code)
	}

	req3 := sip.NewRequest(sip.INVITE, sip.Uri{User: "missing", Host: "acme.example"})
	_, _, _, _, code = p.ResolveDestination(req3)
	if code != 480 {
		t.Fatalf("miss code=%d", code)
	}
}

func TestNewCancelRequest(t *testing.T) {
	invite := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})
	params := sip.NewParams()
	params.Add("tag", "from-tag")
	invite.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "alice", Host: "acme.example"}, Params: params})
	invite.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: "bob", Host: "acme.example"}})
	cid := sip.CallIDHeader("call-1")
	invite.AppendHeader(&cid)
	invite.AppendHeader(&sip.CSeqHeader{SeqNo: 1, MethodName: sip.INVITE})
	viaParams := sip.NewParams()
	viaParams.Add("branch", "z9hG4bK-1")
	invite.AppendHeader(&sip.ViaHeader{
		ProtocolName: "SIP", ProtocolVersion: "2.0", Transport: "UDP",
		Host: "127.0.0.1", Port: 5060, Params: viaParams,
	})
	invite.SetDestination("10.0.0.2:5060")
	invite.SetSource("127.0.0.1:5060")
	invite.SetTransport("UDP")

	cancel := proxy.NewCancelRequest(invite)
	if cancel.Method != sip.CANCEL {
		t.Fatalf("method=%s", cancel.Method)
	}
	if cancel.CallID() == nil || cancel.CallID().Value() != "call-1" {
		t.Fatal("call-id mismatch")
	}
	if cancel.Destination() != invite.Destination() {
		t.Fatalf("dest=%s", cancel.Destination())
	}
	if cancel.CSeq() == nil || cancel.CSeq().SeqNo != 1 || cancel.CSeq().MethodName != sip.CANCEL {
		t.Fatalf("cseq=%v", cancel.CSeq())
	}
	if cancel.Via() == nil {
		t.Fatal("via required")
	}
	b, _ := cancel.Via().Params.Get("branch")
	if b != "z9hG4bK-1" {
		t.Fatalf("branch=%s", b)
	}
}

func TestProxyWebhookActions(t *testing.T) {
	store := location.NewMemoryStore()
	_ = store.Put("sip:bob@acme.example", []location.Contact{{
		URI: "sip:bob@127.0.0.1:5099", HostPort: "127.0.0.1:5099", Transport: "udp",
	}}, time.Hour)

	var lastAction string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(webhook.Response{Action: lastAction, Target: "10.0.0.8:5070", Code: 486})
	}))
	defer srv.Close()

	mk := func() *proxy.Proxy {
		snap := &resources.Snapshot{
			Revision: 1,
			Routes: []*resources.Route{{
				Metadata: resources.ObjectMeta{Name: "wh", Tenant: "acme"},
				Spec: resources.RouteSpec{
					Priority: 10,
					Match:    resources.RouteMatch{Methods: []string{"INVITE"}},
					Action:   resources.RouteAction{Type: "webhook", Target: srv.URL},
				},
			}},
			Tenants: map[string]*resources.Tenant{},
			Trunks:  map[string]*resources.Trunk{},
			Secrets: map[string]string{},
		}
		return &proxy.Proxy{Engine: routing.NewEngine(snap), Store: store}
	}

	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})

	lastAction = "proxy"
	dst, route, _, tenant, code := mk().ResolveDestination(req)
	if code != 0 || dst != "10.0.0.8:5070" || route != "wh" || tenant != "acme" {
		t.Fatalf("proxy action: dst=%s route=%s tenant=%s code=%d", dst, route, tenant, code)
	}

	lastAction = "continue"
	dst, _, _, _, code = mk().ResolveDestination(req)
	if code != 0 || dst != "127.0.0.1:5099" {
		t.Fatalf("continue->lookup: dst=%s code=%d", dst, code)
	}

	lastAction = "reject"
	dst, _, _, _, code = mk().ResolveDestination(req)
	if code != 486 || dst != "" {
		t.Fatalf("reject: dst=%s code=%d", dst, code)
	}
}

func TestProxyWebhookTimeoutFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(800 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(webhook.Response{Action: "proxy", Target: "should-not-use:1"})
	}))
	defer srv.Close()

	snap := &resources.Snapshot{
		Revision: 1,
		Routes: []*resources.Route{{
			Metadata: resources.ObjectMeta{Name: "wh-timeout"},
			Spec: resources.RouteSpec{
				Match:  resources.RouteMatch{Methods: []string{"INVITE"}},
				Action: resources.RouteAction{Type: "webhook", Target: srv.URL, Code: 503},
			},
		}},
		Tenants: map[string]*resources.Tenant{},
		Trunks:  map[string]*resources.Trunk{},
		Secrets: map[string]string{},
	}
	p := &proxy.Proxy{Engine: routing.NewEngine(snap), Store: location.NewMemoryStore()}
	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})
	dst, _, _, _, code := p.ResolveDestination(req)
	if code != 503 || dst != "" {
		t.Fatalf("timeout fallback: dst=%s code=%d", dst, code)
	}
}
