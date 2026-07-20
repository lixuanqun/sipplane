package registrar

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/auth"
	"github.com/sipplane/sipplane/internal/location"
	"github.com/sipplane/sipplane/internal/metrics"
	"github.com/sipplane/sipplane/internal/nat"
	"github.com/sipplane/sipplane/internal/outbound"
	"github.com/sipplane/sipplane/internal/routing"
)

// Registrar handles REGISTER requests.
type Registrar struct {
	Store           location.Store
	Auth            *auth.Challenger
	Engine          *routing.Engine
	Log             *slog.Logger
	RequireAuth     bool
	EnablePath      bool
	EnableOutbound  bool
	AdvertisedHost  string
	AdvertisedPort  int
	AdvertisedURI   string // sip:host:port;lr for Path header (legacy simple Path)
	OutboundSecret  []byte
}

func (r *Registrar) Handle(req *sip.Request, tx sip.ServerTransaction) {
	log := r.Log
	if log == nil {
		log = slog.Default()
	}

	to := req.To()
	if to == nil {
		respond(tx, req, 400, "Missing To")
		return
	}
	aor := normalizeAOR(to.Address)

	cont := req.Contact()
	if cont == nil {
		respond(tx, req, 400, "Missing Contact")
		return
	}

	snap := r.Engine.Snapshot()
	ep := snap.FindEndpointByAOR(aor)
	if ep == nil {
		// Also try user@domain from To
		ep = snap.FindEndpointByUsername(to.Address.User, "")
	}
	if ep == nil {
		respond(tx, req, 404, "Endpoint not found")
		metrics.ObserveRequest("REGISTER", "", "", "", 404)
		return
	}
	if !ep.Spec.Allow.CanRegister() {
		respond(tx, req, 403, "Register not allowed")
		metrics.ObserveRequest("REGISTER", ep.Metadata.Tenant, "", "", 403)
		return
	}

	if r.RequireAuth {
		password, ok := snap.ResolvePassword(ep.Spec.Auth.PasswordSecretRef)
		if !ok || password == "" {
			password = ep.Spec.Auth.Password
		}
		if password == "" {
			respond(tx, req, 500, "Endpoint credentials misconfigured")
			return
		}
		authHdr := headerValue(req, "Authorization")
		if authHdr == "" {
			res := sip.NewResponseFromRequest(req, 401, "Unauthorized", nil)
			res.AppendHeader(sip.NewHeader("WWW-Authenticate", r.Auth.ChallengeHeader()))
			_ = tx.Respond(res)
			metrics.ObserveRequest("REGISTER", ep.Metadata.Tenant, "", "", 401)
			return
		}
		uri := req.Recipient.Addr()
		if err := r.Auth.Verify(authHdr, ep.Spec.Auth.Username, password, "REGISTER", uri); err != nil {
			log.Debug("digest verify failed", "err", err, "user", ep.Spec.Auth.Username)
			res := sip.NewResponseFromRequest(req, 401, "Unauthorized", nil)
			res.AppendHeader(sip.NewHeader("WWW-Authenticate", r.Auth.ChallengeHeader()))
			_ = tx.Respond(res)
			metrics.ObserveRequest("REGISTER", ep.Metadata.Tenant, "", "", 401)
			return
		}
	}

	expires := extractExpires(req, cont)
	if expires == 0 {
		_ = r.Store.Delete(aor)
		res := sip.NewResponseFromRequest(req, 200, "OK", nil)
		ex := sip.ExpiresHeader(0)
		res.AppendHeader(&ex)
		if c := req.Contact(); c != nil {
			res.AppendHeader(sip.HeaderClone(c))
		}
		_ = tx.Respond(res)
		metrics.RegisterBindings.WithLabelValues(label(ep.Metadata.Tenant)).Set(float64(r.Store.Count()))
		metrics.ObserveRequest("REGISTER", ep.Metadata.Tenant, "", "", 200)
		return
	}

	hostPort, transport, rewritten := nat.FixContact(cont, req)
	flowTok := ""
	wantOutbound := r.EnableOutbound && outbound.SupportsOutbound(req)
	if wantOutbound || r.EnablePath {
		secret := r.OutboundSecret
		if len(secret) == 0 {
			secret = []byte("sipplane-outbound")
		}
		flow := outbound.FromRequest(req, r.AdvertisedHost, r.AdvertisedPort)
		flowTok = flow.Token(secret)
		// Prefer flow remote as dial target when NAT rewrite already applied or outbound on.
		if wantOutbound {
			hostPort = flow.HostPort()
		}
	}
	contacts := []location.Contact{{
		URI:       cont.Address.String(),
		HostPort:  hostPort,
		Transport: transport,
		Raw:       cont.Value(),
		FlowToken: flowTok,
	}}
	if err := r.Store.Put(aor, contacts, time.Duration(expires)*time.Second); err != nil {
		respond(tx, req, 500, "Location store error")
		return
	}

	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	ex := sip.ExpiresHeader(expires)
	res.AppendHeader(&ex)
	res.AppendHeader(sip.HeaderClone(cont))
	if wantOutbound {
		outbound.AddPathWithFlow(res, r.AdvertisedHost, r.AdvertisedPort, flowTok, true)
	} else if r.EnablePath {
		if flowTok != "" {
			outbound.AddPathWithFlow(res, r.AdvertisedHost, r.AdvertisedPort, flowTok, false)
		} else {
			nat.AddPath(res, r.AdvertisedURI)
		}
	}
	_ = tx.Respond(res)

	metrics.RegisterBindings.WithLabelValues(label(ep.Metadata.Tenant)).Set(float64(r.Store.Count()))
	metrics.ObserveRequest("REGISTER", ep.Metadata.Tenant, "", "", 200)
	log.Info("registered", "aor", aor, "contact", hostPort, "expires", expires, "nat_rewrite", rewritten, "outbound", wantOutbound)
}

func normalizeAOR(u sip.Uri) string {
	u.UriParams = nil
	u.Headers = nil
	u.Port = 0
	return strings.ToLower(u.String())
}

func extractExpires(req *sip.Request, cont *sip.ContactHeader) uint32 {
	if h := req.GetHeader("Expires"); h != nil {
		if v, err := strconv.ParseUint(h.Value(), 10, 32); err == nil {
			return uint32(v)
		}
	}
	if cont != nil && cont.Params != nil {
		if v, ok := cont.Params.Get("expires"); ok {
			if n, err := strconv.ParseUint(v, 10, 32); err == nil {
				return uint32(n)
			}
		}
	}
	return 3600
}

func headerValue(req *sip.Request, name string) string {
	h := req.GetHeader(name)
	if h == nil {
		return ""
	}
	return h.Value()
}

func respond(tx sip.ServerTransaction, req *sip.Request, code int, reason string) {
	res := sip.NewResponseFromRequest(req, code, reason, nil)
	_ = tx.Respond(res)
}

func label(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// AORFromRequestURI builds AOR string from request URI for lookups.
func AORFromRequestURI(u sip.Uri) string {
	return normalizeAOR(u)
}

// FormatAOR formats user@host as sip URI AOR.
func FormatAOR(user, host string) string {
	return normalizeAOR(sip.Uri{User: user, Host: host})
}