package dataplane_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/icholy/digest"
	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/dataplane"
	"github.com/sipplane/sipplane/internal/resources"
)

func TestHealthAndRegisterInviteFlow(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpAddr := ln.Addr().String()
	_ = ln.Close()

	sipPort := freeUDPPort(t)
	cfg := config.Config{
		Listen:         "127.0.0.1:" + itoa(sipPort),
		Transport:      "udp",
		AdvertisedHost: "127.0.0.1",
		AdvertisedPort: sipPort,
		HTTPListen:     httpAddr,
		Realm:          "sipplane",
		LogLevel:       "error",
	}

	snap := labSnapshot()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := dataplane.New(cfg, snap, log)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Run(ctx) }()

	waitHTTP(t, "http://"+httpAddr+"/readyz")
	waitHTTP(t, "http://"+httpAddr+"/healthz")
	waitHTTP(t, "http://"+httpAddr+"/metrics")

	alicePort := freeUDPPort(t)
	bobPort := freeUDPPort(t)
	proxyAddr := cfg.Listen

	// Bob: one UA shared by server (receive INVITE) and client (REGISTER).
	bobUA, err := sipgo.NewUA(sipgo.WithUserAgent("bob-ua"))
	if err != nil {
		t.Fatal(err)
	}
	defer bobUA.Close()
	bobSrv, err := sipgo.NewServer(bobUA)
	if err != nil {
		t.Fatal(err)
	}
	bobClient, err := sipgo.NewClient(bobUA, sipgo.WithClientHostname("127.0.0.1"), sipgo.WithClientPort(bobPort))
	if err != nil {
		t.Fatal(err)
	}
	bobInvite := make(chan *sip.Request, 1)
	bobSrv.OnInvite(func(req *sip.Request, tx sip.ServerTransaction) {
		bobInvite <- req.Clone()
		_ = tx.Respond(sip.NewResponseFromRequest(req, 200, "OK", nil))
	})
	bobSrv.OnAck(func(req *sip.Request, tx sip.ServerTransaction) {})
	bobSrv.OnBye(func(req *sip.Request, tx sip.ServerTransaction) {
		_ = tx.Respond(sip.NewResponseFromRequest(req, 200, "OK", nil))
	})
	go func() { _ = bobSrv.ListenAndServe(ctx, "udp", "127.0.0.1:"+itoa(bobPort)) }()
	time.Sleep(150 * time.Millisecond)

	registerWith(t, bobClient, "bob", "bob-secret", "127.0.0.1", bobPort, proxyAddr)

	// Alice: shared UA + listen so REGISTER/INVITE responses can be received on contact port.
	aliceUA, err := sipgo.NewUA(sipgo.WithUserAgent("alice-ua"))
	if err != nil {
		t.Fatal(err)
	}
	defer aliceUA.Close()
	aliceSrv, err := sipgo.NewServer(aliceUA)
	if err != nil {
		t.Fatal(err)
	}
	aliceClient, err := sipgo.NewClient(aliceUA, sipgo.WithClientHostname("127.0.0.1"), sipgo.WithClientPort(alicePort))
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = aliceSrv.ListenAndServe(ctx, "udp", "127.0.0.1:"+itoa(alicePort)) }()
	time.Sleep(150 * time.Millisecond)
	registerWith(t, aliceClient, "alice", "alice-secret", "127.0.0.1", alicePort, proxyAddr)

	recipient := sip.Uri{User: "bob", Host: "acme.example"}
	req := sip.NewRequest(sip.INVITE, recipient)
	fromAlice := sip.NewParams()
	fromAlice.Add("tag", "alice-tag")
	req.AppendHeader(&sip.FromHeader{
		Address: sip.Uri{User: "alice", Host: "acme.example"},
		Params:  fromAlice,
	})
	req.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: "bob", Host: "acme.example"}})
	req.AppendHeader(sip.NewHeader("Content-Type", "application/sdp"))
	req.SetBody([]byte("v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\nm=audio 10000 RTP/AVP 0\r\n"))
	req.SetDestination(proxyAddr)

	txCtx, txCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer txCancel()
	res, err := aliceClient.Do(txCtx, req)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("invite status=%d body=%s", res.StatusCode, res.StartLine())
	}

	select {
	case got := <-bobInvite:
		rr := got.GetHeader("Record-Route")
		if rr == nil {
			t.Fatal("expected Record-Route on proxied INVITE")
		}
		if !contains(rr.Value(), "127.0.0.1") {
			t.Fatalf("Record-Route host mismatch: %s", rr.Value())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("bob did not receive INVITE")
	}
}

