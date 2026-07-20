package registrar_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/emiago/sipgo/sip"
	"github.com/icholy/digest"
	"github.com/sipplane/sipplane/internal/auth"
	"github.com/sipplane/sipplane/internal/location"
	"github.com/sipplane/sipplane/internal/registrar"
	"github.com/sipplane/sipplane/internal/resources"
	"github.com/sipplane/sipplane/internal/routing"
)

type fakeTx struct {
	resps []*sip.Response
}

func (f *fakeTx) Terminate() {}
func (f *fakeTx) OnTerminate(sip.FnTxTerminate) bool { return false }
func (f *fakeTx) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (f *fakeTx) Err() error                 { return nil }
func (f *fakeTx) Respond(res *sip.Response) error {
	f.resps = append(f.resps, res.Clone())
	return nil
}
func (f *fakeTx) Acks() <-chan *sip.Request { return nil }
func (f *fakeTx) OnCancel(sip.FnTxCancel) bool { return false }

func TestRegistrarDigestAndOutboundPath(t *testing.T) {
	store := location.NewMemoryStore()
	snap := &resources.Snapshot{
		Revision: 1,
		Secrets:  map[string]string{"inline/acme/alice": "alice-secret"},
		Endpoints: []*resources.Endpoint{{
			Metadata: resources.ObjectMeta{Name: "alice", Tenant: "acme"},
			Spec: resources.EndpointSpec{
				AORs: []string{"sip:alice@acme.example"},
				Auth: resources.EndpointAuth{Username: "alice", PasswordSecretRef: "inline/acme/alice"},
			},
		}},
		Tenants: map[string]*resources.Tenant{},
		Trunks:  map[string]*resources.Trunk{},
	}
	reg := &registrar.Registrar{
		Store:          store,
		Auth:           auth.NewChallenger("sipplane"),
		Engine:         routing.NewEngine(snap),
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		RequireAuth:    true,
		EnablePath:     true,
		EnableOutbound: true,
		AdvertisedHost: "sip.example.com",
		AdvertisedPort: 5060,
		OutboundSecret: []byte("test-secret"),
	}

	req := sip.NewRequest(sip.REGISTER, sip.Uri{Host: "acme.example"})
	params := sip.NewParams()
	params.Add("tag", "t1")
	req.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "alice", Host: "acme.example"}, Params: params})
	req.AppendHeader(&sip.ToHeader{Address: sip.Uri{User: "alice", Host: "acme.example"}})
	req.AppendHeader(&sip.ContactHeader{Address: sip.Uri{User: "alice", Host: "192.168.1.9", Port: 5060}})
	ex := sip.ExpiresHeader(3600)
	req.AppendHeader(&ex)
	req.AppendHeader(sip.NewHeader("Supported", "outbound"))
	req.SetSource("203.0.113.20:45000")
	req.SetTransport("UDP")

	tx := &fakeTx{}
	reg.Handle(req, tx)
	if len(tx.resps) != 1 || tx.resps[0].StatusCode != 401 {
		t.Fatalf("want 401, got %+v", tx.resps)
	}
	chalHdr := tx.resps[0].GetHeader("WWW-Authenticate")
	chal, err := digest.ParseChallenge(chalHdr.Value())
	if err != nil {
		t.Fatal(err)
	}
	cred, err := digest.Digest(chal, digest.Options{
		Method: "REGISTER", URI: req.Recipient.Addr(), Username: "alice", Password: "alice-secret", Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	req2 := req.Clone()
	req2.AppendHeader(sip.NewHeader("Authorization", cred.String()))
	tx2 := &fakeTx{}
	reg.Handle(req2, tx2)
	if len(tx2.resps) != 1 || tx2.resps[0].StatusCode != 200 {
		t.Fatalf("want 200, got %+v", tx2.resps)
	}
	path := tx2.resps[0].GetHeader("Path")
	if path == nil || path.Value() == "" {
		t.Fatal("expected Path header for outbound")
	}
	if !contains(path.Value(), "ob") {
		t.Fatalf("Path missing ob: %s", path.Value())
	}

	contacts, err := store.Get("sip:alice@acme.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 || contacts[0].FlowToken == "" {
		t.Fatalf("expected flow token binding: %+v", contacts)
	}
	if contacts[0].HostPort != "203.0.113.20:45000" {
		t.Fatalf("hostport=%s", contacts[0].HostPort)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
