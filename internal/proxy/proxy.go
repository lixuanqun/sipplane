package proxy

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/accesslog"
	"github.com/sipplane/sipplane/internal/hep"
	"github.com/sipplane/sipplane/internal/location"
	"github.com/sipplane/sipplane/internal/metrics"
	"github.com/sipplane/sipplane/internal/redirect"
	"github.com/sipplane/sipplane/internal/routing"
	"github.com/sipplane/sipplane/internal/webhook"
)

// Proxy is a stateful SIP proxy (INVITE/ACK/BYE/CANCEL/OPTIONS).
type Proxy struct {
	Client    *sipgo.Client
	Engine    *routing.Engine
	Store     location.Store
	Access    *accesslog.Logger
	Log       *slog.Logger
	HEP       *hep.Exporter
	MissCode       int // default 480
	ErrorCode      int // location error default 503
	RedirectPolicy redirect.Policy
}

func (p *Proxy) log() *slog.Logger {
	if p.Log == nil {
		return slog.Default()
	}
	return p.Log
}

func (p *Proxy) missCode() int {
	if p.MissCode == 0 {
		return 480
	}
	return p.MissCode
}

func (p *Proxy) errorCode() int {
	if p.ErrorCode == 0 {
		return 503
	}
	return p.ErrorCode
}

// HandleInvite routes INVITE (and CANCEL/BYE via same relay path).
func (p *Proxy) HandleInvite(req *sip.Request, tx sip.ServerTransaction) {
	p.relay(req, tx, true)
}

// HandleCancel relays CANCEL.
func (p *Proxy) HandleCancel(req *sip.Request, tx sip.ServerTransaction) {
	p.relay(req, tx, true)
}

// HandleBye relays BYE.
func (p *Proxy) HandleBye(req *sip.Request, tx sip.ServerTransaction) {
	p.relay(req, tx, false)
}

// HandleAck forwards ACK hop-by-hop without transaction.
func (p *Proxy) HandleAck(req *sip.Request, tx sip.ServerTransaction) {
	dst, routeName, trunkName, tenant, code := p.resolveDestination(req)
	_ = code
	if dst == "" {
		return
	}
	req.SetDestination(dst)
	if err := p.Client.WriteRequest(req, sipgo.ClientRequestAddVia); err != nil {
		p.log().Error("ack write failed", "err", err)
	}
	_ = routeName
	_ = trunkName
	_ = tenant
}

// HandleOptions answers locally or proxies based on routes.
func (p *Proxy) HandleOptions(req *sip.Request, tx sip.ServerTransaction) {
	decision, ok := p.Engine.Match(req)
	if ok && decision.Action.Type == "proxy" && decision.DestAddr != "" {
		p.relay(req, tx, false)
		return
	}
	// Local keepalive / health
	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	res.AppendHeader(sip.NewHeader("Allow", "INVITE, ACK, CANCEL, BYE, OPTIONS, REGISTER"))
	_ = tx.Respond(res)
	metrics.ObserveRequest("OPTIONS", "", "", "", 200)
}