func labSnapshot() *resources.Snapshot {
	return &resources.Snapshot{
		Revision: 1,
		Tenants:  map[string]*resources.Tenant{"acme": {Metadata: resources.ObjectMeta{Name: "acme"}}},
		Endpoints: []*resources.Endpoint{
			{
				Metadata: resources.ObjectMeta{Name: "alice", Tenant: "acme"},
				Spec: resources.EndpointSpec{
					AORs: []string{"sip:alice@acme.example"},
					Auth: resources.EndpointAuth{Username: "alice", Password: "alice-secret", PasswordSecretRef: "inline/acme/alice"},
				},
			},
			{
				Metadata: resources.ObjectMeta{Name: "bob", Tenant: "acme"},
				Spec: resources.EndpointSpec{
					AORs: []string{"sip:bob@acme.example"},
					Auth: resources.EndpointAuth{Username: "bob", Password: "bob-secret", PasswordSecretRef: "inline/acme/bob"},
				},
			},
		},
		Secrets: map[string]string{
			"inline/acme/alice": "alice-secret",
			"inline/acme/bob":   "bob-secret",
		},
		Routes: []*resources.Route{{
			Metadata: resources.ObjectMeta{Name: "ua-to-ua", Tenant: "acme"},
			Spec: resources.RouteSpec{
				Priority: 100,
				Match:    resources.RouteMatch{Methods: []string{"INVITE", "ACK", "BYE", "CANCEL", "OPTIONS"}},
				Action:   resources.RouteAction{Type: "registerLookup"},
			},
		}},
		Trunks: map[string]*resources.Trunk{},
	}
}

func registerWith(t *testing.T, client *sipgo.Client, user, pass, contactHost string, contactPort int, proxyAddr string) {
	t.Helper()
	recipient := sip.Uri{Host: "acme.example"}
	req := sip.NewRequest(sip.REGISTER, recipient)
	fromParams := sip.NewParams()
	fromParams.Add("tag", user+"-tag")
	req.AppendHeader(&sip.FromHeader{
		Address: sip.Uri{User: user, Host: "acme.example"},
		Params:  fromParams,
	})
	req.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: user, Host: "acme.example"}})
	req.AppendHeader(&sip.ContactHeader{Address: sip.Uri{User: user, Host: contactHost, Port: contactPort}})
	ex := sip.ExpiresHeader(3600)
	req.AppendHeader(&ex)
	req.SetDestination(proxyAddr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("register %s: %v", user, err)
	}
	if res.StatusCode != 401 {
		t.Fatalf("register %s expected 401, got %d", user, res.StatusCode)
	}
	chalHdr := res.GetHeader("WWW-Authenticate")
	if chalHdr == nil {
		t.Fatal("missing WWW-Authenticate")
	}
	chal, err := digest.ParseChallenge(chalHdr.Value())
	if err != nil {
		t.Fatal(err)
	}
	cred, err := digest.Digest(chal, digest.Options{
		Method:   "REGISTER",
		URI:      req.Recipient.Addr(),
		Username: user,
		Password: pass,
		Count:    1,
	})
	if err != nil {
		t.Fatal(err)
	}

	req2 := req.Clone()
	req2.RemoveHeader("Via")
	req2.RemoveHeader("Authorization")
	req2.AppendHeader(sip.NewHeader("Authorization", cred.String()))
	cseq := req2.CSeq()
	cseq.SeqNo++
	req2.SetDestination(proxyAddr)

	res2, err := client.Do(ctx, req2)
	if err != nil {
		t.Fatalf("register auth %s: %v", user, err)
	}
	if res2.StatusCode != 200 {
		t.Fatalf("register auth %s status=%d", user, res2.StatusCode)
	}
}

func waitHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", url)
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
