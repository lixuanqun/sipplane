package discovery

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/resources"
)

func TestHealthCheckerOPTIONSProbe(t *testing.T) {
	ln, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.LocalAddr().(*net.UDPAddr).Port
	_ = ln.Close()

	ua, err := sipgo.NewUA(sipgo.WithUserAgent("trunk-health"))
	if err != nil {
		t.Fatal(err)
	}
	defer ua.Close()
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		t.Fatal(err)
	}
	gotOPTS := make(chan struct{}, 1)
	srv.OnOptions(func(req *sip.Request, tx sip.ServerTransaction) {
		select {
		case gotOPTS <- struct{}{}:
		default:
		}
		_ = tx.Respond(sip.NewResponseFromRequest(req, 200, "OK", nil))
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.ListenAndServe(ctx, "udp", "127.0.0.1:"+itoa(port)) }()
	time.Sleep(80 * time.Millisecond)

	clientUA, _ := sipgo.NewUA(sipgo.WithUserAgent("hc-client"))
	defer clientUA.Close()
	client, _ := sipgo.NewClient(clientUA, sipgo.WithClientHostname("127.0.0.1"))

	trunks := map[string]*resources.Trunk{
		"carrier-a": {
			Metadata: resources.ObjectMeta{Name: "carrier-a"},
			Spec: resources.TrunkSpec{
				Destination: resources.TrunkDestination{Host: "127.0.0.1", Port: port, Transport: "udp"},
				Options:     resources.TrunkOptions{SendOptionsPing: true},
			},
		},
	}
	groups := GroupsFromPingTrunks(trunks)
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
	hc := &HealthChecker{
		Client:  client,
		Timeout: time.Second,
		Groups:  groups,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	hc.ProbeOnce(ctx)

	select {
	case <-gotOPTS:
	case <-time.After(2 * time.Second):
		t.Fatal("OPTIONS not received")
	}
	if !groups[0].IsHealthy("carrier-a") {
		t.Fatal("expected healthy after 200")
	}

	// Unreachable destination should accumulate fails and eject.
	bad := GroupsFromPingTrunks(map[string]*resources.Trunk{
		"down": {
			Metadata: resources.ObjectMeta{Name: "down"},
			Spec: resources.TrunkSpec{
				Destination: resources.TrunkDestination{Host: "127.0.0.1", Port: 1},
				Options:     resources.TrunkOptions{SendOptionsPing: true},
			},
		},
	})
	hcBad := &HealthChecker{Client: client, Timeout: 200 * time.Millisecond, Groups: bad, Log: slog.Default()}
	for i := 0; i < 5; i++ {
		hcBad.ProbeOnce(ctx)
	}
	if bad[0].IsHealthy("down") {
		t.Fatal("expected ejected after consecutive fails")
	}
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
