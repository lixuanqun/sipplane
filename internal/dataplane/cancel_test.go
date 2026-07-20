package dataplane_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/dataplane"
	"github.com/sipplane/sipplane/internal/proxy"
)

func TestCancelProxiedInvite(t *testing.T) {
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
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := dataplane.New(cfg, labSnapshot(), log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Run(ctx) }()
	waitHTTP(t, "http://"+httpAddr+"/readyz")

	bobPort := freeUDPPort(t)
	alicePort := freeUDPPort(t)
	proxyAddr := cfg.Listen

	bobUA, _ := sipgo.NewUA(sipgo.WithUserAgent("bob-cancel"))
	defer bobUA.Close()
	bobSrv, _ := sipgo.NewServer(bobUA)
	bobClient, _ := sipgo.NewClient(bobUA, sipgo.WithClientHostname("127.0.0.1"), sipgo.WithClientPort(bobPort))
	gotCancel := make(chan struct{}, 1)
	bobSrv.OnInvite(func(req *sip.Request, tx sip.ServerTransaction) {
		// Matched CANCEL is delivered on the INVITE server tx, not Server.OnCancel.
		tx.OnCancel(func(*sip.Request) {
			select {
			case gotCancel <- struct{}{}:
			default:
			}
		})
		_ = tx.Respond(sip.NewResponseFromRequest(req, 180, "Ringing", nil))
		<-tx.Done()
	})
	go func() { _ = bobSrv.ListenAndServe(ctx, "udp", "127.0.0.1:"+itoa(bobPort)) }()
	time.Sleep(100 * time.Millisecond)
	registerWith(t, bobClient, "bob", "bob-secret", "127.0.0.1", bobPort, proxyAddr)

	aliceUA, _ := sipgo.NewUA(sipgo.WithUserAgent("alice-cancel"))
	defer aliceUA.Close()
	aliceSrv, _ := sipgo.NewServer(aliceUA)
	aliceClient, _ := sipgo.NewClient(aliceUA, sipgo.WithClientHostname("127.0.0.1"), sipgo.WithClientPort(alicePort))
	go func() { _ = aliceSrv.ListenAndServe(ctx, "udp", "127.0.0.1:"+itoa(alicePort)) }()
	time.Sleep(100 * time.Millisecond)
	registerWith(t, aliceClient, "alice", "alice-secret", "127.0.0.1", alicePort, proxyAddr)

	invite := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})
	from := sip.NewParams()
	from.Add("tag", "cancel-tag")
	invite.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "alice", Host: "acme.example"}, Params: from})
	invite.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: "bob", Host: "acme.example"}})
	invite.SetDestination(proxyAddr)

	txCtx, txCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer txCancel()
	clTx, err := aliceClient.TransactionRequest(txCtx, invite)
	if err != nil {
		t.Fatal(err)
	}
	defer clTx.Terminate()

	saw180 := false
	saw487 := false
	deadline := time.After(4 * time.Second)
loop:
	for {
		select {
		case res, ok := <-clTx.Responses():
			if !ok {
				break loop
			}
			if res.StatusCode == 180 && !saw180 {
				saw180 = true
				// Send CANCEL asynchronously: sipgo client Responses() is unbuffered;
				// a sync write here can deadlock against the inbound 487 delivery.
				go func() {
					cancelReq := proxy.NewCancelRequest(invite)
					cancelReq.SetDestination(proxyAddr)
					_ = aliceClient.WriteRequest(cancelReq, func(*sipgo.Client, *sip.Request) error { return nil })
				}()
			}
			if res.StatusCode == 487 {
				saw487 = true
				break loop
			}
			if res.StatusCode >= 200 {
				t.Fatalf("unexpected final response %d", res.StatusCode)
			}
		case <-deadline:
			break loop
		}
	}
	if !saw180 {
		t.Fatal("expected 180 before cancel")
	}
	if !saw487 {
		t.Fatal("expected 487 Request Terminated after CANCEL")
	}
	select {
	case <-gotCancel:
	case <-time.After(3 * time.Second):
		t.Fatal("bob did not observe CANCEL on INVITE transaction")
	}
}
