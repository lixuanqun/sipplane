package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/resources"
)

func TestDecideContinue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Response{Action: "continue"})
	}))
	defer srv.Close()

	c := New(srv.URL, time.Second, Response{Action: "reject", Code: 503})
	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})
	req.SetSource("1.2.3.4:5060")
	out := c.Decide(context.Background(), req, &resources.Route{Metadata: resources.ObjectMeta{Name: "r1"}})
	if out.Action != "continue" {
		t.Fatalf("%+v", out)
	}
}

func TestDecideTimeoutFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(Response{Action: "continue"})
	}))
	defer srv.Close()

	c := New(srv.URL, 20*time.Millisecond, Response{Action: "reject", Code: 503, Reason: "timeout"})
	req := sip.NewRequest(sip.INVITE, sip.Uri{User: "bob", Host: "acme.example"})
	out := c.Decide(context.Background(), req, nil)
	if out.Action != "reject" || out.Code != 503 {
		t.Fatalf("want fallback reject, got %+v", out)
	}
}
