package dataplane

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sipplane/sipplane/internal/accesslog"
	"github.com/sipplane/sipplane/internal/auth"
	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/controlplane/watcher"
	"github.com/sipplane/sipplane/internal/discovery"
	"github.com/sipplane/sipplane/internal/hep"
	"github.com/sipplane/sipplane/internal/location"
	"github.com/sipplane/sipplane/internal/metrics"
	"github.com/sipplane/sipplane/internal/policy"
	"github.com/sipplane/sipplane/internal/proxy"
	"github.com/sipplane/sipplane/internal/redirect"
	"github.com/sipplane/sipplane/internal/registrar"
	"github.com/sipplane/sipplane/internal/resources"
	"github.com/sipplane/sipplane/internal/routing"
)

// Server is the SIP data-plane process.
type Server struct {
	Cfg      config.Config
	Log      *slog.Logger
	Store    location.Store
	Engine   *routing.Engine
	Policies *policy.Chain
	ready    atomic.Bool
	revision atomic.Int64
	ua       *sipgo.UserAgent
	sipSrv   *sipgo.Server
	client   *sipgo.Client
	httpSrv  *http.Server
}

// Options customizes dataplane construction.
type Options struct {
	Store    location.Store
	Policies *policy.Chain
}

// New creates a data-plane server from config and resource snapshot.
func New(cfg config.Config, snap *resources.Snapshot, log *slog.Logger, opts ...Options) (*Server, error) {
	if log == nil {
		log = slog.Default()
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	store := opt.Store
	if store == nil {
		store = location.NewMemoryStore()
	}
	eng := routing.NewEngine(snap)

	host := cfg.AdvertisedHost
	if host == "" {
		host = "127.0.0.1"
	}
	ua, err := sipgo.NewUA(sipgo.WithUserAgent("sipplane"))
	if err != nil {
		return nil, err
	}
	sipSrv, err := sipgo.NewServer(ua)
	if err != nil {
		_ = ua.Close()
		return nil, err
	}
	client, err := sipgo.NewClient(ua,
		sipgo.WithClientHostname(host),
		sipgo.WithClientPort(cfg.AdvertisedPort),
	)
	if err != nil {
		_ = ua.Close()
		return nil, err
	}

	s := &Server{
		Cfg:      cfg,
		Log:      log,
		Store:    store,
		Engine:   eng,
		Policies: opt.Policies,
		ua:       ua,
		sipSrv:   sipSrv,
		client:   client,
	}
	if snap != nil {
		s.revision.Store(snap.Revision)
		metrics.ConfigRevision.Set(float64(snap.Revision))
	}
	s.ready.Store(true)

	access := accesslog.New(log)
	var hepExp *hep.Exporter
	if cfg.HEPAddr != "" {
		hepExp = hep.NewExporter(cfg.HEPAddr, cfg.HEPCaptureID, log)
	}
	reg := &registrar.Registrar{
		Store:          store,
		Auth:           auth.NewChallenger(cfg.Realm),
		Engine:         eng,
		Log:            log,
		RequireAuth:    true,
		EnablePath:     cfg.EnablePath,
		EnableOutbound: cfg.EnableOutbound,
		AdvertisedHost: cfg.AdvertisedHost,
		AdvertisedPort: cfg.AdvertisedPort,
		AdvertisedURI:  cfg.AdvertisedSIPURI(),
		OutboundSecret: []byte(cfg.OutboundSecret),
	}
	px := &proxy.Proxy{
		Client:         client,
		Engine:         eng,
		Store:          store,
		Access:         access,
		Log:            log,
		HEP:            hepExp,
		RedirectPolicy: redirect.NormalizePolicy(cfg.RedirectPolicy),
	}

	wrap := func(next func(req *sip.Request, tx sip.ServerTransaction)) func(req *sip.Request, tx sip.ServerTransaction) {
		return func(req *sip.Request, tx sip.ServerTransaction) {
			if s.Policies != nil {
				if res := s.Policies.Ingress(req); res.Decision == policy.Deny {
					r := sip.NewResponseFromRequest(req, res.Code, res.Reason, nil)
					_ = tx.Respond(r)
					metrics.ObserveRequest(req.Method.String(), "", "", "", res.Code)
					return
				}
			}
			next(req, tx)
		}
	}

	sipSrv.OnRegister(wrap(reg.Handle))
	sipSrv.OnInvite(wrap(px.HandleInvite))
	sipSrv.OnAck(px.HandleAck)
	sipSrv.OnCancel(wrap(px.HandleCancel))
	sipSrv.OnBye(wrap(px.HandleBye))
	sipSrv.OnOptions(wrap(px.HandleOptions))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !s.ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	s.httpSrv = &http.Server{Addr: cfg.HTTPListen, Handler: mux}

	return s, nil
}

// ReplaceSnapshot atomically swaps routing config.
func (s *Server) ReplaceSnapshot(snap *resources.Snapshot) {
	s.Engine.ReplaceSnapshot(snap)
	s.revision.Store(snap.Revision)
	metrics.ConfigRevision.Set(float64(snap.Revision))
}

// Run starts HTTP and SIP listeners until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 4)

	if s.Cfg.ControlURL != "" {
		stale := 60 * time.Second
		if s.Cfg.ConfigStaleAfter != "" {
			if d, err := time.ParseDuration(s.Cfg.ConfigStaleAfter); err == nil {
				stale = d
			}
		}
		w := &watcher.Watcher{
			BaseURL:    s.Cfg.ControlURL,
			Token:      s.Cfg.ControlToken,
			Applier:    s,
			Log:        s.Log,
			StaleAfter: stale,
		}
		go func() {
			if err := w.Run(ctx); err != nil {
				s.Log.Error("config watcher stopped", "err", err)
			}
		}()
	}

	if snap := s.Engine.Snapshot(); snap != nil {
		if groups := discovery.GroupsFromPingTrunks(snap.Trunks); len(groups) > 0 {
			hc := &discovery.HealthChecker{
				Client:   s.client,
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				Groups:   groups,
				Log:      s.Log,
			}
			go hc.Run(ctx)
			s.Log.Info("options health checker started", "groups", len(groups))
		}
	}

	go func() {
		s.Log.Info("http listening", "addr", s.Cfg.HTTPListen)
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	for _, tr := range s.Cfg.Transports() {
		tr := tr
		go func() {
			s.Log.Info("sip listening", "transport", tr, "addr", s.Cfg.Listen,
				"advertised_host", s.Cfg.AdvertisedHost, "advertised_port", s.Cfg.AdvertisedPort)
			var err error
			if tr == "tls" || tr == "wss" {
				if s.Cfg.TLSCertFile == "" || s.Cfg.TLSKeyFile == "" {
					errCh <- fmt.Errorf("tls transport requires tls_cert_file and tls_key_file")
					return
				}
				cert, loadErr := tls.LoadX509KeyPair(s.Cfg.TLSCertFile, s.Cfg.TLSKeyFile)
				if loadErr != nil {
					errCh <- fmt.Errorf("load tls cert: %w", loadErr)
					return
				}
				err = s.sipSrv.ListenAndServeTLS(ctx, tr, s.Cfg.Listen, &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				})
			} else {
				err = s.sipSrv.ListenAndServe(ctx, tr, s.Cfg.Listen)
			}
			if err != nil {
				errCh <- fmt.Errorf("sip %s: %w", tr, err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
		_ = s.ua.Close()
		return nil
	case err := <-errCh:
		_ = s.ua.Close()
		return err
	}
}

// SetReady marks readiness (for tests / watcher).
func (s *Server) SetReady(v bool) { s.ready.Store(v) }

// SIPServer exposes the sipgo server for tests.
func (s *Server) SIPServer() *sipgo.Server { return s.sipSrv }

// Client exposes the sipgo client for tests.
func (s *Server) Client() *sipgo.Client { return s.client }

// UserAgent exposes UA for tests.
func (s *Server) UserAgent() *sipgo.UserAgent { return s.ua }
