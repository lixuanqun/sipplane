package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sipplane/sipplane/internal/controlplane/api"
	"github.com/sipplane/sipplane/internal/controlplane/store"
	"github.com/sipplane/sipplane/internal/resources"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:8090", "control-plane HTTP listen")
	seed := flag.String("seed", "", "optional YAML file/dir to seed store (apply once if revision=0)")
	dbURL := flag.String("database-url", envOr("SIPPLANE_DATABASE_URL", ""), "Postgres DSN; empty = memory store")
	authToken := flag.String("auth-token", envOr("SIPPLANE_CONTROL_TOKEN", ""), "Bearer token for /v1/* (empty = open; required in production)")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var st store.Store
	var closer func()

	if *dbURL != "" {
		pg, err := store.OpenPostgres(ctx, *dbURL)
		if err != nil {
			log.Error("postgres open failed", "err", err)
			os.Exit(1)
		}
		st = pg
		closer = pg.Close
		log.Info("using postgres config store")
	} else {
		var initial *resources.Snapshot
		if *seed != "" {
			snap, err := resources.LoadDir(*seed)
			if err != nil {
				log.Error("seed load failed", "err", err)
				os.Exit(1)
			}
			initial = snap
			log.Info("seeded memory store", "revision", snap.Revision, "routes", len(snap.Routes))
		}
		st = store.NewMemory(initial)
		log.Info("using memory config store")
	}

	if *dbURL != "" && *seed != "" {
		rev, err := st.Revision(ctx)
		if err != nil {
			log.Error("revision", "err", err)
			os.Exit(1)
		}
		if rev == 0 {
			data, err := readSeedYAML(*seed)
			if err != nil {
				log.Error("read seed", "err", err)
				os.Exit(1)
			}
			newRev, err := st.Apply(ctx, "seed", data, false)
			if err != nil {
				log.Error("seed apply", "err", err)
				os.Exit(1)
			}
			log.Info("seeded postgres", "revision", newRev)
		}
	}

	srv := &api.Server{Store: st, Actor: "sipplane-control", AuthToken: *authToken}
	httpSrv := &http.Server{Addr: *listen, Handler: srv.Handler()}

	go func() {
		if *authToken != "" {
			log.Info("control plane listening", "addr", *listen, "auth", "bearer")
		} else {
			log.Warn("control plane listening without auth token (lab only)", "addr", *listen)
		}
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	_ = httpSrv.Shutdown(context.Background())
	if closer != nil {
		closer()
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
