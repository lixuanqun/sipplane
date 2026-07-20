package dataplane_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/dataplane"
)

func TestOPTIONSKeepalive(t *testing.T) {
	httpAddr, proxyAddr, cancel := startDP(t, "udp")
	defer cancel()

	ua, err := sipgo.NewUA(sipgo.WithUserAgent("opt-udp"))
	if err != nil {
		t.Fatal(err)
	}
	defer ua.Close()
	client, err := sipgo.NewClient(ua, sipgo.WithClientHostname("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}

	req := sip.NewRequest(sip.OPTIONS, sip.Uri{User: "sipplane", Host: "health"})
	from := sip.NewParams()
	from.Add("tag", "opt-tag")
	req.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "probe", Host: "acme.example"}, Params: from})
	req.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: "sipplane", Host: "health"}})
	req.SetDestination(proxyAddr)

	ctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	res, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("OPTIONS: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("OPTIONS status=%d", res.StatusCode)
	}
	allow := res.GetHeader("Allow")
	if allow == nil || !strings.Contains(strings.ToUpper(allow.Value()), "OPTIONS") {
		t.Fatalf("expected Allow with OPTIONS, got %v", allow)
	}

	body := scrapeMetrics(t, "http://"+httpAddr+"/metrics")
	if !strings.Contains(body, "sipplane_sip_requests_total") {
		t.Fatal("metrics missing sipplane_sip_requests_total")
	}
	if !strings.Contains(body, `method="OPTIONS"`) {
		t.Fatalf("metrics missing OPTIONS counter:\n%s", body)
	}
}

func TestTCPListenAndOPTIONS(t *testing.T) {
	_, proxyAddr, cancel := startDP(t, "tcp")
	defer cancel()

	ua, err := sipgo.NewUA(sipgo.WithUserAgent("opt-tcp"))
	if err != nil {
		t.Fatal(err)
	}
	defer ua.Close()
	client, err := sipgo.NewClient(ua, sipgo.WithClientHostname("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}

	req := sip.NewRequest(sip.OPTIONS, sip.Uri{User: "sipplane", Host: "health"})
	from := sip.NewParams()
	from.Add("tag", "opt-tcp-tag")
	req.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "probe", Host: "acme.example"}, Params: from})
	req.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: "sipplane", Host: "health"}})
	req.SetTransport("TCP")
	req.SetDestination(proxyAddr)

	ctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	res, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("TCP OPTIONS: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("TCP OPTIONS status=%d", res.StatusCode)
	}
}

func TestMetricsAfterRegister(t *testing.T) {
	httpAddr, proxyAddr, cancel := startDP(t, "udp")
	defer cancel()

	port := freeUDPPort(t)
	ua, err := sipgo.NewUA(sipgo.WithUserAgent("metrics-ua"))
	if err != nil {
		t.Fatal(err)
	}
	defer ua.Close()
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		t.Fatal(err)
	}
	client, err := sipgo.NewClient(ua, sipgo.WithClientHostname("127.0.0.1"), sipgo.WithClientPort(port))
	if err != nil {
		t.Fatal(err)
	}
	ctx, ccancel := context.WithCancel(context.Background())
	defer ccancel()
	go func() { _ = srv.ListenAndServe(ctx, "udp", "127.0.0.1:"+itoa(port)) }()
	time.Sleep(100 * time.Millisecond)

	registerWith(t, client, "alice", "alice-secret", "127.0.0.1", port, proxyAddr)

	body := scrapeMetrics(t, "http://"+httpAddr+"/metrics")
	if !strings.Contains(body, `method="REGISTER"`) {
		t.Fatalf("expected REGISTER metrics, got:\n%s", body)
	}
	if !strings.Contains(body, `code="200"`) {
		t.Fatalf("expected code=200 metrics, got:\n%s", body)
	}
}

func startDP(t *testing.T, transport string) (httpAddr, sipAddr string, cancel context.CancelFunc) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpAddr = ln.Addr().String()
	_ = ln.Close()

	var sipPort int
	if transport == "tcp" {
		sipPort = freeTCPPort(t)
	} else {
		sipPort = freeUDPPort(t)
	}
	cfg := config.Config{
		Listen:         "127.0.0.1:" + itoa(sipPort),
		Transport:      transport,
		AdvertisedHost: "127.0.0.1",
		AdvertisedPort: sipPort,
		HTTPListen:     httpAddr,
		Realm:          "sipplane",
		LogLevel:       "error",
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := dataplane.New(cfg, labSnapshot(), log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = srv.Run(ctx) }()
	waitHTTP(t, "http://"+httpAddr+"/readyz")
	return httpAddr, cfg.Listen, cancel
}

func scrapeMetrics(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("metrics status=%d", resp.StatusCode)
	}
	return string(b)
}