func (p *Proxy) relay(req *sip.Request, tx sip.ServerTransaction, isInviteFamily bool) {
	metrics.SIPTransactionsInflight.WithLabelValues(req.Method.String()).Inc()
	defer metrics.SIPTransactionsInflight.WithLabelValues(req.Method.String()).Dec()

	callID := ""
	if c := req.CallID(); c != nil {
		callID = c.Value()
	}
	doneLog := p.Access.Start(callID, req.Method.String(), req.Source())

	dst, routeName, trunkName, tenant, earlyCode := p.resolveDestination(req)
	revision := int64(0)
	if snap := p.Engine.Snapshot(); snap != nil {
		revision = snap.Revision
	}

	if earlyCode != 0 {
		reply(tx, req, earlyCode, reasonFor(earlyCode))
		doneLog(earlyCode, dst, tenant, routeName, trunkName, revision)
		metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, earlyCode)
		return
	}
	if dst == "" {
		code := p.missCode()
		reply(tx, req, code, reasonFor(code))
		doneLog(code, "", tenant, routeName, trunkName, revision)
		metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, code)
		return
	}

	ctx := context.Background()
	// Clone before ClientRequestAddVia — mutating the server-tx origin corrupts
	// local CANCEL/487 Via matching back to the UAC (RFC 3261 §16.6 / §9.1).
	outReq := req.Clone()
	outReq.SetDestination(dst)
	if p.HEP != nil {
		p.HEP.Send([]byte(outReq.String()), req.Source(), dst, strings.EqualFold(req.Transport(), "UDP"))
	}
	clTx, err := p.Client.TransactionRequest(ctx, outReq, sipgo.ClientRequestAddVia, sipgo.ClientRequestAddRecordRoute)
	if err != nil {
		p.log().Error("transaction request failed", "err", err)
		reply(tx, req, 500, "Server Internal Error")
		doneLog(500, dst, tenant, routeName, trunkName, revision)
		metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, 500)
		return
	}
	defer clTx.Terminate()

	// Forward CANCEL immediately. Server tx Done() only closes after Timer H (~32s).
	if isInviteFamily && req.IsInvite() {
		tx.OnCancel(func(*sip.Request) {
			cancelReq := newCancelRequest(outReq)
			if err := p.Client.WriteRequest(cancelReq, func(*sipgo.Client, *sip.Request) error { return nil }); err != nil {
				p.log().Error("cancel write failed", "err", err)
			}
		})
	}

	finalCode := 0
	for {
		select {
		case res, more := <-clTx.Responses():
			if !more {
				if finalCode == 0 {
					finalCode = 408
				}
				doneLog(finalCode, dst, tenant, routeName, trunkName, revision)
				metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, finalCode)
				return
			}
			res.SetDestination(req.Source())
			res.RemoveHeader("Via")
			if !res.IsProvisional() {
				finalCode = res.StatusCode
			}
			// Optional 302 follow (BACKLOG B4)
			if redirect.ShouldFollow(res, p.RedirectPolicy) {
				if hp, ok := redirect.ContactHostPort(res); ok {
					outReq.SetDestination(hp)
					dst = hp
					clTx.Terminate()
					clTx, err = p.Client.TransactionRequest(ctx, outReq, sipgo.ClientRequestAddVia, sipgo.ClientRequestAddRecordRoute)
					if err != nil {
						reply(tx, req, 500, "Redirect failed")
						doneLog(500, dst, tenant, routeName, trunkName, revision)
						metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, 500)
						return
					}
					continue
				}
			}
			if p.HEP != nil {
				p.HEP.Send([]byte(res.String()), dst, req.Source(), strings.EqualFold(req.Transport(), "UDP"))
			}
			if err := tx.Respond(res); err != nil {
				p.log().Error("respond failed", "err", err)
			}

		case <-clTx.Done():
			if finalCode == 0 {
				finalCode = 408
			}
			doneLog(finalCode, dst, tenant, routeName, trunkName, revision)
			metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, finalCode)
			return

		case ack := <-tx.Acks():
			ack.SetDestination(dst)
			_ = p.Client.WriteRequest(ack)

		case <-tx.Done():
			if finalCode == 0 {
				if errors.Is(tx.Err(), sip.ErrTransactionCanceled) {
					finalCode = 487
				} else {
					finalCode = 408
				}
			}
			doneLog(finalCode, dst, tenant, routeName, trunkName, revision)
			metrics.ObserveRequest(req.Method.String(), tenant, routeName, trunkName, finalCode)
			return
		}
	}
}

