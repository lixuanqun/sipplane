package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/dataplane"
	"github.com/sipplane/sipplane/internal/location"
	"github.com/sipplane/sipplane/internal/otelx"
	"github.com/sipplane/sipplane/internal/policy"
	"github.com/sipplane/sipplane/internal/resources"
)

func main() {
	bootstrap := flag.String("config", "examples/config/bootstrap.yaml", "data-plane bootstrap YAML")
	resourcesDir := flag.String("resources", "", "override resources directory (default from bootstrap)")
	listen := flag.String("listen", "", "override SIP listen address")
	advertisedHost := flag.String("advertised-host", "", "override advertised_host (RFC 0004)")
	flag.Parse()

	cfg, err := config.LoadFile(*bootstrap)
	if err != nil && !os.IsNotExist(err) {
		slog.Error("load bootstrap", "err", err)
		os.Exit(1)
	}
	if os.IsNotExist(err) {
		cfg = config.Default()
	}
	if *resourcesDir != "" {
		cfg.ConfigDir = *resourcesDir
	}
	if *listen != "" {
		cfg.Listen = *listen
	}
	if *advertisedHost != "" {
		cfg.AdvertisedHost = *advertisedHost
	}
	cfg.ApplyFromEnv()

	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	if err := cfg.Validate(); err != nil {
		log.Error("invalid config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownOTel, err := otelx.Setup(ctx, "sipplane", cfg.OTelEndpoint, log)
	if err != nil {
		log.Error("otel setup", "err", err)
		os.Exit(1)
	}
	defer func() { _ = shutdownOTel(context.Background()) }()

	snap, err := resources.LoadDir(cfg.ConfigDir)
	if err != nil {
		log.Error("load resources", "err", err, "dir", cfg.ConfigDir)
		os.Exit(1)
	}
	log.Info("resources loaded", "revision", snap.Revision, "routes", len(snap.Routes), "endpoints", len(snap.Endpoints))

	var opts dataplane.Options
	var rdb *redis.Client
	if cfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		opts.Store = location.NewRedisStore(rdb)
		log.Info("using redis location store", "addr", cfg.RedisAddr)
	}
	if ch := policy.Build(cfg, rdb); ch != nil {
		opts.Policies = ch
		log.Info("ingress policies enabled", "rate_backend", ch.Backend, "rate_key", ch.KeyMode)
	}

	srv, err := dataplane.New(cfg, snap, log, opts)
	if err != nil {
		log.Error("init dataplane", "err", err)
		os.Exit(1)
	}

	if err := srv.Run(ctx); err != nil {
		log.Error("dataplane stopped", "err", err)
		os.Exit(1)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
