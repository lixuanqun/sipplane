package accesslog

import (
	"log/slog"
	"time"
)

// Event is one SIP transaction completion log line.
type Event struct {
	CallID     string
	Method     string
	Code       int
	DurationMS int64
	Tenant     string
	Route      string
	Trunk      string
	Src        string
	Dst        string
	Revision   int64
}

// Logger writes structured access logs.
type Logger struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Logger {
	if log == nil {
		log = slog.Default()
	}
	return &Logger{log: log}
}

func (l *Logger) Log(ev Event) {
	l.log.Info("access",
		"call_id", ev.CallID,
		"method", ev.Method,
		"code", ev.Code,
		"duration_ms", ev.DurationMS,
		"tenant", ev.Tenant,
		"route", ev.Route,
		"trunk", ev.Trunk,
		"src", ev.Src,
		"dst", ev.Dst,
		"revision", ev.Revision,
	)
}

// Start returns a function that logs with elapsed duration.
func (l *Logger) Start(callID, method, src string) func(code int, dst, tenant, route, trunk string, revision int64) {
	start := time.Now()
	return func(code int, dst, tenant, route, trunk string, revision int64) {
		l.Log(Event{
			CallID:     callID,
			Method:     method,
			Code:       code,
			DurationMS: time.Since(start).Milliseconds(),
			Tenant:     tenant,
			Route:      route,
			Trunk:      trunk,
			Src:        src,
			Dst:        dst,
			Revision:   revision,
		})
	}
}