func (p *Proxy) resolveDestination(req *sip.Request) (dst, routeName, trunkName, tenant string, errCode int) {
	decision, ok := p.Engine.Match(req)
	if !ok {
		// Default: registerLookup by Request-URI AOR
		return p.lookupContact(req)
	}
	routeName = decision.Route.Metadata.Name
	tenant = decision.Route.Metadata.Tenant
	if decision.Trunk != nil {
		trunkName = decision.Trunk.Metadata.Name
	}

	switch decision.Action.Type {
	case "reject":
		code := decision.Action.Code
		if code == 0 {
			code = 403
		}
		return "", routeName, trunkName, tenant, code
	case "proxy", "loadBalance":
		return decision.DestAddr, routeName, trunkName, tenant, 0
	case "registerLookup":
		dst, _, _, _, code := p.lookupContact(req)
		return dst, routeName, trunkName, tenant, code
	case "webhook":
		url := webhook.URLFromAction(decision.Action)
		fb := webhook.Response{Action: "reject", Code: 503, Reason: "Webhook Unavailable"}
		if decision.Action.Code > 0 {
			fb.Code = decision.Action.Code
		}
		cli := webhook.New(url, 500*time.Millisecond, fb)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		out := cli.Decide(ctx, req, decision.Route)
		switch out.Action {
		case "continue", "registerLookup":
			dst, _, _, _, code := p.lookupContact(req)
			return dst, routeName, trunkName, tenant, code
		case "proxy":
			return out.Target, routeName, trunkName, tenant, 0
		case "reject":
			code := out.Code
			if code == 0 {
				code = 403
			}
			return "", routeName, trunkName, tenant, code
		default:
			return "", routeName, trunkName, tenant, fb.Code
		}
	default:
		return "", routeName, trunkName, tenant, 500
	}
}

func (p *Proxy) lookupContact(req *sip.Request) (dst, routeName, trunkName, tenant string, errCode int) {
	aor := normalizeAOR(req.Recipient)
	contacts, err := p.Store.Get(aor)
	if err != nil {
		if errors.Is(err, location.ErrNotFound) {
			metrics.LocationLookups.WithLabelValues("miss").Inc()
			return "", "", "", "", p.missCode()
		}
		metrics.LocationLookups.WithLabelValues("error").Inc()
		return "", "", "", "", p.errorCode()
	}
	metrics.LocationLookups.WithLabelValues("hit_local").Inc()
	if len(contacts) == 0 {
		return "", "", "", "", p.missCode()
	}
	return contacts[0].HostPort, "", "", "", 0
}

func normalizeAOR(u sip.Uri) string {
	u.UriParams = nil
	u.Headers = nil
	u.Port = 0
	return strings.ToLower(u.String())
}

func reply(tx sip.ServerTransaction, req *sip.Request, code int, reason string) {
	res := sip.NewResponseFromRequest(req, code, reason, nil)
	_ = tx.Respond(res)
}

func reasonFor(code int) string {
	switch code {
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 480:
		return "Temporarily Unavailable"
	case 487:
		return "Request Terminated"
	case 503:
		return "Service Unavailable"
	default:
		return "Error"
	}
}

func newCancelRequest(inviteRequest *sip.Request) *sip.Request {
	// Mirror sipgo/sip.newCancelRequest: same top Via + CSeq seq as INVITE (RFC 3261 §9.1).
	cancelReq := sip.NewRequest(sip.CANCEL, inviteRequest.Recipient)
	cancelReq.SipVersion = inviteRequest.SipVersion
	if via := inviteRequest.Via(); via != nil {
		cancelReq.AppendHeader(sip.HeaderClone(via))
	}
	sip.CopyHeaders("Route", inviteRequest, cancelReq)
	maxFwd := sip.MaxForwardsHeader(70)
	cancelReq.AppendHeader(&maxFwd)
	if h := inviteRequest.From(); h != nil {
		cancelReq.AppendHeader(sip.HeaderClone(h))
	}
	if h := inviteRequest.To(); h != nil {
		cancelReq.AppendHeader(sip.HeaderClone(h))
	}
	if h := inviteRequest.CallID(); h != nil {
		cancelReq.AppendHeader(sip.HeaderClone(h))
	}
	if h := inviteRequest.CSeq(); h != nil {
		cancelReq.AppendHeader(sip.HeaderClone(h))
		if cseq := cancelReq.CSeq(); cseq != nil {
			cseq.MethodName = sip.CANCEL
		}
	}
	cancelReq.SetTransport(inviteRequest.Transport())
	cancelReq.SetSource(inviteRequest.Source())
	cancelReq.SetDestination(inviteRequest.Destination())
	return cancelReq
}

// ResolveDestination exposes routing resolution for unit tests.
func (p *Proxy) ResolveDestination(req *sip.Request) (dst, routeName, trunkName, tenant string, errCode int) {
	return p.resolveDestination(req)
}

// NewCancelRequest exposes CANCEL builder for unit tests.
func NewCancelRequest(invite *sip.Request) *sip.Request {
	return newCancelRequest(invite)
}
